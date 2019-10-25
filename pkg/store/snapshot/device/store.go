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
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/store/utils"
	"github.com/onosproject/onos-config/pkg/types/device"
	devicesnapshot "github.com/onosproject/onos-config/pkg/types/snapshot/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"io"
	"net"
	"time"
)

const deviceSnapshotsName = "device-snapshots"
const snapshotsName = "snapshots"

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

	deviceSnapshots, err := group.GetMap(context.Background(), deviceSnapshotsName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	snapshots, err := group.GetMap(context.Background(), snapshotsName, session.WithTimeout(30*time.Second))
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		deviceSnapshots: deviceSnapshots,
		snapshots:       snapshots,
	}, nil
}

// NewLocalStore returns a new local device snapshot store
func NewLocalStore() (Store, error) {
	_, conn := startLocalNode()
	return newLocalStore(conn)
}

// newLocalStore creates a new local device snapshot store
func newLocalStore(conn *grpc.ClientConn) (Store, error) {
	deviceSnapshotsName := primitive.Name{
		Namespace: "local",
		Name:      deviceSnapshotsName,
	}
	deviceSnapshots, err := _map.New(context.Background(), deviceSnapshotsName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	snapshotsName := primitive.Name{
		Namespace: "local",
		Name:      snapshotsName,
	}
	snapshots, err := _map.New(context.Background(), snapshotsName, []*grpc.ClientConn{conn})
	if err != nil {
		return nil, err
	}

	return &atomixStore{
		deviceSnapshots: deviceSnapshots,
		snapshots:       snapshots,
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

	conn, err := grpc.DialContext(context.Background(), deviceSnapshotsName, grpc.WithContextDialer(dialer), grpc.WithInsecure())
	if err != nil {
		panic("Failed to dial")
	}
	return node, conn
}

// Store stores DeviceChanges
type Store interface {
	io.Closer

	// Get gets a device snapshot
	Get(id devicesnapshot.ID) (*devicesnapshot.DeviceSnapshot, error)

	// Create creates a new device snapshot
	Create(snapshot *devicesnapshot.DeviceSnapshot) error

	// Update updates an existing device snapshot
	Update(snapshot *devicesnapshot.DeviceSnapshot) error

	// Delete deletes a device snapshot
	Delete(snapshot *devicesnapshot.DeviceSnapshot) error

	// List lists device snapshot
	List(chan<- *devicesnapshot.DeviceSnapshot) (stream.Context, error)

	// Watch watches the device snapshot store for changes
	Watch(chan<- stream.Event) (stream.Context, error)

	// Store stores a snapshot
	Store(snapshot *devicesnapshot.Snapshot) error

	// Load loads a snapshot
	Load(deviceID device.VersionedID) (*devicesnapshot.Snapshot, error)

	// Load loads all snapshots
	LoadAll(ch chan<- *devicesnapshot.Snapshot) (stream.Context, error)
}

// atomixStore is the default implementation of the DeviceSnapshot store
type atomixStore struct {
	deviceSnapshots _map.Map
	snapshots       _map.Map
}

func (s *atomixStore) Get(id devicesnapshot.ID) (*devicesnapshot.DeviceSnapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.deviceSnapshots.Get(ctx, string(id))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeDeviceSnapshot(entry)
}

func (s *atomixStore) Create(snapshot *devicesnapshot.DeviceSnapshot) error {
	if snapshot.Revision != 0 {
		return errors.New("not a new object")
	}
	if snapshot.DeviceID == "" {
		return errors.New("no device ID specified")
	}
	if snapshot.DeviceVersion == "" {
		return errors.New("no device version specified")
	}

	snapshot.ID = devicesnapshot.GetSnapshotID(snapshot.NetworkSnapshot.ID, snapshot.DeviceID, snapshot.DeviceVersion)

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	entry, err := s.deviceSnapshots.Put(ctx, string(snapshot.ID), bytes, _map.IfNotSet())
	if err != nil {
		return nil
	}

	snapshot.Revision = devicesnapshot.Revision(entry.Version)
	snapshot.Created = entry.Created
	snapshot.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Update(snapshot *devicesnapshot.DeviceSnapshot) error {
	if snapshot.Revision == 0 {
		return errors.New("not a stored object")
	}
	if snapshot.DeviceID == "" {
		return errors.New("no device ID specified")
	}
	if snapshot.DeviceVersion == "" {
		return errors.New("no device version specified")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	snapshot.Updated = time.Now()
	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	entry, err := s.deviceSnapshots.Put(ctx, string(snapshot.ID), bytes, _map.IfVersion(int64(snapshot.Revision)))
	if err != nil {
		return err
	}

	snapshot.Revision = devicesnapshot.Revision(entry.Version)
	snapshot.Updated = entry.Updated
	return nil
}

func (s *atomixStore) Delete(snapshot *devicesnapshot.DeviceSnapshot) error {
	if snapshot.Revision == 0 {
		return errors.New("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.deviceSnapshots.Remove(ctx, string(snapshot.ID), _map.IfVersion(int64(snapshot.Revision)))
	if err != nil {
		return err
	}

	snapshot.Revision = 0
	snapshot.Updated = entry.Updated
	return nil
}

func (s *atomixStore) List(ch chan<- *devicesnapshot.DeviceSnapshot) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *_map.Entry)
	if err := s.deviceSnapshots.Entries(ctx, mapCh); err != nil {
		cancel()
		return nil, err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if snapshot, err := decodeDeviceSnapshot(entry); err == nil {
				ch <- snapshot
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Watch(ch chan<- stream.Event) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *_map.Event)
	if err := s.deviceSnapshots.Watch(ctx, mapCh, _map.WithReplay()); err != nil {
		cancel()
		return nil, err
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if snapshot, err := decodeDeviceSnapshot(event.Entry); err == nil {
				switch event.Type {
				case _map.EventNone:
					ch <- stream.Event{
						Type:   stream.None,
						Object: snapshot,
					}
				case _map.EventInserted:
					ch <- stream.Event{
						Type:   stream.Created,
						Object: snapshot,
					}
				case _map.EventUpdated:
					ch <- stream.Event{
						Type:   stream.Updated,
						Object: snapshot,
					}
				case _map.EventRemoved:
					ch <- stream.Event{
						Type:   stream.Deleted,
						Object: snapshot,
					}
				}
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Store(snapshot *devicesnapshot.Snapshot) error {
	if snapshot.DeviceID == "" {
		return errors.New("no device ID specified")
	}
	if snapshot.DeviceVersion == "" {
		return errors.New("no device version specified")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return err
	}

	_, err = s.snapshots.Put(ctx, string(snapshot.GetVersionedDeviceID()), bytes)
	if err != nil {
		return err
	}
	return nil
}

func (s *atomixStore) Load(deviceID device.VersionedID) (*devicesnapshot.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.Get(ctx, string(deviceID))
	if err != nil {
		return nil, err
	} else if entry == nil {
		return nil, nil
	}
	return decodeSnapshot(entry)
}

func (s *atomixStore) LoadAll(ch chan<- *devicesnapshot.Snapshot) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan *_map.Entry)
	if err := s.snapshots.Entries(ctx, mapCh); err != nil {
		cancel()
		return nil, err
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if snapshot, err := decodeSnapshot(entry); err == nil {
				ch <- snapshot
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Close() error {
	_ = s.deviceSnapshots.Close()
	return s.snapshots.Close()
}

func decodeDeviceSnapshot(entry *_map.Entry) (*devicesnapshot.DeviceSnapshot, error) {
	snapshot := &devicesnapshot.DeviceSnapshot{}
	if err := proto.Unmarshal(entry.Value, snapshot); err != nil {
		return nil, err
	}
	snapshot.ID = devicesnapshot.ID(entry.Key)
	snapshot.Revision = devicesnapshot.Revision(entry.Version)
	snapshot.Created = entry.Created
	snapshot.Updated = entry.Updated
	return snapshot, nil
}

func decodeSnapshot(entry *_map.Entry) (*devicesnapshot.Snapshot, error) {
	snapshot := &devicesnapshot.Snapshot{}
	if err := proto.Unmarshal(entry.Value, snapshot); err != nil {
		return nil, err
	}
	snapshot.ID = devicesnapshot.ID(entry.Key)
	return snapshot, nil
}
