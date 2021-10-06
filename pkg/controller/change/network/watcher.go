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
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	"github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-api/go/onos/topo"
	devicetopo "github.com/onosproject/onos-config/pkg/device"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-lib-go/pkg/controller"
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
func (w *Watcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx != nil {
		return nil
	}

	networkChangeCh := make(chan stream.Event, queueSize)
	ctx, err := w.Store.Watch(networkChangeCh, networkchangestore.WithReplay())
	if err != nil {
		return err
	}
	w.ctx = ctx

	go func() {
		for networkChangeEvent := range networkChangeCh {
			ch <- controller.NewID(string(networkChangeEvent.Object.(*networkchange.NetworkChange).ID))
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

// DeviceWatcher is a device watcher
type DeviceWatcher struct {
	DeviceStore devicestore.Store
	ChangeStore networkchangestore.Store
	ch          chan<- controller.ID
	mu          sync.Mutex
}

// Start starts the device change watcher
func (w *DeviceWatcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	if w.ch != nil {
		w.mu.Unlock()
		return nil
	}

	w.ch = ch
	w.mu.Unlock()

	deviceCh := make(chan *devicetopo.ListResponse)
	if err := w.DeviceStore.Watch(deviceCh); err != nil {
		return err
	}

	go func() {
		for response := range deviceCh {
			log.Infof("Received device event for device %v %v", response.Device.ID, response.Device.Version)
			deviceID := device.NewVersionedID(device.ID(response.Device.ID), device.Version(response.Device.Version))
			networkChangeCh := make(chan *networkchange.NetworkChange)
			ctx, err := w.ChangeStore.List(networkChangeCh)
			if err != nil {
				log.Errorf(err.Error())
			} else {
				for networkChange := range networkChangeCh {
					for _, deviceChange := range networkChange.Changes {
						if device.NewVersionedID(deviceChange.DeviceID, deviceChange.DeviceVersion) == deviceID {
							ch <- controller.NewID(string(networkChange.ID))
						}
					}
				}
			}
			ctx.Close()
		}
	}()
	return nil
}

// Stop stops the device change watcher
func (w *DeviceWatcher) Stop() {
	w.mu.Lock()
	if w.ch != nil {
		close(w.ch)
	}
	w.mu.Unlock()
}

var _ controller.Watcher = &DeviceChangeWatcher{}

// DeviceChangeWatcher is a device change watcher
type DeviceChangeWatcher struct {
	DeviceStore devicestore.Store
	ChangeStore devicechangestore.Store
	ch          chan<- controller.ID
	streams     map[device.VersionedID]stream.Context
	mu          sync.Mutex
	wg          sync.WaitGroup
}

// Start starts the device change watcher
func (w *DeviceChangeWatcher) Start(ch chan<- controller.ID) error {
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
		for response := range deviceCh {
			w.updateWatch(response.Device, ch)
		}
	}()
	return nil
}

// updateWatch watches changes for the given device
func (w *DeviceChangeWatcher) updateWatch(topodevice *devicetopo.Device, ch chan<- controller.ID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	deviceID := device.NewVersionedID(device.ID(topodevice.ID), device.Version(topodevice.Version))
	log.Infof("Updating watch for device %v", deviceID)

	// If the protocol state is connected, ensure a stream is open
	// If the protocol state is not connected, close any stream that's open
	ctx := w.streams[deviceID]
	state := getProtocolState(topodevice)
	if state == topo.ChannelState_CONNECTED {
		if ctx != nil {
			return
		}
	} else {
		if ctx != nil {
			log.Infof("Closing watch for device %v: device disconnected", deviceID)
			ctx.Close()
			delete(w.streams, deviceID)
		}
		return
	}

	// Open a new device event stream
	deviceCh := make(chan stream.Event, queueSize)
	ctx, err := w.ChangeStore.Watch(deviceID, deviceCh, devicechangestore.WithReplay())
	if err != nil {
		log.Errorf("Failed to watch device %v: %v", deviceID, err)
		return
	}
	w.streams[deviceID] = ctx

	log.Infof("Watching device %v (%s)", deviceID, topodevice.Type)

	w.wg.Add(1)
	go func() {
		for event := range deviceCh {
			ch <- controller.NewID(string(event.Object.(*devicechange.DeviceChange).NetworkChange.ID))
		}
		w.wg.Done()
	}()
}

// Stop stops the device change watcher
func (w *DeviceChangeWatcher) Stop() {
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

var _ controller.Watcher = &DeviceChangeWatcher{}
