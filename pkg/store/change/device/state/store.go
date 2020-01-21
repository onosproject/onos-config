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
	"errors"
	"github.com/cenkalti/backoff"
	changetype "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicesnapshotstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"sort"
	"strings"
	"sync"
	"time"
)

// NewStore returns a new store backed by the device change store
func NewStore(networkChangeStore networkchangestore.Store, deviceSnapshotStore devicesnapshotstore.Store) (Store, error) {
	store := &deviceChangeStoreStateStore{
		changeStore:   networkChangeStore,
		snapshotStore: deviceSnapshotStore,
		devices:       make(map[devicetype.VersionedID]*deviceChangeStateStore),
		waiters:       make(map[networkchange.Revision]chan struct{}),
	}
	if err := store.listen(); err != nil {
		return nil, err
	}
	return store, nil
}

// Store is a device state store
type Store interface {
	// Get gets the state of the given device
	Get(id devicetype.VersionedID, revision networkchange.Revision) ([]*devicechange.PathValue, error)
}

// deviceChangeStoreStateStore is a device state store that listens to the device change store
type deviceChangeStoreStateStore struct {
	changeStore   networkchangestore.Store
	snapshotStore devicesnapshotstore.Store
	devices       map[devicetype.VersionedID]*deviceChangeStateStore
	waiters       map[networkchange.Revision]chan struct{}
	changeIndex   networkchange.Index
	rollbackIndex networkchange.Index
	revision      networkchange.Revision
	mu            sync.RWMutex
}

func (s *deviceChangeStoreStateStore) listen() error {
	return backoff.Retry(s.watch, backoff.NewConstantBackOff(1*time.Second))
}

func (s *deviceChangeStoreStateStore) watch() error {
	ch := make(chan stream.Event)
	watchCtx, err := s.changeStore.Watch(ch, networkchangestore.WithReplay())
	if err != nil {
		return err
	}
	go func() {
		defer watchCtx.Close()
		s.processCh(ch)
	}()
	return nil
}

func (s *deviceChangeStoreStateStore) processCh(ch chan stream.Event) {
	for event := range ch {
		s.mu.Lock()
		err := s.processChange(event.Object.(*networkchange.NetworkChange))
		s.mu.Unlock()
		if err != nil {
			go func() {
				_ = s.listen()
			}()
			return
		}
	}
}

func (s *deviceChangeStoreStateStore) processChange(networkChange *networkchange.NetworkChange) error {
	switch networkChange.Status.Phase {
	case changetype.Phase_CHANGE:
		if networkChange.Index <= s.changeIndex {
			return nil
		}
		if err := s.processNetworkChange(networkChange); err != nil {
			return err
		}
		s.changeIndex = networkChange.Index
	case changetype.Phase_ROLLBACK:
		if networkChange.Index <= s.rollbackIndex {
			return nil
		}
		if err := s.processNetworkRollback(networkChange); err != nil {
			return err
		}
		s.rollbackIndex = networkChange.Index
	}

	if networkChange.Revision > s.revision {
		s.revision = networkChange.Revision
	}

	waiter, ok := s.waiters[networkChange.Revision]
	if ok {
		delete(s.waiters, networkChange.Revision)
		close(waiter)
	}
	return nil
}

func (s *deviceChangeStoreStateStore) processNetworkChange(networkChange *networkchange.NetworkChange) error {
	for _, deviceChange := range networkChange.Changes {
		state, ok := s.devices[deviceChange.GetVersionedDeviceID()]
		if !ok {
			state = &deviceChangeStateStore{
				deviceID: deviceChange.GetVersionedDeviceID(),
				state:    make(map[string]*devicechange.TypedValue),
			}
			snapshot, err := s.snapshotStore.Load(deviceChange.GetVersionedDeviceID())
			if err != nil {
				return err
			} else if snapshot != nil {
				for _, value := range snapshot.Values {
					state.update(value)
				}
			}
			s.devices[deviceChange.GetVersionedDeviceID()] = state
		}

		for _, value := range deviceChange.Values {
			if value.Removed {
				state.remove(value.Path)
			} else {
				state.update(&devicechange.PathValue{
					Path:  value.Path,
					Value: value.Value,
				})
			}
		}
	}
	return nil
}

func (s *deviceChangeStoreStateStore) processNetworkRollback(networkChange *networkchange.NetworkChange) error {
	listCh := make(chan *networkchange.NetworkChange)
	listCtx, err := s.changeStore.List(listCh)
	if err != nil {
		listCtx.Close()
		return err
	}

	states := make(map[devicetype.VersionedID]*deviceChangeStateStore)
	for _, devChange := range networkChange.Changes {
		state := &deviceChangeStateStore{
			deviceID: devChange.GetVersionedDeviceID(),
			state:    make(map[string]*devicechange.TypedValue),
		}
		snapshot, err := s.snapshotStore.Load(devChange.GetVersionedDeviceID())
		if err != nil {
			listCtx.Close()
			return err
		} else if snapshot != nil {
			for _, value := range snapshot.Values {
				state.update(value)
			}
		}
		states[devChange.GetVersionedDeviceID()] = state
	}

	for netChange := range listCh {
		if netChange.Index >= networkChange.Index {
			listCtx.Close()
			break
		}
		if netChange.Status.Phase == changetype.Phase_CHANGE {
			for _, devChange := range netChange.Changes {
				state, ok := s.devices[devChange.GetVersionedDeviceID()]
				if ok {
					for _, value := range devChange.Values {
						if value.Removed {
							state.remove(value.Path)
						} else {
							state.update(&devicechange.PathValue{
								Path:  value.Path,
								Value: value.Value,
							})
						}
					}
				}
			}
		}
	}
	for device, state := range states {
		s.devices[device] = state
	}
	return nil
}

func (s *deviceChangeStoreStateStore) Get(id devicetype.VersionedID, revision networkchange.Revision) ([]*devicechange.PathValue, error) {
	s.mu.RLock()
	if s.revision < revision {
		s.mu.RUnlock()
		s.mu.Lock()
		if s.revision < revision {
			waiter, ok := s.waiters[revision]
			if !ok {
				waiter = make(chan struct{})
				s.waiters[revision] = waiter
			}
			s.mu.Unlock()
			select {
			case <-waiter:
			case <-time.After(15 * time.Second):
				return nil, errors.New("get timeout")
			}
			s.mu.RLock()
			defer s.mu.RUnlock()
		} else {
			defer s.mu.Unlock()
		}
	} else {
		defer s.mu.RUnlock()
	}

	device, ok := s.devices[id]
	if !ok {
		return []*devicechange.PathValue{}, nil
	}
	return device.get()
}

// deviceChangeStateStore is a device state store that listens to changes for a specific device
type deviceChangeStateStore struct {
	deviceID devicetype.VersionedID
	state    map[string]*devicechange.TypedValue
}

func (s *deviceChangeStateStore) update(value *devicechange.PathValue) {
	s.state[value.Path] = value.Value
}

func (s *deviceChangeStateStore) remove(rootPath string) {
	delete(s.state, rootPath)
	for path := range s.state {
		if strings.Contains(path, rootPath) {
			delete(s.state, path)
		}
	}
}

// get gets the state of the device up to the given revision
func (s *deviceChangeStateStore) get() ([]*devicechange.PathValue, error) {
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
