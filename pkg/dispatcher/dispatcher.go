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
	"log"
)

// Dispatcher manages SB and NB configuration event listeners
type Dispatcher struct {
	deviceListeners map[string]chan events.ConfigEvent
	nbiListeners    map[string]chan events.ConfigEvent
}

// NewDispatcher creates and initializes a new event dispatcher
func NewDispatcher() Dispatcher {
	return Dispatcher{
		deviceListeners: make(map[string]chan events.ConfigEvent),
		nbiListeners:    make(map[string]chan events.ConfigEvent),
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
		deviceChan, ok := d.deviceListeners[events.Event(configEvent).Subject()]
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

// Register is a way for device synchronizers or nbi instances to register for
// channel of events
func (d *Dispatcher) Register(subscriber string, isDevice bool) (chan events.ConfigEvent, error) {
	if isDevice && d.deviceListeners[subscriber] != nil {
		return nil, fmt.Errorf("Device %s is already registered", subscriber)
	} else if d.nbiListeners[subscriber] != nil {
		return nil, fmt.Errorf("NBI %s is already registered", subscriber)
	}
	channel := make(chan events.ConfigEvent)
	if isDevice {
		d.deviceListeners[subscriber] = channel
	} else {
		d.nbiListeners[subscriber] = channel
	}
	log.Printf("%s=%v", subscriber, channel)
	return channel, nil
}

// Unregister closes the device channel and removes it from the deviceListeners
func (d *Dispatcher) Unregister(subscriber string, isDevice bool) error {
	var channel chan events.ConfigEvent
	if isDevice {
		channel = d.deviceListeners[subscriber]
	} else {
		channel = d.nbiListeners[subscriber]
	}
	if channel == nil {
		return fmt.Errorf("Subscriber %s had not been registered", subscriber)
	}
	if isDevice {
		delete(d.deviceListeners, subscriber)
	} else {
		delete(d.nbiListeners, subscriber)
	}
	close(channel)
	return nil
}

// GetListeners returns a list of registered dispatcher names
func (d *Dispatcher) GetListeners() []string {
	listenerKeys := make([]string, 0)
	for k := range d.deviceListeners {
		listenerKeys = append(listenerKeys, k)
	}
	for k := range d.nbiListeners {
		listenerKeys = append(listenerKeys, k)
	}
	return listenerKeys
}

// HasListener returns true if the named listeners has been registered
func (d *Dispatcher) HasListener(name string) bool {
	_, ok := d.deviceListeners[name]
	return ok
}
