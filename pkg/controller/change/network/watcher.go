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

package network

import (
	"github.com/onosproject/onos-config/pkg/controller"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/types"
	devicechangetype "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetype "github.com/onosproject/onos-config/pkg/types/change/network"
	"sync"
)

const queueSize = 100

// Watcher is a network change watcher
type Watcher struct {
	Store networkchangestore.Store
	ch    chan *networkchangetype.NetworkChange
	mu    sync.Mutex
}

// Start starts the network change watcher
func (w *Watcher) Start(ch chan<- types.ID) error {
	w.mu.Lock()
	if w.ch != nil {
		return nil
	}

	configCh := make(chan *networkchangetype.NetworkChange, queueSize)
	w.ch = configCh
	w.mu.Unlock()

	if err := w.Store.Watch(configCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for request := range configCh {
			ch <- types.ID(request.ID)
		}
	}()
	return nil
}

// Stop stops the network change watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	close(w.ch)
	w.mu.Unlock()
}

var _ controller.Watcher = &Watcher{}

// DeviceWatcher is a device change watcher
type DeviceWatcher struct {
	Store devicechangestore.Store
	ch    chan *devicechangetype.Change
	mu    sync.Mutex
}

// Start starts the device change watcher
func (w *DeviceWatcher) Start(ch chan<- types.ID) error {
	w.mu.Lock()
	if w.ch != nil {
		return nil
	}

	configCh := make(chan *devicechangetype.Change, queueSize)
	w.ch = configCh
	w.mu.Unlock()

	if err := w.Store.Watch(configCh); err != nil {
		return err
	}

	go func() {
		defer close(ch)
		for request := range configCh {
			ch <- types.ID(request.NetworkChangeID)
		}
	}()
	return nil
}

// Stop stops the device change watcher
func (w *DeviceWatcher) Stop() {
	w.mu.Lock()
	close(w.ch)
	w.mu.Unlock()
}

var _ controller.Watcher = &Watcher{}
