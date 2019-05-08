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
Package listener is a channel for handling Configuration Events
The channel system here is a two tier affair that forwards changes from the core
configuration store to NBI listeners and to any registered device listeners
This is so that the Configuration system does not have to be aware of the presence
or lack of NBI, Device synchronizers etc.
*/
package listener

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/events"
	"log"
)

var (
	deviceListeners map[string]chan events.Event
	nbiListeners    map[string]chan events.Event
)

func init() {
	deviceListeners = make(map[string]chan events.Event)
	nbiListeners = make(map[string]chan events.Event)
}

// Listen is a go routine function that listens out for changes made in the
// configuration and distributes this to registered deviceListeners on the
// Southbound and registered nbiListeners on the northbound
// Southbound listeners are only sent the events that matter to them
// All events.Events are sent to northbound listeners
func Listen(changeChannel <-chan events.Event) {
	log.Println("Event listener initialized")

	for configEvent := range changeChannel {
		for device, deviceChan := range deviceListeners {
			if configEvent.Subject() == device {
				deviceChan <- configEvent
			}
		}
		for _, nbiChan := range nbiListeners {
			nbiChan <- configEvent
		}

		if len(deviceListeners)+len(nbiListeners) == 0 {
			log.Println("Event discarded", configEvent)
		}
	}
}

// Register is a way for device synchronizers or nbi instances to register for
// channel of events
func Register(subscriber string, isDevice bool) (chan events.Event, error) {
	if isDevice && deviceListeners[subscriber] != nil {
		return nil, fmt.Errorf("Device %s is already registered", subscriber)
	} else if nbiListeners[subscriber] != nil {
		return nil, fmt.Errorf("NBI %s is already registered", subscriber)
	}
	channel := make(chan events.Event)
	if isDevice {
		deviceListeners[subscriber] = channel
	} else {
		nbiListeners[subscriber] = channel
	}
	return channel, nil
}

// Unregister closes the device channel and removes it from the deviceListeners
func Unregister(subscriber string, isDevice bool) error {
	var channel chan events.Event
	if isDevice {
		channel = deviceListeners[subscriber]
	} else {
		channel = nbiListeners[subscriber]
	}
	if channel == nil {
		return fmt.Errorf("Subscriber %s had not been registered", subscriber)
	}
	close(channel)
	if isDevice {
		deviceListeners[subscriber] = nil
	} else {
		nbiListeners[subscriber] = nil
	}
	return nil
}

// ListListeners returns a list of registered listeners
func ListListeners() []string {
	listenerKeys := make([]string, 0)
	for k := range deviceListeners {
		listenerKeys = append(listenerKeys, k)
	}
	for k := range nbiListeners {
		listenerKeys = append(listenerKeys, k)
	}
	return listenerKeys
}

// CheckListener allows a check for a name listener
func CheckListener(name string) bool {
	for k := range deviceListeners {
		if k == name {
			return true
		}
	}
	return false
}
