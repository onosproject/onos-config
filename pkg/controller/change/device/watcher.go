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
	"github.com/onosproject/onos-config/api/types"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/controller"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"sync"
)

const queueSize = 100

// Watcher is a device change watcher
type Watcher struct {
	DeviceCache cache.Cache
	ChangeStore devicechangestore.Store
	ch          chan<- types.ID
	streams     map[devicetype.VersionedID]stream.Context
	cacheStream stream.Context
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
	w.streams = make(map[devicetype.VersionedID]stream.Context)
	w.mu.Unlock()

	deviceCacheCh := make(chan stream.Event)
	go func() {
		for eventObj := range deviceCacheCh {
			// TODO: Handle device deletes
			if eventObj.Type == stream.None || eventObj.Type == stream.Created {
				event := eventObj.Object.(*cache.Info)
				log.Infof("Received device event for device %v %v", event.DeviceID, event.Version)
				w.watchDevice(devicetype.NewVersionedID(event.DeviceID, event.Version), ch)
			}
		}
		w.mu.Lock()
		w.cacheStream.Close()
		w.mu.Unlock()
	}()

	var err error
	w.mu.Lock()
	w.cacheStream, err = w.DeviceCache.Watch(deviceCacheCh, true)
	w.mu.Unlock()
	if err != nil {
		return err
	}

	return nil
}

// watchDevice watches changes for the given device
func (w *Watcher) watchDevice(deviceID devicetype.VersionedID, ch chan<- types.ID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ctx := w.streams[deviceID]
	if ctx != nil {
		log.Errorf("Watcher stream for %s is not nil", deviceID)
		return
	}

	deviceChangeCh := make(chan stream.Event, queueSize)
	ctx, err := w.ChangeStore.Watch(deviceID, deviceChangeCh, devicechangestore.WithReplay())
	if err != nil {
		log.Errorf("Setting up Watcher stream for %s: %s", deviceID, err)
		return
	}
	w.streams[deviceID] = ctx

	w.wg.Add(1)
	go func() {
		for event := range deviceChangeCh {
			ch <- types.ID(event.Object.(*devicechange.DeviceChange).ID)
		}
		w.wg.Done()
	}()
}

// Stop stops the device change watcher
func (w *Watcher) Stop() {
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

var _ controller.Watcher = &Watcher{}
