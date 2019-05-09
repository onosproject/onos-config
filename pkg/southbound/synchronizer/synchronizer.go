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

// Package synchronizer synchronizes configurations down to devices
package synchronizer

import (
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"log"
)

// Devicesync is a go routine that listens out for configuration events specific
// to a device and propagates them downwards through southbound interface
func Devicesync(changeStore *store.ChangeStore,
	device *topocache.Device, deviceChan <-chan events.Event) {

	log.Println("Connecting to", device.Addr, "over gNMI")

	target, err := southbound.GetTarget(southbound.Key{Key: device.Addr})
	if err != nil {
		log.Println("Could not get target", err)
		target, _, err = southbound.ConnectTarget(*device)
		log.Println(device.Addr, "Connected over gNMI")
		if err != nil {
			log.Println(err)
			return
		}
	}

	// Get the device capabilities
	capResponse, capErr := southbound.CapabilitiesWithString(target, "")
	if capErr != nil {
		log.Println(device.Addr, "Capabilities", err)
	}

	log.Println(device.Addr, "Capabilities", capResponse)

	for deviceConfigEvent := range deviceChan {
		change := changeStore.Store[deviceConfigEvent.Value(events.ChangeID)]
		err := change.IsValid()
		if err != nil {
			log.Println("Event discarded because change is invalid", err)
			continue
		}
		gnmiChange, parseError := change.GnmiChange()

		if parseError != nil {
			log.Println("Parsing error for Gnmi change ", parseError)
			continue
		}

		log.Println("Change formatted to gNMI setRequest", gnmiChange)
		setResponse, err := southbound.Set(target, gnmiChange)
		if err != nil {
			log.Println("SetResponse ", err)
			continue
		}
		log.Println(device.Addr, "SetResponse", setResponse)

	}
}
