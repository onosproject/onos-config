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
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/types"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-config/pkg/types/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"sync"
)

const queueSize = 100

// Watcher is a network change watcher
type Watcher struct {
	Store networkchangestore.Store
	ctx   stream.Context
	mu    sync.Mutex
}

// Start starts the network change watcher
func (w *Watcher) Start(ch chan<- types.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx != nil {
		return nil
	}

	configCh := make(chan stream.Event, queueSize)
	ctx, err := w.Store.Watch(configCh, networkchangestore.WithReplay())
	if err != nil {
		return err
	}
	w.ctx = ctx

	go func() {
		for request := range configCh {
			ch <- types.ID(request.Object.(*networkchangetypes.NetworkChange).ID)
		}
		close(ch)
	}()
	return nil
}

// Stop stops the network change watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.ctx != nil {
		w.ctx.Close()
	}
	w.mu.Unlock()
}

var _ controller.Watcher = &Watcher{}

// DeviceWatcher is a device change watcher
type DeviceWatcher struct {
	DeviceStore devicestore.Store
	ChangeStore devicechangestore.Store
	ch          chan<- types.ID
	streams     map[device.VersionedID]stream.Context
	mu          sync.Mutex
	wg          sync.WaitGroup
}

// Start starts the device change watcher
func (w *DeviceWatcher) Start(ch chan<- types.ID) error {
	w.mu.Lock()
	if w.ch != nil {
		w.mu.Unlock()
		return nil
	}

	w.ch = ch
	w.streams = make(map[device.VersionedID]stream.Context)
	w.mu.Unlock()

	deviceCh := make(chan *devicetopo.ListResponse)
	if err := w.DeviceStore.Watch(deviceCh); err != nil {
		return err
	}

	go func() {
		for event := range deviceCh {
			w.watchDevice(device.NewVersionedID(device.ID(event.Device.ID), device.Version(event.Device.Version)), ch)
		}
	}()
	return nil
}

// watchDevice watches changes for the given device
func (w *DeviceWatcher) watchDevice(deviceID device.VersionedID, ch chan<- types.ID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ctx := w.streams[deviceID]
	if ctx != nil {
		return
	}

	deviceCh := make(chan stream.Event, queueSize)
	ctx, err := w.ChangeStore.Watch(deviceID, deviceCh, devicechangestore.WithReplay())
	if err != nil {
		return
	}
	w.streams[deviceID] = ctx

	w.wg.Add(1)
	go func() {
		for event := range deviceCh {
			ch <- types.ID(event.Object.(*devicechangetypes.DeviceChange).NetworkChange.ID)
		}
		w.wg.Done()
	}()
}

// Stop stops the device change watcher
func (w *DeviceWatcher) Stop() {
	w.mu.Lock()
	for _, ctx := range w.streams {
		ctx.Close()
	}
	w.mu.Unlock()
	w.wg.Wait()
	w.mu.Lock()
	if w.ch != nil {
		close(w.ch)
	}
	w.mu.Unlock()
}

var _ controller.Watcher = &DeviceWatcher{}
