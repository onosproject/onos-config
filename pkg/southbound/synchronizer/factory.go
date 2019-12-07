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

var listeners = make(map[topodevice.ID]*deviceListener)

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

	for topoEvent := range topoChannel {
		device := topoEvent.Device
		listener, ok := listeners[device.ID]
		if !ok {
			listener = &deviceListener{
				opStateChan:               opStateChan,
				southboundErrorChan:       southboundErrorChan,
				dispatcher:                dispatcher,
				modelRegistry:             modelRegistry,
				operationalStateCache:     operationalStateCache,
				newTargetFn:               newTargetFn,
				operationalStateCacheLock: operationalStateCacheLock,
				deviceChangeStore:         deviceChangeStore,
			}
			listeners[device.ID] = listener
		}
		listener.notify(device)
	}
}

// deviceListener reacts to device events to establish connections to the device
type deviceListener struct {
	opStateChan               chan<- events.OperationalStateEvent
	southboundErrorChan       chan<- events.DeviceResponse
	dispatcher                *dispatcher.Dispatcher
	modelRegistry             *modelregistry.ModelRegistry
	operationalStateCache     map[topodevice.ID]devicechange.TypedValueMap
	newTargetFn               func() southbound.TargetIf
	operationalStateCacheLock *syncPrimitives.RWMutex
	deviceChangeStore         device.Store
	state                     *topodevice.ProtocolState
	sync                      *deviceSynchronizer
}

// getDeviceState returns the gNMI state for the given device
func getDeviceState(device *topodevice.Device) *topodevice.ProtocolState {
	for _, state := range device.Protocols {
		if state.Protocol == topodevice.Protocol_GNMI {
			return state
		}
	}
	return nil
}

// notify notifies the device synchronizer of a device change
func (s *deviceListener) notify(device *topodevice.Device) {
	state := getDeviceState(device)
	if s.state == nil || (s.state.ChannelState != state.ChannelState && state.ChannelState == topodevice.ChannelState_DISCONNECTED) {
		if s.sync != nil {
			s.sync.close()
		}
		sync := &deviceSynchronizer{
			opStateChan:               s.opStateChan,
			southboundErrorChan:       s.southboundErrorChan,
			dispatcher:                s.dispatcher,
			modelRegistry:             s.modelRegistry,
			operationalStateCache:     s.operationalStateCache,
			newTargetFn:               s.newTargetFn,
			operationalStateCacheLock: s.operationalStateCacheLock,
			deviceChangeStore:         s.deviceChangeStore,
			device:                    device,
		}
		s.sync = sync
		go sync.connect()
	}
	s.state = state
}

// deviceSynchronizer reacts to device events to establish connections to the device
type deviceSynchronizer struct {
	opStateChan               chan<- events.OperationalStateEvent
	southboundErrorChan       chan<- events.DeviceResponse
	dispatcher                *dispatcher.Dispatcher
	modelRegistry             *modelregistry.ModelRegistry
	operationalStateCache     map[topodevice.ID]devicechange.TypedValueMap
	newTargetFn               func() southbound.TargetIf
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

		target, err := s.synchronize()
		if err != nil {
			backoffTime := time.Duration(math.Min(float64(backoffInterval)*float64(2^count), float64(maxBackoffTime)))
			time.Sleep(backoffTime)
		} else {
			s.mu.Lock()
			s.target = target
			closed := s.closed
			s.mu.Unlock()
			if closed {
				_ = target.Client().Close()
			}
			return
		}
	}
}

// synchronize connects to the device for synchronization
func (s *deviceSynchronizer) synchronize() (southbound.TargetIf, error) {
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
	target := s.newTargetFn()

	ctx := context.Background()
	sync, err := New(ctx, s.device, s.opStateChan, s.southboundErrorChan,
		valueMap, mReadOnlyPaths, target, mStateGetMode, s.operationalStateCacheLock, s.deviceChangeStore)
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
		return nil, err
	}

	//spawning two go routines to propagate changes and to get operational state
	//go sync.syncConfigEventsToDevice(target, respChan)
	if sync.getStateMode == modelregistry.GetStateOpState {
		go sync.syncOperationalStateByPartition(ctx, target, s.southboundErrorChan)
	} else if sync.getStateMode == modelregistry.GetStateExplicitRoPaths ||
		sync.getStateMode == modelregistry.GetStateExplicitRoPathsExpandWildcards {
		go sync.syncOperationalStateByPaths(ctx, target, s.southboundErrorChan)
	}
	s.southboundErrorChan <- events.NewDeviceConnectedEvent(events.EventTypeDeviceConnected, string(s.device.ID))
	return target, nil
}

// close closes the synchronizer
func (s *deviceSynchronizer) close() {
	s.mu.Lock()
	if s.target != nil {
		_ = s.target.Client().Close()
	}
	s.closed = true
	s.mu.Unlock()
}
