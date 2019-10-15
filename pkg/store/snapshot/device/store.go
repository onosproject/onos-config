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
	"github.com/atomix/atomix-go-client/pkg/client/counter"
	"github.com/atomix/atomix-go-client/pkg/client/map"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/cenkalti/backoff"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/store/utils"
	devicesnapshot "github.com/onosproject/onos-config/pkg/types/snapshot/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

const counterName = "device-snapshot-indexes"
const primitiveName = "device-snapshots"

const maxRetries = 5

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

	ids, err := group.GetCounter(context.Background(), counterName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	configs, err := group.GetMap(context.Background(), primitiveName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		configs: configs,
		indexes: ids,
	}, nil
}

// NewLocalStore returns a new local device store
func NewLocalStore() (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(conn)
}

// newLocalStore creates a new local device change store
func newLocalStore(conn *grpc.ClientConn) (Store, error) {
	counterName := primitive.Name{
		Namespace: "local",
		Name:      counterName,
	}
	ids, err := counter.New(context.Background(), counterName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	configsName := primitive.Name{
		Namespace: "local",
		Name:      primitiveName,
	}
	configs, err := _map.New(context.Background(), configsName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		configs: configs,
		indexes: ids,
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

	// LastIndex returns the last index
	LastIndex() (devicesnapshot.Index, error)

	// Get gets a device snapshot
	Get(id devicesnapshot.ID) (*devicesnapshot.DeviceSnapshot, error)

	// GetByIndex gets a device snapshot by index
	GetByIndex(index devicesnapshot.Index) (*devicesnapshot.DeviceSnapshot, error)

	// Create creates a new device snapshot
	Create(config *devicesnapshot.DeviceSnapshot) error

	// Update updates an existing device snapshot
	Update(config *devicesnapshot.DeviceSnapshot) error

	// Delete deletes a device snapshot
	Delete(config *devicesnapshot.DeviceSnapshot) error

	// List lists device snapshot
	List(chan<- *devicesnapshot.DeviceSnapshot) error

	// Watch watches the device snapshot store for changes
	Watch(chan<- *devicesnapshot.DeviceSnapshot) error

	// Store stores a snapshot
	Store(snapshot *devicesnapshot.Snapshot) error

	// Load loads a snapshot
	Load(id devicesnapshot.ID) (*devicesnapshot.Snapshot, error)
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	configs _map.Map
	indexes counter.Counter
}

func (s *atomixStore) LastIndex() (devicesnapshot.Index, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	index, err := s.indexes.Get(ctx)
	if err != nil {
		return 0, err
	}
	return devicesnapshot.Index(index), nil
}

func (s *atomixStore) setIndex(index devicesnapshot.Index) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.indexes.Set(ctx, int64(index))
}

func (s *atomixStore) Get(id devicesnapshot.ID) (*devicesnapshot.DeviceSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeDeviceSnapshot(entry)
}

func (s *atomixStore) GetByIndex(index devicesnapshot.Index) (*devicesnapshot.DeviceSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(index.GetSnapshotID()))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeDeviceSnapshot(entry)
}

func (s *atomixStore) Create(config *devicesnapshot.DeviceSnapshot) error {
	if config.Revision != 0 {
		return errors.New("not a new object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return backoff.Retry(func() error {
		lastIndex, err := s.LastIndex()
		if err != nil {
			return err
		}

		index := lastIndex + 1
		config.Index = index
		config.ID = index.GetSnapshotID()

		bytes, err := proto.Marshal(config)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Attempt to set the snapshot at the next index
		entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfNotSet())

		// If the write fails because a snapshot has already been stored, increment the last index and retry
		if err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if change, err := s.configs.Get(ctx, string(config.ID)); err == nil && change != nil {
				_ = s.setIndex(index)
			}
			return err
		}

		config.Revision = devicesnapshot.Revision(entry.Version)
		config.Created = entry.Created
		config.Updated = entry.Updated

		// Store the updated index without failing the write
		_ = s.setIndex(index)
		return nil
	}, backoff.WithContext(backoff.WithMaxRetries(backoff.NewExponentialBackOff(), maxRetries), ctx))
}

func (s *atomixStore) Update(config *devicesnapshot.DeviceSnapshot) error {
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

	config.Revision = devicesnapshot.Revision(entry.Version)
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(config *devicesnapshot.DeviceSnapshot) error {
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

func (s *atomixStore) List(ch chan<- *devicesnapshot.DeviceSnapshot) error {
	lastIndex, err := s.LastIndex()
	if err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for i := devicesnapshot.Index(1); i <= lastIndex; i++ {
			if device, err := s.GetByIndex(i); err == nil && device != nil {
				ch <- device
			}
		}
	}()
	return nil
}

func (s *atomixStore) Watch(ch chan<- *devicesnapshot.DeviceSnapshot) error {
	lastIndex, err := s.LastIndex()
	if err != nil {
		return err
	}

	mapCh := make(chan *_map.Event)
	if err := s.configs.Watch(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for i := devicesnapshot.Index(1); i <= lastIndex; i++ {
			if device, err := s.GetByIndex(i); err == nil && device != nil {
				ch <- device
			}
		}
		for event := range mapCh {
			if config, err := decodeDeviceSnapshot(event.Entry); err == nil {
				ch <- config
			}
		}
	}()
	return nil
}

func (s *atomixStore) Store(snapshot *devicesnapshot.Snapshot) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	_, err = s.configs.Put(ctx, string(snapshot.ID), bytes)
	if err != nil {
		return err
	}
	return nil
}

func (s *atomixStore) Load(id devicesnapshot.ID) (*devicesnapshot.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.configs.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeSnapshot(entry)
}

func (s *atomixStore) Close() error {
	return s.configs.Close()
}

func decodeDeviceSnapshot(entry *_map.Entry) (*devicesnapshot.DeviceSnapshot, error) {
	config := &devicesnapshot.DeviceSnapshot{}
	if err := proto.Unmarshal(entry.Value, config); err != nil {
		return nil, err
	}
	config.ID = devicesnapshot.ID(entry.Key)
	config.Revision = devicesnapshot.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return config, nil
}

func decodeSnapshot(entry *_map.Entry) (*devicesnapshot.Snapshot, error) {
	config := &devicesnapshot.Snapshot{}
	if err := proto.Unmarshal(entry.Value, config); err != nil {
		return nil, err
	}
	config.ID = devicesnapshot.ID(entry.Key)
	return config, nil
}
