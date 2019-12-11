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
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/utils"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"math"
	syncPrimitives "sync"
	"time"
)

var synchronizers = make(map[topodevice.ID]*deviceSynchronizer)
var connections = make(map[topodevice.ID]bool)

const backoffInterval = 10 * time.Millisecond
const maxBackoffTime = 5 * time.Second

// Factory is a go routine thread that listens out for Device creation
// and deletion events and spawns Synchronizer threads for them
// These synchronizers then listen out for configEvents relative to a device and
func Factory(topoChannel <-chan *topodevice.ListResponse, opStateChan chan<- events.OperationalStateEvent,
	southboundErrorChan chan<- events.DeviceResponse, dispatcher *dispatcher.Dispatcher,
	modelRegistry *modelregistry.ModelRegistry, operationalStateCache map[topodevice.ID]devicechange.TypedValueMap,
	newTargetFn func() southbound.TargetIf,
	operationalStateCacheLock *syncPrimitives.RWMutex, deviceChangeStore device.Store) {

	errChan := make(chan events.DeviceResponse)
	for {
		select {
		case topoEvent, ok := <-topoChannel:
			if !ok {
				return
			}

			device := topoEvent.Device
			log.Infof("Received device event %v", device)
			_, ok = synchronizers[device.ID]
			if !ok {
				synchronizer := &deviceSynchronizer{
					opStateChan:               opStateChan,
					southboundErrorChan:       errChan,
					dispatcher:                dispatcher,
					modelRegistry:             modelRegistry,
					operationalStateCache:     operationalStateCache,
					operationalStateCacheLock: operationalStateCacheLock,
					deviceChangeStore:         deviceChangeStore,
					device:                    device,
					target:                    newTargetFn(),
				}
				synchronizers[device.ID] = synchronizer
				go synchronizer.connect()
			}
		case event, ok := <-errChan:
			if !ok {
				return
			}

			log.Infof("Received event %v", event)
			deviceID := topodevice.ID(event.Subject())
			switch event.EventType() {
			case events.EventTypeErrorDeviceConnect:
				deviceID := topodevice.ID(event.Subject())
				synchronizer, ok := synchronizers[deviceID]
				if ok && connections[deviceID] {
					synchronizer.close()
					synchronizer = &deviceSynchronizer{
						opStateChan:               synchronizer.opStateChan,
						southboundErrorChan:       synchronizer.southboundErrorChan,
						dispatcher:                synchronizer.dispatcher,
						modelRegistry:             synchronizer.modelRegistry,
						operationalStateCache:     synchronizer.operationalStateCache,
						operationalStateCacheLock: synchronizer.operationalStateCacheLock,
						deviceChangeStore:         synchronizer.deviceChangeStore,
						device:                    synchronizer.device,
						target:                    synchronizer.target,
					}
					synchronizers[deviceID] = synchronizer
					connections[deviceID] = false
					log.Info("Retrying connecting to device %s ", deviceID)
					go synchronizer.connect()
				}
			case events.EventTypeDeviceConnected:
				connections[deviceID] = true
			}
			southboundErrorChan <- event
		}
	}
}

// deviceSynchronizer reacts to device events to establish connections to the device
type deviceSynchronizer struct {
	opStateChan               chan<- events.OperationalStateEvent
	southboundErrorChan       chan<- events.DeviceResponse
	dispatcher                *dispatcher.Dispatcher
	modelRegistry             *modelregistry.ModelRegistry
	operationalStateCache     map[topodevice.ID]devicechange.TypedValueMap
	operationalStateCacheLock *syncPrimitives.RWMutex
	deviceChangeStore         device.Store
	device                    *topodevice.Device
	target                    southbound.TargetIf
	closed                    bool
	mu                        syncPrimitives.Mutex
}

func (s *deviceSynchronizer) connect() {
	count := 0
	for {
		count++

		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()

		if closed {
			return
		}

		err := s.synchronize()
		if err != nil {
			backoffTime := time.Duration(math.Min(float64(backoffInterval)*float64(2^count), float64(maxBackoffTime)))
			time.Sleep(backoffTime)
		} else {
			return
		}
	}
}

// synchronize connects to the device for synchronization
func (s *deviceSynchronizer) synchronize() error {
	log.Infof("Connecting to device %v", s.device)
	modelName := utils.ToModelName(devicetype.Type(s.device.Type), devicetype.Version(s.device.Version))
	mReadOnlyPaths, ok := s.modelRegistry.ModelReadOnlyPaths[modelName]
	if !ok {
		log.Warnf("Cannot check for read only paths for target %s with %s because "+
			"Model Plugin not available - continuing", s.device.ID, s.device.Version)
	}
	mStateGetMode := modelregistry.GetStateOpState // default
	mPlugin, ok := s.modelRegistry.ModelPlugins[modelName]
	if !ok {
		log.Warnf("Cannot check for StateGetMode for target %s with %s because "+
			"Model Plugin not available - continuing", s.device.ID, s.device.Version)
	} else {
		mStateGetMode = modelregistry.GetStateMode(mPlugin.GetStateMode())
	}
	valueMap := make(devicechange.TypedValueMap)
	s.operationalStateCacheLock.Lock()
	s.operationalStateCache[s.device.ID] = valueMap
	s.operationalStateCacheLock.Unlock()

	ctx := context.Background()
	sync, err := New(ctx, s.device, s.opStateChan, s.southboundErrorChan,
		valueMap, mReadOnlyPaths, s.target, mStateGetMode, s.operationalStateCacheLock, s.deviceChangeStore)
	if err != nil {
		log.Errorf("Error connecting to device %v: %v", s.device, err)
		s.southboundErrorChan <- events.NewErrorEventNoChangeID(events.EventTypeErrorDeviceConnect,
			string(s.device.ID), err)
		//unregistering the listener for changes to the device
		//unregistering the listener for changes to the device
		s.dispatcher.UnregisterOperationalState(string(s.device.ID))
		s.operationalStateCacheLock.Lock()
		delete(s.operationalStateCache, s.device.ID)
		s.operationalStateCacheLock.Unlock()
		return err
	}

	//spawning two go routines to propagate changes and to get operational state
	//go sync.syncConfigEventsToDevice(target, respChan)
	s.southboundErrorChan <- events.NewDeviceConnectedEvent(events.EventTypeDeviceConnected, string(s.device.ID))
	if sync.getStateMode == modelregistry.GetStateOpState {
		go sync.syncOperationalStateByPartition(ctx, s.target, s.southboundErrorChan)
	} else if sync.getStateMode == modelregistry.GetStateExplicitRoPaths ||
		sync.getStateMode == modelregistry.GetStateExplicitRoPathsExpandWildcards {
		go sync.syncOperationalStateByPaths(ctx, s.target, s.southboundErrorChan)
	}
	return nil
}

// close closes the synchronizer
func (s *deviceSynchronizer) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}
