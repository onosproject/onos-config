// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package device

import (
	"context"
	"errors"
	"fmt"
	"github.com/atomix/atomix-go-client/pkg/client/counter"
	"github.com/atomix/atomix-go-client/pkg/client/map"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/store/utils"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"sync"
	"time"
)

const primitiveName = "device-changes"

// getCounterName returns the name of the given device ID counter
func getCounterName(deviceID device.ID) string {
	return fmt.Sprintf("device-change-index-%s", deviceID)
}

// NewAtomixStore returns a new persistent Store
func NewAtomixStore() (Store, error) {
	client, err := utils.GetAtomixClient()
	if err != nil {
		return nil, err
	}

	group, err := client.GetGroup(context.Background(), utils.GetAtomixRaftGroup())
	if err != nil {
		return nil, err
	}

	configs, err := group.GetMap(context.Background(), primitiveName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	indexFactory := func(deviceID device.ID) (counter.Counter, error) {
		return group.GetCounter(context.Background(), getCounterName(deviceID), session.WithTimeout(30*time.Second))
	}

	return &atomixStore{
		configs:      configs,
		indexFactory: indexFactory,
		indexes:      make(map[device.ID]counter.Counter),
		closer:       configs,
	}, nil
}

// NewLocalStore returns a new local device store
func NewLocalStore() (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(conn)
}

// newLocalStore creates a new local device change store
func newLocalStore(conn *grpc.ClientConn) (Store, error) {
	name := primitive.Name{
		Namespace: "local",
		Name:      primitiveName,
	}

	configs, err := _map.New(context.Background(), name, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	indexFactory := func(deviceID device.ID) (counter.Counter, error) {
		counterName := primitive.Name{
			Namespace: "local",
			Name:      getCounterName(deviceID),
		}
		return counter.New(context.Background(), counterName, []*grpc.ClientConn{conn})
	}

	return &atomixStore{
		configs:      configs,
		indexFactory: indexFactory,
		indexes:      make(map[device.ID]counter.Counter),
	}, nil
}

// startLocalNode starts a single local node
func startLocalNode() (*atomix.Node, *grpc.ClientConn) {
	lis := bufconn.Listen(1024 * 1024)
	node := local.NewNode(lis, registry.Registry)
	_ = node.Start()

	dialer := func(ctx context.Context, address string) (net.Conn, error) {
		return lis.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), primitiveName, grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial network configurations")
	}
	return node, conn
}

// Store stores DeviceChanges
type Store interface {
	io.Closer

	// LastIndex returns the last index for the given device
	LastIndex(device.ID) (devicechange.Index, error)

	// Get gets a device change
	Get(id devicechange.ID) (*devicechange.Change, error)

	// Create creates a new device change
	Create(config *devicechange.Change) error

	// Update updates an existing device change
	Update(config *devicechange.Change) error

	// Delete deletes a device change
	Delete(config *devicechange.Change) error

	// List lists device change
	List(device.ID, chan<- *devicechange.Change) error

	// Replay replays the device changes from the given index
	Replay(device.ID, devicechange.Index, chan<- *devicechange.Change) error

	// Watch watches the device change store for changes
	Watch(chan<- *devicechange.Change) error
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	configs      _map.Map
	indexFactory func(deviceID device.ID) (counter.Counter, error)
	indexes      map[device.ID]counter.Counter
	mu           sync.RWMutex
	closer       io.Closer
}

func (s *atomixStore) getIndexCounter(deviceID device.ID) (counter.Counter, error) {
	s.mu.RLock()
	counter, ok := s.indexes[deviceID]
	s.mu.RUnlock()
	if !ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		counter, ok = s.indexes[deviceID]
		if !ok {
			newCounter, err := s.indexFactory(deviceID)
			if err != nil {
				return nil, err
			}
			s.indexes[deviceID] = newCounter
			return newCounter, nil
		}
	}
	return counter, nil
}

func (s *atomixStore) LastIndex(deviceID device.ID) (devicechange.Index, error) {
	indexes, err := s.getIndexCounter(deviceID)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	index, err := indexes.Get(ctx)
	return devicechange.Index(index), err
}

func (s *atomixStore) nextIndex(deviceID device.ID) (devicechange.Index, error) {
	indexes, err := s.getIndexCounter(deviceID)
	if err != nil {
		return 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	index, err := indexes.Increment(ctx, 1)
	return devicechange.Index(index), err
}

func (s *atomixStore) Get(id devicechange.ID) (*devicechange.Change, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeChange(entry)
}

func (s *atomixStore) Create(config *devicechange.Change) error {
	if config.Revision != 0 {
		return errors.New("not a new object")
	}

	index, err := s.nextIndex(config.DeviceID)
	if err != nil {
		return err
	}

	config.Index = index
	config.ID = index.GetChangeID(config.DeviceID)
	config.Created = time.Now()
	config.Updated = time.Now()

	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfNotSet())
	if err != nil {
		return err
	}

	config.Revision = devicechange.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(config *devicechange.Change) error {
	if config.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	config.Updated = time.Now()
	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfVersion(int64(config.Revision)))
	if err != nil {
		return err
	}

	config.Revision = devicechange.Revision(entry.Version)
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(config *devicechange.Change) error {
	if config.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Remove(ctx, string(config.ID), _map.IfVersion(int64(config.Revision)))
	if err != nil {
		return err
	}

	config.Revision = 0
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(device device.ID, ch chan<- *devicechange.Change) error {
	return s.Replay(device, devicechange.Index(0), ch)
}

func (s *atomixStore) Replay(device device.ID, index devicechange.Index, ch chan<- *devicechange.Change) error {
	lastIndex, err := s.LastIndex(device)
	if err != nil {
		return err
	}

	go func() {
		for i := index; i <= lastIndex; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			entry, err := s.configs.Get(ctx, string(i.GetChangeID(device)))
			cancel()
			if err != nil {
				ch <- nil
				break
			} else if entry != nil {
				change, err := decodeChange(entry)
				if err != nil {
					ch <- nil
					break
				}
				ch <- change
			}
		}
		close(ch)
	}()
	return nil
}

func (s *atomixStore) Watch(ch chan<- *devicechange.Change) error {
	mapCh := make(chan *_map.Event)
	if err := s.configs.Watch(context.Background(), mapCh, _map.WithReplay()); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if config, err := decodeChange(event.Entry); err == nil {
				ch <- config
			}
		}
	}()
	return nil
}

func (s *atomixStore) Close() error {
	return s.configs.Close()
}

func decodeChange(entry *_map.Entry) (*devicechange.Change, error) {
	config := &devicechange.Change{}
	if err := proto.Unmarshal(entry.Value, config); err != nil {
		return nil, err
	}
	config.ID = devicechange.ID(entry.Key)
	config.Revision = devicechange.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return config, nil
}
