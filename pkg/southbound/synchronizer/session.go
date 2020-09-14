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
	"strconv"
	"sync"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/cluster"

	"github.com/cenkalti/backoff"

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
	backoffInterval     = 10 * time.Millisecond
	maxBackoffTime      = 5 * time.Second
	mastershipTermKey   = "onos-config.mastership.term"
	mastershipMasterKey = "onos-config.mastership.master"
)

// Session a gNMI session
type Session struct {
	deviceStore               devicestore.Store
	mastershipState           *mastership.Mastership
	nodeID                    cluster.NodeID
	connected                 bool
	opStateChan               chan<- events.OperationalStateEvent
	deviceResponseChan        chan events.DeviceResponse
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

func (s *Session) getCurrentTerm() (int, error) {
	device, err := s.deviceStore.Get(s.device.ID)
	if err != nil {
		return 0, err
	}

	term := device.Attributes[mastershipTermKey]
	if term == "" {
		return 0, nil
	}
	return strconv.Atoi(term)
}

// open open a new gNMI session
func (s *Session) open() error {
	s.deviceResponseChan = make(chan events.DeviceResponse)

	go func() {
		_ = s.updateDeviceState()

	}()
	go func() {
		s.mu.Lock()
		s.connected = false
		s.mu.Unlock()

		currentTerm, err := s.getCurrentTerm()
		if err != nil {
			log.Error(err)
		}

		if s.mastershipState.Master == s.nodeID && uint64(s.mastershipState.Term) >= uint64(currentTerm) {
			err := s.connect()
			if err != nil {
				log.Error(err)
			} else {
				s.mu.Lock()
				s.connected = true
				s.mu.Unlock()
			}
		}

	}()

	return nil
}

// connect connects to a device using a gNMI session
func (s *Session) connect() error {
	log.Info("Connecting to device:", s.device)
	count := 0
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = backoffInterval
	// MaxInterval caps the RetryInterval
	b.MaxInterval = maxBackoffTime
	// Never stops retrying
	b.MaxElapsedTime = 0

	notify := func(err error, t time.Duration) {
		count++
		log.Infof("Failed to connect to %s. Retry after %v Attempt %d", s.device.ID, b.GetElapsedTime(), count)
	}

	err := backoff.RetryNotify(s.synchronize, b, notify)
	if err != nil {
		return err
	}

	return nil

}

// synchronize connects to the device for synchronization
func (s *Session) synchronize() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	s.mu.RLock()
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

	sync, err := New(ctx, s.device, s.opStateChan, s.deviceResponseChan,
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
	s.deviceResponseChan <- events.NewDeviceConnectedEvent(events.EventTypeDeviceConnected, string(s.device.ID))
	if sync.getStateMode == modelregistry.GetStateOpState {
		go sync.syncOperationalStateByPartition(ctx, s.target, s.deviceResponseChan)
	} else if sync.getStateMode == modelregistry.GetStateExplicitRoPaths ||
		sync.getStateMode == modelregistry.GetStateExplicitRoPathsExpandWildcards {
		go sync.syncOperationalStateByPaths(ctx, s.target, s.deviceResponseChan)
	}
	return nil
}

// disconnects the gNMI session from the device
func (s *Session) disconnect() error {
	log.Info("Disconnecting device:", s.device)
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
	log.Info("Close session for device:", s.device)
	err := s.disconnect()
	if err != nil {
		log.Error(err)
	}

}
