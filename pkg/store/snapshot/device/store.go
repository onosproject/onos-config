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
	"fmt"
	"github.com/atomix/atomix-go-client/pkg/atomix"
	"github.com/atomix/atomix-go-client/pkg/atomix/map"
	"github.com/atomix/atomix-go-framework/pkg/atomix/meta"
	"github.com/gogo/protobuf/proto"
	"github.com/onosproject/onos-api/go/onos/config/device"
	devicesnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"io"
	"os"
	"time"
)

// NewAtomixStore returns a new persistent Store
func NewAtomixStore() (Store, error) {
	client := atomix.NewClient(atomix.WithClientID(os.Getenv("POD_NAME")))
	deviceSnapshots, err := client.GetMap(context.Background(), fmt.Sprintf("%s-device-snapshots", os.Getenv("SERVICE_NAME")))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	snapshots, err := client.GetMap(context.Background(), fmt.Sprintf("%s-snapshots", os.Getenv("SERVICE_NAME")))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return &atomixStore{
		deviceSnapshots: deviceSnapshots,
		snapshots:       snapshots,
	}, nil
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

	// Watch watches the snapshot store for changes
	WatchAll(chan<- stream.Event) (stream.Context, error)
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
		return nil, errors.FromAtomix(err)
	}
	return decodeDeviceSnapshot(entry)
}

