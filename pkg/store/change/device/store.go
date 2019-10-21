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
	"github.com/atomix/atomix-go-client/pkg/client/indexedmap"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/store/utils"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"sync"
	"time"
)

// getDeviceChangesName returns the name of the changes map for the given device ID
func getDeviceChangesName(deviceID devicetopo.ID) string {
	return fmt.Sprintf("device-changes-%s", deviceID)
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

	changesFactory := func(deviceID devicetopo.ID) (indexedmap.IndexedMap, error) {
		return group.GetIndexedMap(context.Background(), getDeviceChangesName(deviceID), session.WithTimeout(30*time.Second))
	}

	return &atomixStore{
		changesFactory: changesFactory,
		deviceChanges:  make(map[devicetopo.ID]indexedmap.IndexedMap),
	}, nil
}

// NewLocalStore returns a new local device store
func NewLocalStore() (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(conn)
}

// newLocalStore creates a new local device change store
func newLocalStore(conn *grpc.ClientConn) (Store, error) {
	changesFactory := func(deviceID devicetopo.ID) (indexedmap.IndexedMap, error) {
		counterName := primitive.Name{
			Namespace: "local",
			Name:      getDeviceChangesName(deviceID),
		}
		return indexedmap.New(context.Background(), counterName, []*grpc.ClientConn{conn})
	}

	return &atomixStore{
		changesFactory: changesFactory,
		deviceChanges:  make(map[devicetopo.ID]indexedmap.IndexedMap),
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

	conn, err := grpc.DialContext(context.Background(), "device-changes", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial network configurations")
	}
	return node, conn
}

// Store stores DeviceChanges
type Store interface {
	io.Closer

	// Get gets a device change
	Get(id devicechangetypes.ID) (*devicechangetypes.DeviceChange, error)

	// Create creates a new device change
	Create(config *devicechangetypes.DeviceChange) error

	// Update updates an existing device change
	Update(config *devicechangetypes.DeviceChange) error

	// Delete deletes a device change
	Delete(config *devicechangetypes.DeviceChange) error

	// List lists device change
	List(devicetopo.ID, chan<- *devicechangetypes.DeviceChange) error

	// Watch watches the device change store for changes
	Watch(devicetopo.ID, chan<- *devicechangetypes.DeviceChange) error
}

// atomixStore is the default implementation of the NetworkConfig store
type atomixStore struct {
	changesFactory func(deviceID devicetopo.ID) (indexedmap.IndexedMap, error)
	deviceChanges  map[devicetopo.ID]indexedmap.IndexedMap
	mu             sync.RWMutex
}

func (s *atomixStore) getDeviceChanges(deviceID devicetopo.ID) (indexedmap.IndexedMap, error) {
	s.mu.RLock()
	changes, ok := s.deviceChanges[deviceID]
	s.mu.RUnlock()
	if !ok {
		s.mu.Lock()
		defer s.mu.Unlock()
		changes, ok = s.deviceChanges[deviceID]
		if !ok {
			newChanges, err := s.changesFactory(deviceID)
			if err != nil {
				return nil, err
			}
			s.deviceChanges[deviceID] = newChanges
			return newChanges, nil
		}
	}
	return changes, nil
}

func (s *atomixStore) Get(id devicechangetypes.ID) (*devicechangetypes.DeviceChange, error) {
	changes, err := s.getDeviceChanges(id.GetDeviceID())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := changes.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeChange(entry)
}

func (s *atomixStore) Create(config *devicechangetypes.DeviceChange) error {
	if config.Change.DeviceID == "" {
		return errors.New("no device ID specified")
	}
	if config.NetworkChange.ID == "" {
		return errors.New("no NetworkChange ID specified")
	}
	if config.Revision != 0 {
		return errors.New("not a new object")
	}
	config.ID = networkchangetypes.ID(config.NetworkChange.ID).GetDeviceChangeID(config.Change.DeviceID)

	changes, err := s.getDeviceChanges(config.Change.DeviceID)
	if err != nil {
		return err
	}

	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := changes.Put(ctx, string(config.ID), bytes, indexedmap.IfNotSet())
	if err != nil {
		return err
	}

	config.Index = devicechangetypes.Index(entry.Index)
	config.Revision = devicechangetypes.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(config *devicechangetypes.DeviceChange) error {
	if config.ID == "" {
		return errors.New("no change ID configured")
	}
	if config.Index == 0 {
		return errors.New("not a stored object: no storage index found")
	}
	if config.Revision == 0 {
		return errors.New("not a stored object: no storage revision found")
	}

	changes, err := s.getDeviceChanges(config.Change.DeviceID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	entry, err := changes.Put(ctx, string(config.ID), bytes, indexedmap.IfVersion(indexedmap.Version(config.Revision)))
	if err != nil {
		return err
	}

	config.Revision = devicechangetypes.Revision(entry.Version)
	if config.Created.IsZero() {
		config.Created = entry.Created
	}
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(config *devicechangetypes.DeviceChange) error {
	if config.ID == "" {
		return errors.New("no change ID configured")
	}
	if config.Index == 0 {
		return errors.New("not a stored object: no storage index found")
	}
	if config.Revision == 0 {
		return errors.New("not a stored object")
	}

	changes, err := s.getDeviceChanges(config.Change.DeviceID)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := changes.RemoveIndex(ctx, indexedmap.Index(config.Index), indexedmap.IfVersion(indexedmap.Version(config.Revision)))
	if err != nil {
		return err
	}

	config.Revision = 0
	config.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(device devicetopo.ID, ch chan<- *devicechangetypes.DeviceChange) error {
	changes, err := s.getDeviceChanges(device)
	if err != nil {
		return err
	}

	mapCh := make(chan *indexedmap.Entry)
	if err := changes.Entries(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if config, err := decodeChange(entry); err == nil {
				ch <- config
			}
		}
	}()
	return nil
}

func (s *atomixStore) Watch(device devicetopo.ID, ch chan<- *devicechangetypes.DeviceChange) error {
	changes, err := s.getDeviceChanges(device)
	if err != nil {
		return err
	}

	mapCh := make(chan *indexedmap.Event)
	if err := changes.Watch(context.Background(), mapCh, indexedmap.WithReplay()); err != nil {
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
	var returnErr error
	for _, changes := range s.deviceChanges {
		if err := changes.Close(); err != nil {
			returnErr = err
		}
	}
	return returnErr
}

func decodeChange(entry *indexedmap.Entry) (*devicechangetypes.DeviceChange, error) {
	change := &devicechangetypes.DeviceChange{}
	if err := proto.Unmarshal(entry.Value, change); err != nil {
		return nil, err
	}
	change.ID = devicechangetypes.ID(entry.Key)
	change.Index = devicechangetypes.Index(entry.Index)
	change.Revision = devicechangetypes.Revision(entry.Version)
	change.Created = entry.Created
	change.Updated = entry.Updated
	return change, nil
}
