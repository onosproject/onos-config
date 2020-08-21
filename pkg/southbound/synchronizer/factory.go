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
	"fmt"
	"math"
	syncPrimitives "sync"
	"time"

	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/utils"
	topodevice "github.com/onosproject/onos-topo/api/device"
)

var connectionMonitors = make(map[topodevice.ID]*connectionMonitor)
var connections = make(map[topodevice.ID]bool)

const backoffInterval = 10 * time.Millisecond
const maxBackoffTime = 5 * time.Second

// Factory device factory data structures
type Factory struct {
	topoChannel               <-chan *topodevice.ListResponse
	opStateChan               chan<- events.OperationalStateEvent
	southboundErrorChan       chan<- events.DeviceResponse
	dispatcher                *dispatcher.Dispatcher
	modelRegistry             *modelregistry.ModelRegistry
	operationalStateCache     map[topodevice.ID]devicechange.TypedValueMap
	newTargetFn               func() southbound.TargetIf
	operationalStateCacheLock *syncPrimitives.RWMutex
	deviceChangeStore         device.Store
}

// NewFactory create a new factory
func NewFactory(options ...func(*Factory)) (*Factory, error) {
	factory := &Factory{}

	for _, option := range options {
		option(factory)
	}

	// TODO do some checks to make sure the data structures initialized properly

	return factory, nil

}

// WithTopoChannel sets factory topo channel
func WithTopoChannel(topoChannel <-chan *topodevice.ListResponse) func(*Factory) {
	return func(factory *Factory) {
		factory.topoChannel = topoChannel
	}
}

// WithOpStateChannel sets factory opStateChannel
func WithOpStateChannel(opStateChan chan<- events.OperationalStateEvent) func(*Factory) {
	return func(factory *Factory) {
		factory.opStateChan = opStateChan
	}
}

// WithSouthboundErrChan sets factory southbound error channel
func WithSouthboundErrChan(southboundErrorChan chan<- events.DeviceResponse) func(*Factory) {
	return func(factory *Factory) {
		factory.southboundErrorChan = southboundErrorChan
	}
}

// WithDispatcher sets factory dispatcher
func WithDispatcher(dispatcher *dispatcher.Dispatcher) func(*Factory) {
	return func(factory *Factory) {
		factory.dispatcher = dispatcher
	}
}

// WithModelRegistry set factory model registry
func WithModelRegistry(modelRegistry *modelregistry.ModelRegistry) func(*Factory) {
	return func(factory *Factory) {
		factory.modelRegistry = modelRegistry
	}
}

// WithOperationalStateCache sets factory operational state cache
func WithOperationalStateCache(operationalStateCache map[topodevice.ID]devicechange.TypedValueMap) func(*Factory) {
	return func(factory *Factory) {
		factory.operationalStateCache = operationalStateCache
	}
}

// WithNewTargetFn sets factory southbound target function
func WithNewTargetFn(newTargetFn func() southbound.TargetIf) func(*Factory) {
	return func(factory *Factory) {
		factory.newTargetFn = newTargetFn
	}
}

// WithOperationalStateCacheLock sets factory operational state cache lock
func WithOperationalStateCacheLock(operationalStateCacheLock *syncPrimitives.RWMutex) func(*Factory) {
	return func(factory *Factory) {
		factory.operationalStateCacheLock = operationalStateCacheLock
	}
}

// WithDeviceChangeStore sets factory device change store
func WithDeviceChangeStore(deviceChangeStore device.Store) func(*Factory) {
	return func(factory *Factory) {
		factory.deviceChangeStore = deviceChangeStore
	}
}

