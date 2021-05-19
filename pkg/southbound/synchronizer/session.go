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
	configmodel "github.com/onosproject/onos-config-model/pkg/model"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"sync"
	"time"

	"github.com/onosproject/onos-lib-go/pkg/cluster"

	"github.com/cenkalti/backoff"

	"github.com/onosproject/onos-config/pkg/utils"

	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"

	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/mastership"

	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/store/change/device"
)

const (
	backoffInterval = 10 * time.Millisecond
	maxBackoffTime  = 5 * time.Second
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

func (s *Session) getCurrentTerm() (uint64, error) {
	d, err := s.deviceStore.Get(s.device.ID)
	if err != nil {
		return 0, err
	}

	return d.MastershipTerm, nil
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

		if s.mastershipState.Master == s.nodeID && uint64(s.mastershipState.Term) >= currentTerm {
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
	log.Infof("Connecting to device: %s:%s at %s", s.device.ID, s.device.Version, s.device.Address)
	count := 0
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = backoffInterval
	// MaxInterval caps the RetryInterval
	b.MaxInterval = maxBackoffTime
	// Never stops retrying
	b.MaxElapsedTime = 0

	notify := func(err error, t time.Duration) {
		count++
		log.Infof("Failed to connect to %s:%s. Retry after %v Attempt %d",
			s.device.ID, s.device.Version, b.GetElapsedTime(), count)
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
	plugin, err := s.modelRegistry.GetPlugin(modelName)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warnf("Model Plugin not available for device %s:%s", s.device.ID, s.device.Version)
		} else {
			s.mu.RUnlock()
			log.Error(err)
			return err
		}
	}
	var mReadOnlyPaths modelregistry.ReadOnlyPathMap
	mStateGetMode := configmodel.GetStateOpState // default
	if plugin != nil {
		mReadOnlyPaths = plugin.ReadOnlyPaths
		pluginStateGetMode := plugin.Model.GetStateMode()
		if pluginStateGetMode != configmodel.GetStateNone {
			mStateGetMode = pluginStateGetMode
		}
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
	if sync.getStateMode == configmodel.GetStateOpState {
		go sync.syncOperationalStateByPartition(ctx, s.deviceResponseChan)
	} else if sync.getStateMode == configmodel.GetStateExplicitRoPaths ||
		sync.getStateMode == configmodel.GetStateExplicitRoPathsExpandWildcards {
		go sync.syncOperationalStateByPaths(ctx, s.deviceResponseChan)
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
