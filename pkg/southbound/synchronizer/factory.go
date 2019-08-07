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
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	topopb "github.com/onosproject/onos-topo/pkg/northbound/proto"
	log "k8s.io/klog"
)

// Factory is a go routine thread that listens out for Device creation
// and deletion events and spawns Synchronizer threads for them
// These synchronizers then listen out for configEvents relative to a device and
func Factory(changeStore *store.ChangeStore, configStore *store.ConfigurationStore, deviceStore *topocache.DeviceStore,
	topoChannel <-chan events.TopoEvent, opStateChan chan<- events.OperationalStateEvent,
	errChan chan<- events.DeviceResponse, dispatcher *dispatcher.Dispatcher,
	readOnlyPaths map[string]modelregistry.ReadOnlyPathMap, operationalStateCache map[string]change.TypedValueMap) {
	for topoEvent := range topoChannel {
		deviceName := events.Event(topoEvent).Subject()
		if !dispatcher.HasListener(deviceName) && topoEvent.Connect() {
			configChan, respChan, err := dispatcher.RegisterDevice(deviceName)
			if err != nil {
				log.Error(err)
			}
			device := events.Event(topoEvent).Object().(*topopb.Device)
			ctx := context.Background()
			completeID := utils.ToConfigName(deviceName, device.SoftwareVersion)
			cfg := configStore.Store[store.ConfigName(completeID)]
			modelName := utils.ToModelName(cfg.Type, device.SoftwareVersion)
			mReadOnlyPaths, ok := readOnlyPaths[modelName]
			if !ok {
				log.Warningf("Cannot check for read only paths for target %s with %s because "+
					"Model Plugin not available - continuing", deviceName, device.SoftwareVersion)
			}
			operationalStateCache[deviceName] = make(change.TypedValueMap)
			sync, err := New(ctx, changeStore, configStore, device, configChan, opStateChan,
				errChan, operationalStateCache[deviceName], mReadOnlyPaths)
			if err != nil {
				log.Error("Error in connecting to client: ", err)
				errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceConnect,
					string(deviceName), err)
				//unregistering the listener for changes to the device
				unregErr := dispatcher.UnregisterDevice(deviceName)
				if unregErr != nil {
					errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceDisconnect,
						string(deviceName), unregErr)
				}
			} else {
				//spawning two go routines to propagate changes and to get operational state
				go sync.syncConfigEventsToDevice(respChan)
				go sync.syncOperationalState(errChan)
				//respChan <- events.CreateConnectedEvent(events.EventTypeDeviceConnected, string(deviceName))
			}
		} else if dispatcher.HasListener(deviceName) && !topoEvent.Connect() {

			err := dispatcher.UnregisterDevice(deviceName)
			if err != nil {
				log.Error(err)
				//TODO evaluate if fall through without upstreaming
				//errChan <- events.CreateErrorEventNoChangeID(events.EventTypeErrorDeviceDisconnect,
				//	string(deviceName), err)
			}
		}
	}
}
