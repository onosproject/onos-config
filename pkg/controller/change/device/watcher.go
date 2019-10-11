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
	"github.com/onosproject/onos-config/pkg/controller"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/types"
	devicechangetype "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"sync"
)

const queueSize = 100

// Watcher is a device change watcher
type Watcher struct {
	DeviceStore devicestore.Store
	ChangeStore devicechangestore.Store
	ch          chan<- types.ID
	channels    map[device.ID]chan *devicechangetype.DeviceChange
	mu          sync.Mutex
	wg          sync.WaitGroup
}

// Start starts the device change watcher
func (w *Watcher) Start(ch chan<- types.ID) error {
	w.mu.Lock()
	if w.ch != nil {
		w.mu.Unlock()
		return nil
	}

	w.ch = ch
	w.channels = make(map[device.ID]chan *devicechangetype.DeviceChange)
	w.mu.Unlock()

	deviceCh := make(chan *device.Device)
	if err := w.DeviceStore.Watch(deviceCh); err != nil {
		return err
	}

	go func() {
		for device := range deviceCh {
			w.watchDevice(device.ID, ch)
		}
	}()
	return nil
}

// watchDevice watches changes for the given device
func (w *Watcher) watchDevice(device device.ID, ch chan<- types.ID) {
	w.mu.Lock()
	deviceCh := w.channels[device]
	if deviceCh != nil {
		w.mu.Unlock()
		return
	}

	deviceCh = make(chan *devicechangetype.DeviceChange, queueSize)
	w.channels[device] = deviceCh
	w.mu.Unlock()

	if err := w.ChangeStore.Watch(device, deviceCh); err != nil {
		w.mu.Lock()
		delete(w.channels, device)
		w.mu.Unlock()
		return
	}

	w.wg.Add(1)
	go func() {
		for request := range deviceCh {
			ch <- types.ID(request.ID)
		}
		w.wg.Done()
	}()
}

// Stop stops the device change watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	for _, ch := range w.channels {
		close(ch)
	}
	w.mu.Unlock()
	w.wg.Wait()
	w.mu.Lock()
	if w.ch != nil {
		close(w.ch)
	}
	w.mu.Unlock()
}

var _ controller.Watcher = &Watcher{}
