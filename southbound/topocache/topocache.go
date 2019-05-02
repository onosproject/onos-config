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

/*
Package topocache is a mechanism for holding a cache of Devices
When onos-topology is in place it will be the ultimate reference of device
availability and accessibility
Until then this simple cache will load a set of Device definitions from file
*/
package topocache

import (
	"encoding/json"
	"fmt"
	"github.com/opennetworkinglab/onos-config/events"
	"os"
	"time"
)

const storeTypeDevice = "device"
const storeVersion = "1.0.0"

// Device - the definition of Device will ultimately come from onos-topology
type Device struct {
	Addr, Target, Usr, Pwd, CaPath, CertPath, KeyPath string
	Timeout                                           time.Duration
}

// DeviceStore is the model of the Device store
type DeviceStore struct {
	Version   string
	Storetype string
	Store     map[string]Device
}

// LoadDeviceStore loads a device store from a file - will eventually be from onos-topology
func LoadDeviceStore(file string, topoChannel chan<- events.Event) (*DeviceStore, error) {
	storeFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer storeFile.Close()

	jsonDecoder := json.NewDecoder(storeFile)
	var deviceStore = DeviceStore{}
	jsonDecoder.Decode(&deviceStore)
	if deviceStore.Storetype != storeTypeDevice {
		return nil,
			fmt.Errorf("Store type invalid: " + deviceStore.Storetype)
	} else if deviceStore.Version != storeVersion {
		return nil,
			fmt.Errorf("Store version invalid: " + deviceStore.Version)
	}

	// We send a creation event for each device in store
	for _, device := range deviceStore.Store {
		values := make(map[string]string)
		values["connect"] = "true"
		topoChannel <- events.CreateEvent(device.Addr, events.EventTypeTopoCache, values)
	}

	return &deviceStore, nil
}


