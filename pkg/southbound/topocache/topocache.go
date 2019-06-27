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
Package topocache is a mechanism for holding a cache of Devices.

When onos-topology is in place it will be the ultimate reference of device
availability and accessibility
Until then this simple cache will load a set of Device definitions from file
*/
package topocache

import (
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-config/pkg/events"
	"os"
	"regexp"
)

const storeTypeDevice = "device"
const storeVersion = "1.0.0"
const deviceIDNamePattern = `[a-zA-Z0-9\-:_]{4,40}`
const deviceVersionPattern = `[a-zA-Z0-9_\.]{2,10}` //same as configuration.go

// ID is an alias for string
type ID string

// Device - the definition of Device will ultimately come from onos-topology
type Device struct {
	ID                                                ID
	Addr, Target, Usr, Pwd, CaPath, CertPath, KeyPath string
	Plain, Insecure                                   bool
	Timeout                                           int64
	SoftwareVersion                                   string
}

// DeviceStore is the model of the Device store
type DeviceStore struct {
	Version     string
	Storetype   string
	Store       map[ID]Device
	topoChannel chan<- events.TopoEvent
}

// LoadDeviceStore loads a device store from a file - will eventually be from onos-topology
func LoadDeviceStore(file string, topoChannel chan<- events.TopoEvent) (*DeviceStore, error) {
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
	deviceStore.topoChannel = topoChannel
	// Validate that the store is OK before sending out any events
	for id, device := range deviceStore.Store {
		err := validateDevice(id, device)
		if err != nil {
			return nil, fmt.Errorf("Error loading store: %s: %v", id, err)
		}
	}

	// We send a creation event for each device in store
	for _, device := range deviceStore.Store {
		topoChannel <- events.CreateTopoEvent(string(device.ID), true, device.Addr)
	}

	return &deviceStore, nil
}

// AddOrUpdateDevice adds or updates the specified device in the device inventory.
func (store *DeviceStore) AddOrUpdateDevice(id ID, device Device) error {
	err := validateDevice(id, device)
	if err == nil {
		store.Store[id] = device
		store.topoChannel <- events.CreateTopoEvent(string(device.ID), true, device.Addr)
	}
	return err
}

// RemoveDevice removes the device with the specified address from the device inventory.
func (store *DeviceStore) RemoveDevice(id ID) {
	delete(store.Store, id)
}

func validateDevice(id ID, device Device) error {
	if device.ID == "" {
		return fmt.Errorf("device %v has blank ID", device)
	}
	if device.ID != id {
		return fmt.Errorf("device %v ID mismatch with key: %v", device, id)
	}

	rname := regexp.MustCompile(deviceIDNamePattern)
	matchName := rname.FindString(string(id))
	if string(id) != matchName {
		return fmt.Errorf("name %s does not match pattern %s",
			id, deviceIDNamePattern)
	}

	if device.Addr == "" {
		return fmt.Errorf("device %v has blank address", device)
	}
	if device.SoftwareVersion == "" {
		return fmt.Errorf("device %v has blank software version", device)
	}
	rversion := regexp.MustCompile(deviceVersionPattern)
	matchVer := rversion.FindString(device.SoftwareVersion)
	if device.SoftwareVersion != matchVer {
		return fmt.Errorf("version %s does not match pattern %s",
			device.SoftwareVersion, deviceVersionPattern)
	}
	return nil
}
