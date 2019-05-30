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
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/listener"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"log"
)

// Factory is a go routine thread that listens out for Device creation
// and deletion events and spawns Synchronizer threads for them
// These synchronizers then listen out for configEvents relative to a device and
// propagate them downwards to the gNMI dispatcher
func Factory(changeStore *store.ChangeStore, deviceStore *topocache.DeviceStore,
	topoChannel <-chan events.TopoEvent, dispatcher *listener.Dispatcher) {

	for topoEvent := range topoChannel {
		deviceName := events.Event(topoEvent).Subject()
		if !dispatcher.HasListener(deviceName) && topoEvent.Connect() {

			configChan, err := dispatcher.Register(deviceName, true)
			if err != nil {
				log.Fatal(err)
			}
			device := deviceStore.Store[deviceName]
			go Devicesync(changeStore, &device, configChan)
		} else if dispatcher.HasListener(deviceName) && !topoEvent.Connect() {
			err := dispatcher.Unregister(deviceName, true)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
