// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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
	"fmt"
	"github.com/opennetworkinglab/onos-config/events"
	"log"
)

// Devicesync is a go routine that listens out for configuration events specific
// to a device and propagates them downwards through southbound interface
func Devicesync(device string, deviceChan <-chan events.ConfigurationEvent) {
	log.Println("Listen for config changes on", device)

	for deviceConfigEvent := range deviceChan {
		fmt.Println("Change for device", device, deviceConfigEvent)
	}
}
