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
	"strings"
	"testing"
)

const (
	Test1Cont1A             = "/test1:cont1a"
	Test1Cont1ACont2A       = "/test1:cont1a/cont2a"
	Test1Cont1ACont2ALeaf2A = "/test1:cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B = "/test1:cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C = "/test1:cont1a/cont2a/leaf2c"
)

const (
	ValueEmpty     = ""
	ValueLeaf2A13  = "13"
	ValueLeaf2B159 = "1.579"
)

// TestMain should only contain static data.
// It is run once for all tests - each test is then run on its own thread, so if
// anything is shared the order of it's modification is not deterministic
// Also there can only be one TestMain per package
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func setUp() (*Manager, map[string]*change.Change, map[store.ConfigName]store.Configuration) {

	var change1 *change.Change
	var device1config *store.Configuration
	var mgrTest *Manager

	var (
		changeStoreTest        map[string]*change.Change
		configurationStoreTest map[store.ConfigName]store.Configuration
		networkStoreTest       []store.NetworkConfiguration
		deviceStoreTest        map[topocache.ID]topocache.Device
	)

	var err error
	config1Value01, _ := change.CreateChangeValue(Test1Cont1A, ValueEmpty, false)
	config1Value02, _ := change.CreateChangeValue(Test1Cont1ACont2A, ValueEmpty, false)
	config1Value03, _ := change.CreateChangeValue(Test1Cont1ACont2ALeaf2A, ValueLeaf2A13, false)
	change1, err = change.CreateChange(change.ValueCollections{
		config1Value01, config1Value02, config1Value03}, "Original Config for test switch")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	changeStoreTest = make(map[string]*change.Change)
	changeStoreTest[store.B64(change1.ID)] = change1

	device1config, err = store.CreateConfiguration("Device1", "1.0.0", "TestDevice",
		[]change.ID{change1.ID})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	configurationStoreTest = make(map[store.ConfigName]store.Configuration)
	configurationStoreTest[device1config.Name] = *device1config

	deviceStoreTest = make(map[topocache.ID]topocache.Device)
	deviceStoreTest["Device1"] = topocache.Device{
		ID:              topocache.ID("Device1"),
		SoftwareVersion: "1.0.0",
		Addr:            "127.0.0.1:10161",
		Timeout:         10,
	}
	mgrTest, err = NewManager(
		&store.ConfigurationStore{
			Version:   "1.0",
			Storetype: "config",
			Store:     configurationStoreTest,
		},
		&store.ChangeStore{
			Version:   "1.0",
			Storetype: "change",
			Store:     changeStoreTest,
		},
		&topocache.DeviceStore{
			Version:   "1.0",
			Storetype: "change",
			Store:     deviceStoreTest,
		},
		&store.NetworkStore{
			Version:   "1.0",
			Storetype: "network",
			Store:     networkStoreTest,
		},
		make(chan events.TopoEvent, 10))
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	mgrTest.Run()
	return mgrTest, changeStoreTest, configurationStoreTest
}

func Test_LoadManager(t *testing.T) {
	mgr, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sample.json",
		"../../configs/deviceStore-sample.json",
		"../../configs/networkStore-sample.json",
	)
	assert.Assert(t, err == nil, "failed to load manager")
	assert.Equal(t, len(mgr.DeviceStore.Store), 4, "wrong number of devices loaded")
}