// TopoEventHandler handle topo device events
func (f *Factory) TopoEventHandler() {
	errChan := make(chan events.DeviceResponse)
	for {
		select {
		case topoEvent, ok := <-f.topoChannel:
			if !ok {
				return
			}

			device := topoEvent.Device
			connMon, ok := connectionMonitors[device.ID]
			if !ok && topoEvent.Type != topodevice.ListResponse_REMOVED {
				log.Infof("Topo device %s %s", device.ID, topoEvent.Type)
				connMon = &connectionMonitor{
					opStateChan:               f.opStateChan,
					southboundErrorChan:       errChan,
					dispatcher:                f.dispatcher,
					modelRegistry:             f.modelRegistry,
					operationalStateCache:     f.operationalStateCache,
					operationalStateCacheLock: f.operationalStateCacheLock,
					deviceChangeStore:         f.deviceChangeStore,
					device:                    device,
					target:                    f.newTargetFn(),
				}
				connectionMonitors[device.ID] = connMon
				go connMon.connect()
			} else if ok && topoEvent.Type == topodevice.ListResponse_UPDATED {
				changed := false
				if connMon.device.Address != topoEvent.Device.Address {
					oldAddress := connMon.device.Address
					connMon.device.Address = topoEvent.Device.Address
					changed = true
					log.Infof("Topo device %s is being UPDATED - waiting to complete", device.ID)
					connMon.close()
					// TODO Change grpc.DialContext() used to non blocking so that we can
					//  close the connection right away See https://github.com/onosproject/onos-config/issues/981
					waitTime := *connMon.device.GetTimeout() //Use the old timeout in case it has changed
					if maxBackoffTime > waitTime {
						waitTime = maxBackoffTime
					}
					time.Sleep(waitTime + time.Millisecond*20) // close might not take effect until timeout
					go connMon.reconnect()
					log.Infof("Topo device %s UPDATED address %s -> %s ", device.ID, oldAddress, topoEvent.Device.Address)
				}
				if connMon.device.Timeout.String() != topoEvent.Device.Timeout.String() {
					connMon.mu.Lock()
					oldTimeout := connMon.device.Timeout
					connMon.device.Timeout = topoEvent.Device.Timeout
					changed = true
					connMon.mu.Unlock()
					log.Infof("Topo device %s UPDATED timeout %s -> %s ", device.ID, oldTimeout, topoEvent.Device.Timeout)
				}
				if len(connMon.device.Protocols) != len(topoEvent.Device.Protocols) {
					// Ignoring any topo protocol updates - we set the gNMI one and
					// Don't really care about the others
					changed = true
				}
				if !changed {
					log.Infof("Topo device %s UPDATE not supported %v", device.ID, device)
					f.southboundErrorChan <- events.NewErrorEventNoChangeID(
						events.EventTypeTopoUpdate, string(device.ID),
						fmt.Errorf("topo update event ignored %v", topoEvent))
				}
			} else if ok && topoEvent.Type == topodevice.ListResponse_REMOVED {
				log.Infof("Topo device %s is being REMOVED - waiting to complete", device.ID)
				delete(connectionMonitors, device.ID)
				delete(connections, device.ID)
				connMon.close()
				// TODO Change grpc.DialContext() used to non blocking so that we can
				//  close the connection right away See https://github.com/onosproject/onos-config/issues/981
				waitTime := time.Duration(math.Max(float64(*connMon.device.GetTimeout()), float64(maxBackoffTime)))
				time.Sleep(waitTime + 100*time.Millisecond)
				log.Infof("Topo device %s REMOVED after %s", device.ID, waitTime)
			} else {
				log.Warnf("Unhandled event from topo service %v", topoEvent)
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
				connMon, ok := connectionMonitors[deviceID]
				if ok && connections[deviceID] {
					connections[deviceID] = false
					go connMon.reconnect()
				}
			case events.EventTypeDeviceConnected:
				connections[deviceID] = true
			}
			f.southboundErrorChan <- event
		}
	}

}

// connectionMonitor reacts to device events to establish connections to the device
type connectionMonitor struct {
	opStateChan               chan<- events.OperationalStateEvent
	southboundErrorChan       chan<- events.DeviceResponse
	dispatcher                *dispatcher.Dispatcher
	modelRegistry             *modelregistry.ModelRegistry
	operationalStateCache     map[topodevice.ID]devicechange.TypedValueMap
	operationalStateCacheLock *syncPrimitives.RWMutex
	deviceChangeStore         device.Store
	device                    *topodevice.Device
	target                    southbound.TargetIf
	cancel                    context.CancelFunc
	closed                    bool
	mu                        syncPrimitives.RWMutex
}

