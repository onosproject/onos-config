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
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	devicetopo "github.com/onosproject/onos-config/pkg/device"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"sync"
)

const queueSize = 100

// Watcher is a device watcher
type Watcher struct {
	DeviceStore devicestore.Store
	ChangeStore devicechangestore.Store
	ch          chan<- controller.ID
	cacheStream stream.Context
	mu          sync.Mutex
}

// Start starts the device change watcher
func (w *Watcher) Start(ch chan<- controller.ID) error {
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
			deviceID := devicetype.NewVersionedID(devicetype.ID(response.Device.ID), devicetype.Version(response.Device.Version))
			deviceChangeCh := make(chan *devicechange.DeviceChange)
			ctx, err := w.ChangeStore.List(deviceID, deviceChangeCh)
			if err != nil {
				log.Errorf(err.Error())
			} else {
				for deviceChange := range deviceChangeCh {
					ch <- controller.NewID(deviceChange.ID)
				}
			}
			ctx.Close()
		}
	}()
	return nil
}

// Stop stops the device change watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.ch != nil {
		close(w.ch)
	}
	w.mu.Unlock()
}

var _ controller.Watcher = &ChangeWatcher{}

// ChangeWatcher is a device change watcher
type ChangeWatcher struct {
	DeviceStore devicestore.Store
	ChangeStore devicechangestore.Store
	ch          chan<- controller.ID
	streams     map[devicetype.VersionedID]stream.Context
	cacheStream stream.Context
	mu          sync.Mutex
	wg          sync.WaitGroup
}

// Start starts the device change watcher
func (w *ChangeWatcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	if w.ch != nil {
		w.mu.Unlock()
		return nil
	}

	w.ch = ch
	w.streams = make(map[devicetype.VersionedID]stream.Context)
	w.mu.Unlock()

	deviceCh := make(chan *devicetopo.ListResponse)
	if err := w.DeviceStore.Watch(deviceCh); err != nil {
		return err
	}

	go func() {
		for response := range deviceCh {
			log.Infof("Received device event for device %v %v", response.Device.ID, response.Device.Version)
			deviceID := devicetype.NewVersionedID(devicetype.ID(response.Device.ID), devicetype.Version(response.Device.Version))
			w.watchDevice(deviceID, ch)
		}
	}()
	return nil
}

// watchDevice watches changes for the given device
func (w *ChangeWatcher) watchDevice(deviceID devicetype.VersionedID, ch chan<- controller.ID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ctx := w.streams[deviceID]
	if ctx != nil {
		log.Errorf("ChangeWatcher stream for %s is not nil", deviceID)
		return
	}

	deviceChangeCh := make(chan stream.Event, queueSize)
	ctx, err := w.ChangeStore.Watch(deviceID, deviceChangeCh, devicechangestore.WithReplay())
	if err != nil {
		log.Errorf("Setting up ChangeWatcher stream for %s: %s", deviceID, err)
		return
	}
	w.streams[deviceID] = ctx

	w.wg.Add(1)
	go func() {
		for event := range deviceChangeCh {
			ch <- controller.NewID(string(event.Object.(*devicechange.DeviceChange).ID))
		}
		w.wg.Done()
	}()
}

// Stop stops the device change watcher
func (w *ChangeWatcher) Stop() {
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

var _ controller.Watcher = &ChangeWatcher{}
