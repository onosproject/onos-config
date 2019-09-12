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
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
	"time"
)

// Factory is a go routine thread that listens out for Device creation
// and deletion events and spawns Synchronizer threads for them
// These synchronizers then listen out for configEvents relative to a device and
func Factory(changeStore *store.ChangeStore, configStore *store.ConfigurationStore,
	topoChannel <-chan events.TopoEvent, opStateChan chan<- events.OperationalStateEvent,
	errChan chan<- events.DeviceResponse, dispatcher *dispatcher.Dispatcher,
	readOnlyPaths map[string]modelregistry.ReadOnlyPathMap, operationalStateCache map[devicepb.ID]change.TypedValueMap) {
	for topoEvent := range topoChannel {
		device := events.Event(topoEvent).Object().(devicepb.Device)
		if !dispatcher.HasListener(device.ID) && topoEvent.Connect() {
			configChan, respChan, err := dispatcher.RegisterDevice(device.ID)
			if err != nil {
				log.Error(err)
			}
			ctx := context.Background()
			configName := store.ConfigName(utils.ToConfigName(device.ID, device.Version))
			cfg, ok := configStore.Store[configName]
			if !ok {
				if device.Type == "" {
					log.Warningf("No device type specified for device %s", configName)
				}
				cfg = store.Configuration{
					Name:    configName,
					Device:  string(device.ID),
					Version: device.Version,
					Type:    string(device.Type),
					Created: time.Now(),
					Updated: time.Now(),
					Changes: []change.ID{},
				}
				configStore.Store[configName] = cfg
			}

			modelName := utils.ToModelName(cfg.Type, device.Version)
			mReadOnlyPaths, ok := readOnlyPaths[modelName]
			if !ok {
				log.Warningf("Cannot check for read only paths for target %s with %s because "+
					"Model Plugin not available - continuing", device.ID, device.Version)
			}
			operationalStateCache[device.ID] = make(change.TypedValueMap)
			sync, err := New(ctx, changeStore, configStore, &device, configChan, opStateChan,
				errChan, operationalStateCache[device.ID], mReadOnlyPaths)
			if err != nil {
				log.Error("Error in connecting to client: ", err)
				errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceConnect,
					string(device.ID), err)
				//unregistering the listener for changes to the device
				unregErr := dispatcher.UnregisterDevice(device.ID)
				if unregErr != nil {
					errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceDisconnect,
						string(device.ID), unregErr)
				}
			} else {
				//spawning two go routines to propagate changes and to get operational state
				go sync.syncConfigEventsToDevice(respChan)
				go sync.syncOperationalState(errChan)
				//respChan <- events.CreateConnectedEvent(events.EventTypeDeviceConnected, string(deviceName))
			}
		} else if dispatcher.HasListener(device.ID) && !topoEvent.Connect() {

			err := dispatcher.UnregisterDevice(device.ID)
			if err != nil {
				log.Error(err)
				//TODO evaluate if fall through without upstreaming
				//errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceDisconnect,
				//	string(deviceName), err)
			}
		}
	}
}
