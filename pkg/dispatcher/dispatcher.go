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

/*
Package dispatcher enables registering and unregistering of listeners for different Events.

The channel system here is a two tier affair that forwards changes from the core
configuration store to NBI listeners and to any registered device listeners
This is so that the Configuration system does not have to be aware of the presence
or lack of NBI, Device synchronizers etc.
*/
package dispatcher

import (
	"fmt"
	"sync"

	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("dispatcher")

// Dispatcher manages SB and NB configuration event listeners
type Dispatcher struct {
	nbiOpStateListenersLock sync.RWMutex
	nbiOpStateListeners     map[string]chan events.OperationalStateEvent
}

// NewDispatcher creates and initializes a new event dispatcher
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		nbiOpStateListeners: make(map[string]chan events.OperationalStateEvent),
	}
}

// ListenOperationalState is a go routine function that listens out for changes made in the
// configuration and distributes this to registered deviceListeners on the
// Southbound and registered nbiListeners on the northbound
// Southbound listeners are only sent the events that matter to them
// All events.Events are sent to northbound listeners
func (d *Dispatcher) ListenOperationalState(operationalStateChannel <-chan events.OperationalStateEvent) {
	log.Info("Operational State Event listener initialized")

	for operationalStateEvent := range operationalStateChannel {
		d.nbiOpStateListenersLock.RLock()
		for _, nbiChan := range d.nbiOpStateListeners {
			nbiChan <- operationalStateEvent
		}
		d.nbiOpStateListenersLock.RUnlock()
	}
}

// RegisterOpState is a way for nbi instances to register for
// channel of events
func (d *Dispatcher) RegisterOpState(subscriber string) (chan events.OperationalStateEvent, error) {
	d.nbiOpStateListenersLock.Lock()
	defer d.nbiOpStateListenersLock.Unlock()
	if _, ok := d.nbiOpStateListeners[subscriber]; ok {
		return nil, fmt.Errorf("NBI operational state %s is already registered", subscriber)
	}
	channel := make(chan events.OperationalStateEvent)
	d.nbiOpStateListeners[subscriber] = channel
	return channel, nil
}

// UnregisterOperationalState closes the device channel and removes it from the deviceListeners
func (d *Dispatcher) UnregisterOperationalState(subscriber string) {
	d.nbiOpStateListenersLock.RLock()
	defer d.nbiOpStateListenersLock.RUnlock()
	channel, ok := d.nbiOpStateListeners[subscriber]
	if !ok {
		log.Infof("Subscriber %s had not been registered", subscriber)
		return
	}
	delete(d.nbiOpStateListeners, subscriber)
	close(channel)
}

// GetListeners returns a list of registered listeners names
func (d *Dispatcher) GetListeners() []string {
	listenerKeys := make([]string, 0)
	d.nbiOpStateListenersLock.RLock()
	defer d.nbiOpStateListenersLock.RUnlock()
	for k := range d.nbiOpStateListeners {
		listenerKeys = append(listenerKeys, k)
	}
	return listenerKeys
}
