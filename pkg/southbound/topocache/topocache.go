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
	"context"
	"encoding/json"
	"fmt"
	atomix "github.com/atomix/atomix-go-client/pkg/client"
	"github.com/atomix/atomix-go-client/pkg/client/map_"
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
	store       map_.Map
	topoChannel chan<- events.TopoEvent
}

func NewDeviceStore(topoChannel chan<- events.TopoEvent) (*DeviceStore, error) {
	atomixController := os.Getenv("ATOMIX_CONTROLLER")
	opts := []atomix.ClientOption{
		atomix.WithNamespace(os.Getenv("ATOMIX_NAMESPACE")),
		atomix.WithApplication(os.Getenv("ATOMIX_APP")),
	}
	client, err := atomix.NewClient(atomixController, opts...)
	if err != nil {
		return nil, err
	}

	group, err := client.GetGroup(context.Background(), "raft")
	if err != nil {
		return nil, err
	}

	map_, err := group.GetMap(context.Background(), "device-store")
	if err != nil {
		return nil, err
	}
	return &DeviceStore{
		store:       map_,
		topoChannel: topoChannel,
	}, nil
}

// AddOrUpdateDevice adds or updates the specified device in the device inventory.
func (store *DeviceStore) AddOrUpdateDevice(id ID, device Device) error {
	err := validateDevice(id, device)
	if err == nil {
		deviceJson, err := json.Marshal(device)
		if err != nil {
			return err
		}
		_, err = store.store.Put(context.Background(), string(id), deviceJson)
		if err != nil {
			return err
		}
		store.topoChannel <- events.CreateTopoEvent(string(device.ID), true, device.Addr)
	}
	return err
}

// GetDevice returns a device by ID
func (store *DeviceStore) GetDevice(id ID) (Device, error) {
	kv, err := store.store.Get(context.Background(), string(id))
	if err != nil {
		return Device{}, err
	}
	device := &Device{}
	err = json.Unmarshal(kv.Value, device)
	if err != nil {
		return Device{}, err
	}
	return *device, nil
}

// GetDevices returns a list of devices in the store
func (store *DeviceStore) GetDevices() ([]Device, error) {
	ch := make(chan *map_.KeyValue)
	err := store.store.Entries(context.Background(), ch)
	if err != nil {
		return nil, err
	}

	devices := []Device{}
	for kv := range ch {
		device := &Device{}
		err := json.Unmarshal(kv.Value, device)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *device)
	}
	return devices, nil
}

// RemoveDevice removes the device with the specified address from the device inventory.
func (store *DeviceStore) RemoveDevice(id ID) error {
	_, err := store.store.Remove(context.Background(), string(id))
	if err != nil {
		return err
	}
	return nil
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
