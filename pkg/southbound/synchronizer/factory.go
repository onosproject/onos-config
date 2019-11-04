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
	"github.com/onosproject/onos-config/pkg/southbound"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	devicetype "github.com/onosproject/onos-config/pkg/types/device"
	"github.com/onosproject/onos-config/pkg/utils"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
)

// Factory is a go routine thread that listens out for Device creation
// and deletion events and spawns Synchronizer threads for them
// These synchronizers then listen out for configEvents relative to a device and
func Factory(topoChannel <-chan *devicetopo.ListResponse, opStateChan chan<- events.OperationalStateEvent,
	southboundErrorChan chan<- events.DeviceResponse, dispatcher *dispatcher.Dispatcher,
	modelRegistry *modelregistry.ModelRegistry, operationalStateCache map[devicetopo.ID]devicechangetypes.TypedValueMap) {

	for topoEvent := range topoChannel {
		notifiedDevice := topoEvent.Device
		//TODO evaluate the need for fine-grained checking
		if topoEvent.Type == devicetopo.ListResponse_ADDED {
			ctx := context.Background()

			modelName := utils.ToModelName(devicetype.Type(notifiedDevice.Type), devicetype.Version(notifiedDevice.Version))
			mReadOnlyPaths, ok := modelRegistry.ModelReadOnlyPaths[modelName]
			if !ok {
				log.Warningf("Cannot check for read only paths for target %s with %s because "+
					"Model Plugin not available - continuing", notifiedDevice.ID, notifiedDevice.Version)
			}
			mStateGetMode := modelregistry.GetStateOpState // default
			mPlugin, ok := modelRegistry.ModelPlugins[modelName]
			if !ok {
				log.Warningf("Cannot check for StateGetMode for target %s with %s because "+
					"Model Plugin not available - continuing", notifiedDevice.ID, notifiedDevice.Version)
			} else {
				mStateGetMode = modelregistry.GetStateMode(mPlugin.GetStateMode())
			}
			operationalStateCache[notifiedDevice.ID] = make(devicechangetypes.TypedValueMap)
			target := southbound.NewTarget()
			sync, err := New(ctx, notifiedDevice, opStateChan, southboundErrorChan,
				operationalStateCache[notifiedDevice.ID], mReadOnlyPaths, target, mStateGetMode)
			if err != nil {
				log.Errorf("Error connecting to device %v: %v", notifiedDevice, err)
				southboundErrorChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorDeviceConnect,
					string(notifiedDevice.ID), err)
				//unregistering the listener for changes to the device
				//unregistering the listener for changes to the device
				dispatcher.UnregisterOperationalState(string(notifiedDevice.ID))
				delete(operationalStateCache, notifiedDevice.ID)
			} else {
				//spawning two go routines to propagate changes and to get operational state
				//go sync.syncConfigEventsToDevice(target, respChan)
				if sync.getStateMode == modelregistry.GetStateOpState {
					go sync.syncOperationalStateByPartition(ctx, target, southboundErrorChan)
				} else if sync.getStateMode == modelregistry.GetStateExplicitRoPaths ||
					sync.getStateMode == modelregistry.GetStateExplicitRoPathsExpandWildcards {
					go sync.syncOperationalStateByPaths(ctx, target, southboundErrorChan)
				}
				southboundErrorChan <- events.NewDeviceConnectedEvent(events.EventTypeDeviceConnected, string(notifiedDevice.ID))
			}
		}
	}
}