func Test_LoadManagerBadConfigStore(t *testing.T) {
	_, err := LoadManager(
		"../../configs/configStore-sampleX.json",
		"../../configs/changeStore-sample.json",
		"../../configs/deviceStore-sample.json",
		"../../configs/networkStore-sample.json",
	)
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_LoadManagerBadChangeStore(t *testing.T) {
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sampleX.json",
		"../../configs/deviceStore-sample.json",
		"../../configs/networkStore-sample.json",
	)
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_LoadManagerBadDeviceStore(t *testing.T) {
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sample.json",
		"../../configs/deviceStore-sampleX.json",
		"../../configs/networkStore-sample.json",
	)
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_LoadManagerBadNetworkStore(t *testing.T) {
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sample.json",
		"../../configs/deviceStore-sample.json",
		"../../configs/networkStore-sampleX.json",
	)
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_GetNetworkConfig(t *testing.T) {

	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetNetworkConfig("Device1", "Running", "/*", 0)
	assert.NilError(t, err, "GetNetworkConfig error")

	assert.Equal(t, len(result), 3, "Unexpected result element count")

	assert.Equal(t, result[0].Path, Test1Cont1A, "result %s is different")
}

func Test_SetNetworkConfig(t *testing.T) {

	mgrTest, changeStoreTest, configurationStoreTest := setUp()

	updates := make(map[string]string)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = ValueLeaf2B159
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	changeID, configName, err := mgrTest.SetNetworkConfig("Device1-1.0.0", updates, deletes)
	assert.NilError(t, err, "GetNetworkConfig error")
	testUpdate := configurationStoreTest["Device1-1.0.0"]
	changeIDTest := testUpdate.Changes[len(testUpdate.Changes)-1]
	assert.Equal(t, store.B64(changeID), store.B64(changeIDTest), "Change Ids should correspond")

	assert.Equal(t, string(configName), "Device1-1.0.0")

	//Done in this order because ordering on a path base in the store.
	updatedVals := changeStoreTest[store.B64(changeID)].Config
	assert.Equal(t, len(updatedVals), 3)

	//Asserting deletion 2A
	assert.Equal(t, updatedVals[0].Path, Test1Cont1ACont2ALeaf2A)
	assert.Equal(t, updatedVals[0].Value, "")
	assert.Equal(t, updatedVals[0].Remove, true)

	//Asserting Removal
	assert.Equal(t, updatedVals[1].Path, Test1Cont1ACont2ALeaf2B)
	assert.Equal(t, updatedVals[1].Value, ValueLeaf2B159)
	assert.Equal(t, updatedVals[1].Remove, false)

	//Asserting deletion of 2C
	assert.Equal(t, updatedVals[2].Path, Test1Cont1ACont2ALeaf2C)
	assert.Equal(t, updatedVals[2].Value, "")
	assert.Equal(t, updatedVals[2].Remove, true)

}

func Test_SetBadNetworkConfig(t *testing.T) {

	mgrTest, _, _ := setUp()

	updates := make(map[string]string)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = ValueLeaf2B159
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	_, _, err := mgrTest.SetNetworkConfig("DeviceXXXX", updates, deletes)
	assert.ErrorContains(t, err, "no configuration found")
}

func Test_SetMultipleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _, configurationStoreTest := setUp()

	device2config, err := store.CreateConfiguration("Device1", "1.0.1", "TestDevice",
		[]change.ID{})
	assert.NilError(t, err)
	configurationStoreTest[device2config.Name] = *device2config

	updates := make(map[string]string)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = ValueLeaf2B159
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	_, _, err = mgrTest.SetNetworkConfig("Device1", updates, deletes)
	assert.ErrorContains(t, err, "configurations found for")
}

func Test_SetSingleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _, _ := setUp()

	updates := make(map[string]string)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = ValueLeaf2B159
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	changeID, configName, err := mgrTest.SetNetworkConfig("Device1", updates, deletes)
	assert.NilError(t, err, "Similar config not found")
	assert.Assert(t, changeID != nil)
	assert.Assert(t, configName != "")
}

func matchDeviceID(deviceID string, deviceName string) bool {
	return strings.Contains(deviceID, deviceName)
}

func TestManager_GetAllDeviceIds(t *testing.T) {
	mgrTest, _, configurationStoreTest := setUp()

	device2config, err := store.CreateConfiguration("Device2", "1.0.0", "TestDevice",
		[]change.ID{})
	assert.NilError(t, err)
	configurationStoreTest[device2config.Name] = *device2config
	device3config, err := store.CreateConfiguration("Device3", "1.0.0", "TestDevice",
		[]change.ID{})
	assert.NilError(t, err)
	configurationStoreTest[device3config.Name] = *device3config

	deviceIds := mgrTest.GetAllDeviceIds()
	assert.Equal(t, len(*deviceIds), 3)
	assert.Assert(t, matchDeviceID((*deviceIds)[0], "Device1"))
	assert.Assert(t, matchDeviceID((*deviceIds)[1], "Device2"))
	assert.Assert(t, matchDeviceID((*deviceIds)[2], "Device3"))
}

func TestManager_GetNoConfig(t *testing.T) {
	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetNetworkConfig("No Such Device", "Running", "/*", 0)
	assert.Assert(t, len(result) == 0, "Get of bad device does not return empty array")
	assert.ErrorContains(t, err, "No Configuration found")
}

func networkConfigContainsPath(configs []*change.ConfigValue, whichOne string) bool {
	for _, config := range configs {
		if config.Path == whichOne {
			return true
		}
	}
	return false
}

func TestManager_GetAllConfig(t *testing.T) {
	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetNetworkConfig("Device1", "Running", "/*", 0)
	assert.Assert(t, len(result) == 3, "Get of device all paths does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/test1:cont1a"), "/test1:cont1a not found")
	assert.Assert(t, networkConfigContainsPath(result, "/test1:cont1a/cont2a"), "/test1:cont1a/cont2a not found")
	assert.Assert(t, networkConfigContainsPath(result, "/test1:cont1a/cont2a/leaf2a"), "/test1:cont1a/cont2a/leaf2a not found")
}

func TestManager_GetOneConfig(t *testing.T) {
	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetNetworkConfig("Device1", "Running", "/test1:cont1a/cont2a/leaf2a", 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/test1:cont1a/cont2a/leaf2a"), "/test1:cont1a/cont2a/leaf2a not found")
}

func TestManager_GetConfigNoTarget(t *testing.T) {
	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetNetworkConfig("", "Running", "/test1:cont1a/cont2a/leaf2a", 0)
	assert.Assert(t, len(result) == 0, "Get of device one path does not return proper array")
	assert.ErrorContains(t, err, "No Configuration found for Running")
}

func TestManager_GetManager(t *testing.T) {
	mgrTest, _, _ := setUp()
	assert.Equal(t, mgrTest, GetManager())
	GetManager().Close()
	assert.Equal(t, mgrTest, GetManager())
}
