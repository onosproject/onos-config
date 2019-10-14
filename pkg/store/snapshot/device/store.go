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
	"github.com/atomix/atomix-go-client/pkg/client/map"
	"github.com/atomix/atomix-go-client/pkg/client/primitive"
	"github.com/atomix/atomix-go-client/pkg/client/session"
	"github.com/atomix/atomix-go-local/pkg/atomix/local"
	"github.com/atomix/atomix-go-node/pkg/atomix"
	"github.com/atomix/atomix-go-node/pkg/atomix/registry"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/store/utils"
	devicesnapshot "github.com/onosproject/onos-config/pkg/types/snapshot/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

const primitiveName = "device-snapshots"

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

	return &atomixStore{
		configs: configs,
	}, nil
}

// NewLocalStore returns a new local device store
func NewLocalStore() (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(conn)
}

// newLocalStore creates a new local device change store
func newLocalStore(conn *grpc.ClientConn) (Store, error) {
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

	// Get gets a device snapshot
	Get(id devicesnapshot.ID) (*devicesnapshot.DeviceSnapshot, error)

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

func (s *atomixStore) Create(config *devicesnapshot.DeviceSnapshot) error {
	if config.Revision != 0 {
		return errors.New("not a new object")
	}

	config.ID = devicesnapshot.GetSnapshotID(config.NetworkSnapshotID, config.DeviceID)

	bytes, err := proto.Marshal(config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	entry, err := s.configs.Put(ctx, string(config.ID), bytes, _map.IfNotSet())
	if err != nil {
		return nil
	}

	config.Revision = devicesnapshot.Revision(entry.Version)
	config.Created = entry.Created
	config.Updated = entry.Updated
	return nil
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

	mapCh := make(chan *_map.Entry)
	if err := s.configs.Entries(context.Background(), mapCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if config, err := decodeDeviceSnapshot(entry); err == nil {
				ch <- config
			}
		}
	}()
	return nil
}

func (s *atomixStore) Watch(ch chan<- *devicesnapshot.DeviceSnapshot) error {
	mapCh := make(chan *_map.Event)
	if err := s.configs.Watch(context.Background(), mapCh, _map.WithReplay()); err != nil {
		return err
	}

	go func() {
		defer close(ch)
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
