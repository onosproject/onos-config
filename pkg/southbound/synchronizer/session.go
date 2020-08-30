// Copyright 2020-present Open Networking Foundation.
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
	"math"
	"sync"
	"time"

	"github.com/onosproject/onos-config/pkg/utils"

	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"

	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/mastership"

	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	topodevice "github.com/onosproject/onos-topo/api/device"
)

const (
	backoffInterval = 10 * time.Millisecond
	maxBackoffTime  = 5 * time.Second
)

// Session a gNMI session
type Session struct {
	mastershipStore           mastership.Store
	deviceStore               devicestore.Store
	mastershipState           *mastership.Mastership
	closeCh                   chan struct{}
	opStateChan               chan<- events.OperationalStateEvent
	southboundErrorChan       chan events.DeviceResponse
	dispatcher                *dispatcher.Dispatcher
	modelRegistry             *modelregistry.ModelRegistry
	operationalStateCache     map[topodevice.ID]devicechange.TypedValueMap
	operationalStateCacheLock *sync.RWMutex
	deviceChangeStore         device.Store
	device                    *topodevice.Device
	target                    southbound.TargetIf
	cancel                    context.CancelFunc
	closed                    bool
	mu                        sync.RWMutex
}

// open open a new gNMI session
func (s *Session) open() error {
	log.Info("Opening a gNMI session")
	ch := make(chan mastership.Mastership)
	err := s.mastershipStore.Watch(s.device.ID, ch)
	if err != nil {
		return err
	}

	go func() {
		err := s.updateDeviceState()
		if err != nil {
			return
		}
	}()

	go func() {
		connected := false
		state, _ := s.mastershipStore.GetMastership(s.device.ID)
		if state != nil {
			s.mu.Lock()
			s.mastershipState = state
			s.mu.Unlock()
			if state.Master == s.mastershipStore.NodeID() {
				log.Info("Master node", s.mastershipStore.NodeID())
				err := s.connect()
				if err != nil {
					log.Error(err)
				} else {
					connected = true
				}
			}

		}

		for {
			select {
			case state := <-ch:
				s.mu.Lock()
				s.mastershipState = &state
				s.mu.Unlock()
				if state.Master == s.mastershipStore.NodeID() && !connected {
					log.Info("Election changed", s.mastershipStore.NodeID())
					err := s.connect()
					if err != nil {
						log.Error(err)
					} else {
						connected = true
					}
				} else if state.Master != s.mastershipStore.NodeID() && connected {
					log.Info("It is not master then disconnect")
					err := s.disconnect()
					if err != nil {
						log.Error(err)
					} else {
						connected = false
					}
				}
			case <-s.closeCh:
				return
			}
		}
	}()

	return nil
}

// connect connects to a device using a gNMI session
func (s *Session) connect() error {
	log.Info("Connecting to device:", s.device.ID)
	count := 0
	for {
		count++

		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()

		if closed {
			return nil
		}

		err := s.synchronize()
		if err != nil {
			backoffTime := time.Duration(math.Min(float64(backoffInterval)*math.Pow(2, float64(count)), float64(maxBackoffTime)))
			log.Infof("Failed to connect to %s. Retry after %v Attempt %d", s.device.ID, backoffTime, count)
			time.Sleep(backoffTime)
		} else {
			return nil
		}
	}

}

// synchronize connects to the device for synchronization
func (s *Session) synchronize() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	s.mu.RLock()
	log.Infof("Connecting to device %v", s.device)
	modelName := utils.ToModelName(devicetype.Type(s.device.Type), devicetype.Version(s.device.Version))
	mReadOnlyPaths, ok := s.modelRegistry.ModelReadOnlyPaths[modelName]
	if !ok {
		log.Warnf("Cannot check for read only paths for target %cm with %cm because "+
			"Model Plugin not available - continuing", s.device.ID, s.device.Version)
	}
	mStateGetMode := modelregistry.GetStateOpState // default
	mPlugin, ok := s.modelRegistry.ModelPlugins[modelName]
	if !ok {
		log.Warnf("Cannot check for StateGetMode for target %cm with %cm because "+
			"Model Plugin not available - continuing", s.device.ID, s.device.Version)
	} else {
		mStateGetMode = modelregistry.GetStateMode(mPlugin.GetStateMode())
	}
	valueMap := make(devicechange.TypedValueMap)
	s.operationalStateCacheLock.Lock()
	s.operationalStateCache[s.device.ID] = valueMap
	s.operationalStateCacheLock.Unlock()
	s.mu.RUnlock()

	sync, err := New(ctx, s.device, s.opStateChan, s.southboundErrorChan,
		valueMap, mReadOnlyPaths, s.target, mStateGetMode, s.operationalStateCacheLock, s.deviceChangeStore)
	if err != nil {
		log.Errorf("Error connecting to device %v: %v", s.device, err)
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

// disconnects the gNMI session from the device
func (s *Session) disconnect() error {
	log.Info("Disconnecting device:", s.device.ID)
	s.mu.Lock()
	s.closed = true
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.mu.Unlock()
	s.operationalStateCacheLock.Lock()
	delete(s.operationalStateCache, s.device.ID)
	s.operationalStateCacheLock.Unlock()
	return nil
}

// Close close a gNMI session
func (s *Session) Close() {
	log.Info("Close session for device:", s.device.ID)
	err := s.disconnect()
	if err != nil {
		log.Error(err)
	}
	if s.closeCh != nil {
		close(s.closeCh)
	}
}
