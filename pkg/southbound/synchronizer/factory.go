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

package synchronizer

import (
	"context"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	log "k8s.io/klog"
)

// Factory is a go routine thread that listens out for Device creation
// and deletion events and spawns Synchronizer threads for them
// These synchronizers then listen out for configEvents relative to a device and
func Factory(changeStore *store.ChangeStore, configStore *store.ConfigurationStore, deviceStore *topocache.DeviceStore,
	topoChannel <-chan events.TopoEvent, opStateChan chan<- events.OperationalStateEvent,
	errChan chan<- events.ErrorEvent, dispatcher *dispatcher.Dispatcher) {
	for topoEvent := range topoChannel {
		deviceName := topocache.ID(events.Event(topoEvent).Subject())
		if !dispatcher.HasListener(deviceName) && topoEvent.Connect() {
			configChan, err := dispatcher.RegisterDevice(deviceName)
			if err != nil {
				log.Error(err)
			}
			device := deviceStore.Store[topocache.ID(deviceName)]
			ctx := context.Background()
			sync, err := New(ctx, changeStore, configStore, &device, configChan, opStateChan, errChan)
			if err != nil {
				log.Error("Error in connecting to client: ", err)
				errChan <- events.CreateErrorEvent(events.EventTypeErrorDeviceConnect,
					string(deviceName), make([]byte, 0), err)
				//unregistering the listener for changes to the device
				unregErr := dispatcher.UnregisterDevice(deviceName)
				if unregErr != nil {
					errChan <- events.CreateErrorEvent(events.EventTypeErrorDeviceDisconnect,
						string(deviceName), make([]byte, 0), unregErr)
				}
			} else {
				//spawning two go routines to propagate changes and to get operational state
				go sync.syncConfigEventsToDevice(errChan)
				go sync.syncOperationalState(errChan)
			}
		} else if dispatcher.HasListener(deviceName) && !topoEvent.Connect() {

			err := dispatcher.UnregisterDevice(deviceName)
			if err != nil {
				log.Error(err)
				errChan <- events.CreateErrorEvent(events.EventTypeErrorDeviceDisconnect,
					string(deviceName), make([]byte, 0), err)
			}
		}
	}
}
