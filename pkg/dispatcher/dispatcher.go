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
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"log"
)

// Dispatcher manages SB and NB configuration event listeners
type Dispatcher struct {
	deviceListeners     map[topocache.ID]chan events.ConfigEvent
	nbiListeners        map[string]chan events.ConfigEvent
	nbiOpStateListeners map[string]chan events.OperationalStateEvent
}

// NewDispatcher creates and initializes a new event dispatcher
func NewDispatcher() Dispatcher {
	return Dispatcher{
		deviceListeners:     make(map[topocache.ID]chan events.ConfigEvent),
		nbiListeners:        make(map[string]chan events.ConfigEvent),
		nbiOpStateListeners: make(map[string]chan events.OperationalStateEvent),
	}
}

// Listen is a go routine function that listens out for changes made in the
// configuration and distributes this to registered deviceListeners on the
// Southbound and registered nbiListeners on the northbound
// Southbound listeners are only sent the events that matter to them
// All events.Events are sent to northbound listeners
func (d *Dispatcher) Listen(changeChannel <-chan events.ConfigEvent) {
	log.Println("Event listener initialized")

	for configEvent := range changeChannel {
		log.Println("Listener: Event", configEvent)
		deviceChan, ok := d.deviceListeners[topocache.ID(events.Event(configEvent).Subject())]
		if ok {
			log.Println("Device Simulators must be active")
			//TODO need a timeout or be done in separate routine
			log.Println(deviceChan)
			deviceChan <- configEvent
		}

		for _, nbiChan := range d.nbiListeners {
			nbiChan <- configEvent
		}

		if len(d.deviceListeners)+len(d.nbiListeners) == 0 {
			log.Println("Event discarded", configEvent)
		}
	}
}

// ListenOperationalState is a go routine function that listens out for changes made in the
// configuration and distributes this to registered deviceListeners on the
// Southbound and registered nbiListeners on the northbound
// Southbound listeners are only sent the events that matter to them
// All events.Events are sent to northbound listeners
func (d *Dispatcher) ListenOperationalState(operationalStateChannel <-chan events.OperationalStateEvent) {
	log.Println("Operational State Event listener initialized")

	for operationalStateEvent := range operationalStateChannel {
		log.Println("Listener: Operational State Event", operationalStateEvent)
		for _, nbiChan := range d.nbiOpStateListeners {
			nbiChan <- operationalStateEvent
		}

		if len(d.nbiOpStateListeners) == 0 {
			//TODO make sure not to flood the system
			log.Println("Operational Event discarded", operationalStateEvent)
		}
	}
}

// RegisterDevice is a way for device synchronizers to register for
// channel of config events
func (d *Dispatcher) RegisterDevice(id topocache.ID) (chan events.ConfigEvent, error) {
	if d.deviceListeners[id] != nil {
		return nil, fmt.Errorf("Device %s is already registered", id)
	}
	channel := make(chan events.ConfigEvent)
	d.deviceListeners[id] = channel
	log.Printf("Registering Device %s on channel w%v", id, channel)
	return channel, nil
}

// RegisterNbi is a way for nbi instances to register for
// channel of config events
func (d *Dispatcher) RegisterNbi(subscriber string) (chan events.ConfigEvent, error) {
	if d.nbiListeners[subscriber] != nil {
		return nil, fmt.Errorf("NBI %s is already registered", subscriber)
	}
	channel := make(chan events.ConfigEvent)
	d.nbiListeners[subscriber] = channel
	log.Printf("Registering NBI %s on channel w%v", subscriber, channel)
	return channel, nil
}

// RegisterOpState is a way for nbi instances to register for
// channel of events
func (d *Dispatcher) RegisterOpState(subscriber string) (chan events.OperationalStateEvent, error) {
	if d.nbiOpStateListeners[subscriber] != nil {
		return nil, fmt.Errorf("NBI operational state %s is already registered", subscriber)
	}
	channel := make(chan events.OperationalStateEvent)
	d.nbiOpStateListeners[subscriber] = channel
	return channel, nil
}

// UnregisterDevice closes the device config channel and removes it from the deviceListeners
func (d *Dispatcher) UnregisterDevice(id topocache.ID) error {
	channel, ok := d.deviceListeners[id]
	if !ok {
		return fmt.Errorf("Subscriber %s had not been registered", id)
	}
	delete(d.deviceListeners, id)
	close(channel)
	return nil
}

// UnregisterNbi closes the nbi config channel and removes it from the nbiListeners
func (d *Dispatcher) UnregisterNbi(subscriber string) error {
	channel, ok := d.nbiListeners[subscriber]
	if !ok {
		return fmt.Errorf("Subscriber %s had not been registered", subscriber)
	}
	delete(d.nbiListeners, subscriber)
	close(channel)
	return nil
}

// UnregisterOperationalState closes the device channel and removes it from the deviceListeners
func (d *Dispatcher) UnregisterOperationalState(subscriber string) error {
	var channel chan events.OperationalStateEvent
	channel = d.nbiOpStateListeners[subscriber]
	if channel == nil {
		return fmt.Errorf("Subscriber %s had not been registered", subscriber)
	}
	delete(d.nbiOpStateListeners, subscriber)
	close(channel)
	return nil
}

// GetListeners returns a list of registered listeners names
func (d *Dispatcher) GetListeners() []string {
	listenerKeys := make([]string, 0)
	for k := range d.deviceListeners {
		listenerKeys = append(listenerKeys, string(k))
	}
	for k := range d.nbiListeners {
		listenerKeys = append(listenerKeys, k)
	}
	for k := range d.nbiOpStateListeners {
		listenerKeys = append(listenerKeys, k)
	}
	return listenerKeys
}

// HasListener returns true if the named listeners has been registered
func (d *Dispatcher) HasListener(name topocache.ID) bool {
	_, ok := d.deviceListeners[name]
	return ok
}
