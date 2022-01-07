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
	"sync"

	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/mastership"
)

// SessionManager is a gNMI session manager
type SessionManager struct {
	topoChannel               chan *topodevice.ListResponse
	opStateChan               chan<- events.OperationalStateEvent
	deviceStore               devicestore.Store
	closeCh                   chan struct{}
	dispatcher                *dispatcher.Dispatcher
	modelRegistry             *modelregistry.ModelRegistry
	sessions                  map[topodevice.ID]*Session
	operationalStateCache     map[topodevice.ID]devicechange.TypedValueMap
	newTargetFn               func() southbound.TargetIf
	operationalStateCacheLock *sync.RWMutex
	deviceChangeStore         device.Store
	mastershipStore           mastership.Store
	mu                        sync.RWMutex
}

// NewSessionManager create a new session manager
func NewSessionManager(options ...func(*SessionManager)) (*SessionManager, error) {
	sessionManager := &SessionManager{}

	for _, option := range options {
		option(sessionManager)
	}

	return sessionManager, nil

}

// WithTopoChannel sets topo channel
func WithTopoChannel(topoChannel chan *topodevice.ListResponse) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.topoChannel = topoChannel
	}
}

// WithSessions sets list of sessions
func WithSessions(sessions map[topodevice.ID]*Session) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.sessions = sessions
	}
}

// WithDeviceStore sets device store
func WithDeviceStore(deviceStore devicestore.Store) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.deviceStore = deviceStore
	}
}

// WithMastershipStore sets mastership store
func WithMastershipStore(mastershipStore mastership.Store) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.mastershipStore = mastershipStore
	}
}

// WithOpStateChannel sets opStateChannel
func WithOpStateChannel(opStateChan chan<- events.OperationalStateEvent) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.opStateChan = opStateChan
	}
}

// WithDispatcher sets dispatcher
func WithDispatcher(dispatcher *dispatcher.Dispatcher) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.dispatcher = dispatcher
	}
}

// WithModelRegistry sets model registry
func WithModelRegistry(modelRegistry *modelregistry.ModelRegistry) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.modelRegistry = modelRegistry
	}
}

// WithOperationalStateCache sets operational state cache
func WithOperationalStateCache(operationalStateCache map[topodevice.ID]devicechange.TypedValueMap) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.operationalStateCache = operationalStateCache
	}
}

// WithNewTargetFn sets southbound target function
func WithNewTargetFn(newTargetFn func() southbound.TargetIf) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.newTargetFn = newTargetFn
	}
}

// WithOperationalStateCacheLock sets operational state cache lock
func WithOperationalStateCacheLock(operationalStateCacheLock *sync.RWMutex) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.operationalStateCacheLock = operationalStateCacheLock
	}
}

// WithDeviceChangeStore sets device change store
func WithDeviceChangeStore(deviceChangeStore device.Store) func(*SessionManager) {
	return func(sessionManager *SessionManager) {
		sessionManager.deviceChangeStore = deviceChangeStore
	}
}

// Start starts session manager
func (sm *SessionManager) Start() error {
	log.Info("Session manager starting")
	go sm.processDeviceEvents(sm.topoChannel)

	err := sm.deviceStore.Watch(sm.topoChannel)
	if err != nil {
		return err
	}

	log.Info("Session manager started")
	return nil
}

// processDeviceEvents process incoming device events
func (sm *SessionManager) processDeviceEvents(ch <-chan *topodevice.ListResponse) {
	for event := range ch {
		log.Infof("Received event type %s for device %s:%s", event.Type, event.Device.ID, event.Device.Version)
		err := sm.processDeviceEvent(event)
		if err != nil {
			log.Errorf("Error updating session %s:%s", event.Device.ID, event.Device.Version, err)
		}
	}
}

// processDeviceEvent process a device event
func (sm *SessionManager) processDeviceEvent(event *topodevice.ListResponse) error {
	switch event.Type {
	case topodevice.ListResponseADDED:
		err := sm.createSession(event.Device)
		if err != nil {
			return err
		}

	case topodevice.ListResponseNONE:
		err := sm.createSession(event.Device)
		if err != nil {
			return err
		}

	case topodevice.ListResponseUPDATED:
		session, ok := sm.sessions[event.Device.ID]
		if !ok {
			log.Errorf("Session for the device %s:%s does not exist", event.Device.ID, event.Device.Version)
			return nil
		}
		// If the address is changed, delete the current session and creates  new one
		if session.device.Address != event.Device.Address {
			err := sm.deleteSession(event.Device)
			if err != nil {
				return err
			}
			err = sm.createSession(event.Device)
			if err != nil {
				return err
			}
		}

	case topodevice.ListResponseREMOVED:
		err := sm.deleteSession(event.Device)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sm *SessionManager) handleMastershipEvents(session *Session) {
	ch := make(chan mastership.Mastership)
	err := sm.mastershipStore.Watch(session.device.ID, ch)
	if err != nil {
		return
	}

	for {
		select {
		case state := <-ch:
			currentTerm, err := session.getCurrentTerm()
			if err != nil {
				log.Error(err)
			}
			if state.Master == sm.mastershipStore.NodeID() && !session.connected && uint64(state.Term) >= currentTerm {
				err := sm.createSession(session.device)
				if err != nil {
					log.Error(err)
				} else {
					sm.mu.Lock()
					session.connected = true
					sm.mu.Unlock()
				}
			} else if state.Master != sm.mastershipStore.NodeID() && session.connected {
				err := sm.deleteSession(session.device)
				if err != nil {
					log.Error(err)
				} else {
					sm.mu.Lock()
					session.connected = false
					sm.mu.Unlock()
				}
			}
		case <-sm.closeCh:
			return
		}
	}
}

// createSession creates a new gNMI session
func (sm *SessionManager) createSession(device *topodevice.Device) error {
	log.Infof("Creating session for device %s:%s", device.ID, device.Version)

	state, err := sm.mastershipStore.GetMastership(device.ID)
	if err != nil {
		return err
	}

	session := &Session{
		opStateChan:               sm.opStateChan,
		dispatcher:                sm.dispatcher,
		modelRegistry:             sm.modelRegistry,
		operationalStateCache:     sm.operationalStateCache,
		operationalStateCacheLock: sm.operationalStateCacheLock,
		deviceChangeStore:         sm.deviceChangeStore,
		device:                    device,
		target:                    sm.newTargetFn(),
		deviceStore:               sm.deviceStore,
		mastershipState:           state,
		nodeID:                    sm.mastershipStore.NodeID(),
	}

	err = session.open()
	if err != nil {
		return err
	}

	go func() {
		sm.handleMastershipEvents(session)
	}()

	// Close the old session and adds the new session to the list of sessions
	oldSession, ok := sm.sessions[device.ID]
	if ok {
		oldSession.Close()
	}
	sm.sessions[device.ID] = session

	return nil
}

// deleteSession deletes a new session
func (sm *SessionManager) deleteSession(device *topodevice.Device) error {
	log.Infof("Deleting session for device %s:%s", device.ID, device.Version)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	session, ok := sm.sessions[device.ID]
	if ok {
		session.Close()
		delete(sm.sessions, device.ID)
		if sm.closeCh != nil {
			close(sm.closeCh)
		}
	}
	return nil

}