func (cm *connectionMonitor) reconnect() {
	cm.mu.Lock()
	if cm.cancel != nil {
		cm.cancel()
		cm.cancel = nil
	}
	cm.mu.Unlock()
	cm.operationalStateCacheLock.Lock()
	delete(cm.operationalStateCache, cm.device.ID)
	cm.operationalStateCacheLock.Unlock()
	cm.connect()
}

func (cm *connectionMonitor) connect() {
	count := 0
	for {
		count++

		cm.mu.Lock()
		closed := cm.closed
		cm.mu.Unlock()

		if closed {
			return
		}

		err := cm.synchronize()
		if err != nil {
			backoffTime := time.Duration(math.Min(float64(backoffInterval)*math.Pow(2, float64(count)), float64(maxBackoffTime)))
			log.Infof("Failed to connect to %s. Retry after %v Attempt %d", cm.device.ID, backoffTime, count)
			time.Sleep(backoffTime)
		} else {
			return
		}
	}
}

// synchronize connects to the device for synchronization
func (cm *connectionMonitor) synchronize() error {
	ctx, cancel := context.WithCancel(context.Background())
	cm.mu.Lock()
	cm.cancel = cancel
	cm.mu.Unlock()

	cm.mu.RLock()
	log.Infof("Connecting to device %v", cm.device)
	modelName := utils.ToModelName(devicetype.Type(cm.device.Type), devicetype.Version(cm.device.Version))
	mReadOnlyPaths, ok := cm.modelRegistry.ModelReadOnlyPaths[modelName]
	if !ok {
		log.Warnf("Cannot check for read only paths for target %cm with %cm because "+
			"Model Plugin not available - continuing", cm.device.ID, cm.device.Version)
	}
	mStateGetMode := modelregistry.GetStateOpState // default
	mPlugin, ok := cm.modelRegistry.ModelPlugins[modelName]
	if !ok {
		log.Warnf("Cannot check for StateGetMode for target %cm with %cm because "+
			"Model Plugin not available - continuing", cm.device.ID, cm.device.Version)
	} else {
		mStateGetMode = modelregistry.GetStateMode(mPlugin.GetStateMode())
	}
	valueMap := make(devicechange.TypedValueMap)
	cm.operationalStateCacheLock.Lock()
	cm.operationalStateCache[cm.device.ID] = valueMap
	cm.operationalStateCacheLock.Unlock()
	cm.mu.RUnlock()

	sync, err := New(ctx, cm.device, cm.opStateChan, cm.southboundErrorChan,
		valueMap, mReadOnlyPaths, cm.target, mStateGetMode, cm.operationalStateCacheLock, cm.deviceChangeStore)
	if err != nil {
		log.Errorf("Error connecting to device %v: %v", cm.device, err)
		//unregistering the listener for changes to the device
		//unregistering the listener for changes to the device
		cm.dispatcher.UnregisterOperationalState(string(cm.device.ID))
		cm.operationalStateCacheLock.Lock()
		delete(cm.operationalStateCache, cm.device.ID)
		cm.operationalStateCacheLock.Unlock()
		return err
	}

	//spawning two go routines to propagate changes and to get operational state
	//go sync.syncConfigEventsToDevice(target, respChan)
	cm.southboundErrorChan <- events.NewDeviceConnectedEvent(events.EventTypeDeviceConnected, string(cm.device.ID))
	if sync.getStateMode == modelregistry.GetStateOpState {
		go sync.syncOperationalStateByPartition(ctx, cm.target, cm.southboundErrorChan)
	} else if sync.getStateMode == modelregistry.GetStateExplicitRoPaths ||
		sync.getStateMode == modelregistry.GetStateExplicitRoPathsExpandWildcards {
		go sync.syncOperationalStateByPaths(ctx, cm.target, cm.southboundErrorChan)
	}
	return nil
}

// close closes the synchronizer
func (cm *connectionMonitor) close() {
	cm.mu.Lock()
	cm.closed = true
	if cm.cancel != nil {
		cm.cancel()
		cm.cancel = nil
	}
	cm.mu.Unlock()
	cm.operationalStateCacheLock.Lock()
	delete(cm.operationalStateCache, cm.device.ID)
	cm.operationalStateCacheLock.Unlock()
}
