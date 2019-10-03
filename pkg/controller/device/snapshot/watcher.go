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

package snapshot

import (
	"github.com/onosproject/onos-config/pkg/controller"
	snapshotstore "github.com/onosproject/onos-config/pkg/store/device/snapshot"
	"github.com/onosproject/onos-config/pkg/types"
	snapshottype "github.com/onosproject/onos-config/pkg/types/device/snapshot"
	"sync"
)

const queueSize = 100

// Watcher is a change watcher
type Watcher struct {
	Store snapshotstore.Store
	ch    chan *snapshottype.DeviceSnapshot
	mu    sync.Mutex
}

// Start starts the change watcher
func (w *Watcher) Start(ch chan<- types.ID) error {
	defer close(ch)

	w.mu.Lock()
	if w.ch != nil {
		return nil
	}

	requestCh := make(chan *snapshottype.DeviceSnapshot, queueSize)
	w.ch = requestCh
	w.mu.Unlock()

	if err := w.Store.Watch(requestCh); err != nil {
		return err
	}

	for request := range requestCh {
		ch <- types.ID(request.ID)
	}
	return nil
}

// Stop stops the change watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	close(w.ch)
	w.mu.Unlock()
}

var _ controller.Watcher = &Watcher{}

// OwnerWatcher is a change network watcher
type OwnerWatcher struct {
	Store snapshotstore.Store
	ch    chan *snapshottype.DeviceSnapshot
	mu    sync.Mutex
}

// Start starts the change owner watcher
func (w *OwnerWatcher) Start(ch chan<- types.ID) error {
	defer close(ch)

	w.mu.Lock()
	if w.ch != nil {
		return nil
	}

	requestCh := make(chan *snapshottype.DeviceSnapshot, queueSize)
	w.ch = requestCh
	w.mu.Unlock()

	if err := w.Store.Watch(requestCh); err != nil {
		return err
	}

	for request := range requestCh {
		ch <- types.ID(request.ID)
	}
	return nil
}

// Stop stops the change owner watcher
func (w *OwnerWatcher) Stop() {
	w.mu.Lock()
	close(w.ch)
	w.mu.Unlock()
}

var _ controller.Watcher = &OwnerWatcher{}
