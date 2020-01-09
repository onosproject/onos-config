// Copyright 2020-present Open Networking Foundation.
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

package state

import (
	changetype "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicesnapshotstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"sort"
	"sync"
)

// NewStore returns a new store backed by the device change store
func NewStore(deviceChangeStore devicechangestore.Store, deviceSnapshotStore devicesnapshotstore.Store) (Store, error) {
	return &deviceChangeStoreStateStore{
		changeStore:   deviceChangeStore,
		snapshotStore: deviceSnapshotStore,
		devices:       make(map[devicetype.VersionedID]*deviceChangeStateStore),
	}, nil
}

// Store is a device state store
type Store interface {
	// Get gets the state of the given device
	Get(id devicetype.VersionedID) ([]*devicechange.PathValue, error)
}

// deviceChangeStoreStateStore is a device state store that listens to the device change store
type deviceChangeStoreStateStore struct {
	changeStore   devicechangestore.Store
	snapshotStore devicesnapshotstore.Store
	devices       map[devicetype.VersionedID]*deviceChangeStateStore
	mu            sync.RWMutex
}

func (s *deviceChangeStoreStateStore) Get(id devicetype.VersionedID) ([]*devicechange.PathValue, error) {
	s.mu.RLock()
	device, ok := s.devices[id]
	s.mu.RUnlock()
	if ok {
		return device.get()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	device, ok = s.devices[id]
	if ok {
		return device.get()
	}

	device, err := newDeviceStore(id, s.changeStore, s.snapshotStore, func() {
		s.mu.Lock()
		delete(s.devices, id)
		s.mu.Unlock()
	})
	if err != nil {
		return nil, err
	}
	s.devices[id] = device
	return device.get()
}

// newDeviceStore returns the state store for the given device
func newDeviceStore(deviceID devicetype.VersionedID, deviceChangeStore devicechangestore.Store, deviceSnapshotStore devicesnapshotstore.Store, closeFunc func()) (*deviceChangeStateStore, error) {
	store := &deviceChangeStateStore{
		deviceID:      deviceID,
		changeStore:   deviceChangeStore,
		snapshotStore: deviceSnapshotStore,
		closeFunc:     closeFunc,
		state:         make(map[string]*devicechange.TypedValue),
	}
	if err := store.listen(); err != nil {
		return nil, err
	}
	return store, nil
}

// deviceChangeStateStore is a device state store that listens to changes for a specific device
type deviceChangeStateStore struct {
	deviceID      devicetype.VersionedID
	changeStore   devicechangestore.Store
	snapshotStore devicesnapshotstore.Store
	closeFunc     func()
	state         map[string]*devicechange.TypedValue
	mu            sync.RWMutex
}

// listen starts listening for device changes
func (s *deviceChangeStateStore) listen() error {
	ch := make(chan stream.Event)
	if _, err := s.changeStore.Watch(s.deviceID, ch); err != nil {
		return err
	}
	go func() {
		for event := range ch {
			s.mu.Lock()
			change := event.Object.(*devicechange.DeviceChange)
			switch change.Status.Phase {
			case changetype.Phase_CHANGE:
				if err := s.updateChange(change); err != nil {
					s.closeFunc()
					return
				}
			case changetype.Phase_ROLLBACK:
				if err := s.updateRollback(change); err != nil {
					s.closeFunc()
					return
				}
			}
			s.mu.Unlock()
		}
	}()
	return nil
}

func (s *deviceChangeStateStore) updateChange(change *devicechange.DeviceChange) error {
	for _, changeValue := range change.Change.Values {
		if changeValue.Removed {
			delete(s.state, changeValue.Path)
		} else {
			s.state[changeValue.Path] = changeValue.Value
		}
	}
	return nil
}

func (s *deviceChangeStateStore) updateRollback(change *devicechange.DeviceChange) error {
	if change.Status.State != changetype.State_COMPLETE {
		return nil
	}

	snapshot, err := s.snapshotStore.Load(s.deviceID)
	if err != nil {
		return err
	}

	changes := make(map[string]*devicechange.TypedValue)
	for _, snapshotChange := range snapshot.Values {
		changes[snapshotChange.Path] = snapshotChange.Value
	}

	ch := make(chan *devicechange.DeviceChange)
	_, err = s.changeStore.List(s.deviceID, ch)
	if err != nil {
		return err
	}

	for prevChange := range ch {
		if prevChange.Index >= change.Index {
			break
		}
		for _, value := range prevChange.Change.Values {
			changes[value.Path] = value.Value
		}
	}

	for path, value := range changes {
		s.state[path] = value
	}

	for _, changeValue := range change.Change.Values {
		if _, ok := changes[changeValue.Path]; !ok {
			delete(s.state, changeValue.Path)
		}
	}
	return nil
}

// get gets the state of the device
func (s *deviceChangeStateStore) get() ([]*devicechange.PathValue, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := make([]*devicechange.PathValue, 0, len(s.state))
	for path, value := range s.state {
		state = append(state, &devicechange.PathValue{
			Path:  path,
			Value: value,
		})
	}
	sort.Slice(state, func(i, j int) bool {
		return state[i].Path < state[j].Path
	})
	return state, nil
}
