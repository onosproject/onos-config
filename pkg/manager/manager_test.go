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

package manager

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"gotest.tools/assert"
	"os"
	"testing"
	"time"
)

var (
	change1 *change.Change
)

var (
	device1config store.Configuration
)

var (
	mgrTest *Manager
)

var (
	changeStoreTest        map[string]*change.Change
	configurationStoreTest map[string]store.Configuration
	networkStoreTest       []store.NetworkConfiguration
	deviceStoreTest        map[string]topocache.Device
)

const (
	Test1Cont1A             = "/test1:cont1a"
	Test1Cont1ACont2A       = "/test1:cont1a/cont2a"
	Test1Cont1ACont2ALeaf2A = "/test1:cont1a/cont2a/leaf2a"
)

const (
	ValueEmpty    = ""
	ValueLeaf2A13 = "13"
)

func TestMain(m *testing.M) {
	var err error
	config1Value01, _ := change.CreateChangeValue(Test1Cont1A, ValueEmpty, false)
	config1Value02, _ := change.CreateChangeValue(Test1Cont1ACont2A, ValueEmpty, false)
	config1Value03, _ := change.CreateChangeValue(Test1Cont1ACont2ALeaf2A, ValueLeaf2A13, false)
	change1, err = change.CreateChange(change.ValueCollections{
		config1Value01, config1Value02, config1Value03,}, "Original Config for test switch")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	changeStoreTest = make(map[string]*change.Change)
	changeStoreTest[store.B64(change1.ID)] = change1

	device1config = store.Configuration{
		Name:        "Device1Version",
		Device:      "Device1",
		Created:     time.Now(),
		Updated:     time.Now(),
		User:        "onos",
		Description: "Configuration for Device 1",
		Changes:     []change.ID{change1.ID},
	}
	configurationStoreTest = make(map[string]store.Configuration)
	configurationStoreTest["Device1Version"] = device1config

	deviceStoreTest = make(map[string]topocache.Device)
	deviceStoreTest["Device1"] = topocache.Device{
		Addr:    "127.0.0.1:10161",
		Timeout: 10,
	}
	mgrTest = NewManager(
			&store.ConfigurationStore{
				Version:   "1.0",
				Storetype: "config",
				Store:     configurationStoreTest,
			},
			&store.ChangeStore{
				Version: "1.0",
				Storetype: "change",
				Store: changeStoreTest,
			},
			&topocache.DeviceStore{
				Version:"1.0",
				Storetype:"change",
				Store: deviceStoreTest,
			},
			&store.NetworkStore{
				Version:"1.0",
				Storetype:"network",
				Store: networkStoreTest,
			},
			make(chan events.Event, 10))
	mgrTest.Run()
	os.Exit(m.Run())

}

func Test_GetNetworkConfig(t *testing.T) {

	result, err := mgrTest.GetNetworkConfig("Device1", "running", "/*", 0)
	assert.NilError(t, err, "GetNetworkConfig error")

	assert.Equal(t, len(result), 3, "Unexpected result element count")

	assert.Equal(t, result[0].Path, Test1Cont1A, "result %s is different")
}