func (s *atomixStore) Create(snapshot *devicesnapshot.DeviceSnapshot) error {
	if snapshot.Revision != 0 {
		return errors.NewInvalid("not a new object")
	}
	if snapshot.DeviceID == "" {
		return errors.NewInvalid("no device ID specified")
	}
	if snapshot.DeviceVersion == "" {
		return errors.NewInvalid("no device version specified")
	}

	snapshot.ID = devicesnapshot.GetSnapshotID(snapshot.NetworkSnapshot.ID, snapshot.DeviceID, snapshot.DeviceVersion)

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return errors.NewInvalid("snapshot encoding failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	entry, err := s.deviceSnapshots.Put(ctx, string(snapshot.ID), bytes, _map.IfNotSet())
	if err != nil {
		return errors.FromAtomix(err)
	}

	snapshot.Revision = devicesnapshot.Revision(entry.Revision)
	return nil
}

func (s *atomixStore) Update(snapshot *devicesnapshot.DeviceSnapshot) error {
	if snapshot.Revision == 0 {
		return errors.NewInvalid("not a stored object")
	}
	if snapshot.DeviceID == "" {
		return errors.NewInvalid("no device ID specified")
	}
	if snapshot.DeviceVersion == "" {
		return errors.NewInvalid("no device version specified")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	snapshot.Updated = time.Now()
	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return errors.NewInvalid("snapshot encoding failed: %v", err)
	}

	entry, err := s.deviceSnapshots.Put(ctx, string(snapshot.ID), bytes, _map.IfMatch(meta.NewRevision(meta.Revision(snapshot.Revision))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	snapshot.Revision = devicesnapshot.Revision(entry.Revision)
	return nil
}

func (s *atomixStore) Delete(snapshot *devicesnapshot.DeviceSnapshot) error {
	if snapshot.Revision == 0 {
		return errors.NewInvalid("not a stored object")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := s.deviceSnapshots.Remove(ctx, string(snapshot.ID), _map.IfMatch(meta.NewRevision(meta.Revision(snapshot.Revision))))
	if err != nil {
		return errors.FromAtomix(err)
	}

	snapshot.Revision = 0
	return nil
}

func (s *atomixStore) List(ch chan<- *devicesnapshot.DeviceSnapshot) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan _map.Entry)
	if err := s.deviceSnapshots.Entries(ctx, mapCh); err != nil {
		cancel()
		return nil, errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if snapshot, err := decodeDeviceSnapshot(&entry); err == nil {
				ch <- snapshot
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

func (s *atomixStore) Watch(ch chan<- stream.Event) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan _map.Event)
	if err := s.deviceSnapshots.Watch(ctx, mapCh, _map.WithReplay()); err != nil {
		cancel()
		return nil, errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if snapshot, err := decodeDeviceSnapshot(&event.Entry); err == nil {
				switch event.Type {
				case _map.EventReplay:
					ch <- stream.Event{
						Type:   stream.None,
						Object: snapshot,
					}
				case _map.EventInsert:
					ch <- stream.Event{
						Type:   stream.Created,
						Object: snapshot,
					}
				case _map.EventUpdate:
					ch <- stream.Event{
						Type:   stream.Updated,
						Object: snapshot,
					}
				case _map.EventRemove:
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
		return errors.NewInvalid("no device ID specified")
	}
	if snapshot.DeviceVersion == "" {
		return errors.NewInvalid("no device version specified")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	bytes, err := proto.Marshal(snapshot)
	if err != nil {
		return errors.NewInvalid("snapshot encoding failed: %v", err)
	}

	_, err = s.snapshots.Put(ctx, string(snapshot.GetVersionedDeviceID()), bytes)
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}

func (s *atomixStore) Load(deviceID device.VersionedID) (*devicesnapshot.Snapshot, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	entry, err := s.snapshots.Get(ctx, string(deviceID))
	if err != nil {
		return nil, errors.FromAtomix(err)
	}
	return decodeSnapshot(entry)
}

func (s *atomixStore) LoadAll(ch chan<- *devicesnapshot.Snapshot) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan _map.Entry)
	if err := s.snapshots.Entries(ctx, mapCh); err != nil {
		cancel()
		return nil, errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for entry := range mapCh {
			if snapshot, err := decodeSnapshot(&entry); err == nil {
				ch <- snapshot
			}
		}
	}()
	return stream.NewCancelContext(cancel), nil
}

// WatchAll is similar to LoadAll for "Snapshot"s, but for continuous streaming
func (s *atomixStore) WatchAll(ch chan<- stream.Event) (stream.Context, error) {
	ctx, cancel := context.WithCancel(context.Background())

	mapCh := make(chan _map.Event)
	if err := s.snapshots.Watch(ctx, mapCh, _map.WithReplay()); err != nil {
		cancel()
		return nil, errors.FromAtomix(err)
	}

	go func() {
		defer close(ch)
		for event := range mapCh {
			if snapshot, err := decodeSnapshot(&event.Entry); err == nil {
				switch event.Type {
				case _map.EventReplay:
					ch <- stream.Event{
						Type:   stream.None,
						Object: snapshot,
					}
				case _map.EventInsert:
					ch <- stream.Event{
						Type:   stream.Created,
						Object: snapshot,
					}
				case _map.EventUpdate:
					ch <- stream.Event{
						Type:   stream.Updated,
						Object: snapshot,
					}
				case _map.EventRemove:
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

func (s *atomixStore) Close() error {
	_ = s.deviceSnapshots.Close(context.Background())
	err := s.snapshots.Close(context.Background())
	if err != nil {
		return errors.FromAtomix(err)
	}
	return nil
}

func decodeDeviceSnapshot(entry *_map.Entry) (*devicesnapshot.DeviceSnapshot, error) {
	snapshot := &devicesnapshot.DeviceSnapshot{}
	if err := proto.Unmarshal(entry.Value, snapshot); err != nil {
		return nil, errors.NewInvalid("device snapshot decoding failed: %v", err)
	}
	snapshot.ID = devicesnapshot.ID(entry.Key)
	snapshot.Revision = devicesnapshot.Revision(entry.Revision)
	return snapshot, nil
}

func decodeSnapshot(entry *_map.Entry) (*devicesnapshot.Snapshot, error) {
	snapshot := &devicesnapshot.Snapshot{}
	if err := proto.Unmarshal(entry.Value, snapshot); err != nil {
		return nil, errors.NewInvalid("snapshot decoding failed: %v", err)
	}
	snapshot.ID = devicesnapshot.ID(entry.Key)
	return snapshot, nil
}
