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
	"bytes"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"gotest.tools/assert"
	log "k8s.io/klog"
	"os"
	"strings"
	"testing"
)

const (
	Test1Cont1A             = "/cont1a"
	Test1Cont1ACont2A       = "/cont1a/cont2a"
	Test1Cont1ACont2ALeaf2A = "/cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B = "/cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C = "/cont1a/cont2a/leaf2c"
	Test1Cont1ACont2ALeaf2D = "/cont1a/cont2a/leaf2d"
)

const (
	ValueLeaf2B159 = 1.579
	ValueLeaf2B314 = 3.14
	ValueLeaf2D314 = 3.14
)

// TestMain should only contain static data.
// It is run once for all tests - each test is then run on its own thread, so if
// anything is shared the order of it's modification is not deterministic
// Also there can only be one TestMain per package
func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
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
	)

	var err error
	config1Value01, _ := change.CreateChangeValue(Test1Cont1A, change.CreateTypedValueEmpty(), false)
	config1Value02, _ := change.CreateChangeValue(Test1Cont1ACont2A, change.CreateTypedValueEmpty(), false)
	config1Value03, _ := change.CreateChangeValue(Test1Cont1ACont2ALeaf2A, change.CreateTypedValueFloat(ValueLeaf2B159), false)
	change1, err = change.CreateChange(change.ValueCollections{
		config1Value01, config1Value02, config1Value03}, "Original Config for test switch")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
	changeStoreTest = make(map[string]*change.Change)
	changeStoreTest[store.B64(change1.ID)] = change1

	device1config, err = store.CreateConfiguration("Device1", "1.0.0", "TestDevice",
		[]change.ID{change1.ID})
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	configurationStoreTest = make(map[store.ConfigName]store.Configuration)
	configurationStoreTest[device1config.Name] = *device1config

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
		&topocache.DeviceStore{},
		&store.NetworkStore{
			Version:   "1.0",
			Storetype: "network",
			Store:     networkStoreTest,
		},
		make(chan events.TopoEvent, 10))
	if err != nil {
		log.Warning(err)
		os.Exit(-1)
	}
	mgrTest.Run()
	return mgrTest, changeStoreTest, configurationStoreTest
}

func Test_LoadManager(t *testing.T) {
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sample.json",
		"../../configs/networkStore-sample.json",
	)
	assert.NilError(t, err, "failed to load manager")
}

func Test_LoadManagerBadConfigStore(t *testing.T) {
	_, err := LoadManager(
		"../../configs/configStore-sampleX.json",
		"../../configs/changeStore-sample.json",
		"../../configs/networkStore-sample.json",
	)
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_LoadManagerBadChangeStore(t *testing.T) {
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sampleX.json",
		"../../configs/networkStore-sample.json",
	)
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_LoadManagerBadNetworkStore(t *testing.T) {
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sample.json",
		"../../configs/networkStore-sampleX.json",
	)
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_GetNetworkConfig(t *testing.T) {

	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetTargetConfig("Device1", "Running", "/*", 0)
	assert.NilError(t, err, "GetTargetConfig error")

	assert.Equal(t, len(result), 3, "Unexpected result element count")

	assert.Equal(t, result[0].Path, Test1Cont1A, "result %s is different")
}

func Test_SetNetworkConfig(t *testing.T) {
	// First verify the value beforehand
	const origChangeHash = "65MwjiKdV5lnh19sKeapnlAUxGw="
	mgrTest, changeStoreTest, configurationStoreTest := setUp()
	assert.Equal(t, len(changeStoreTest), 1)
	i := 0
	keys := make([]string, len(changeStoreTest))
	for k := range changeStoreTest {
		keys[i] = k
		i++
	}
	assert.Equal(t, keys[0], origChangeHash)
	originalChange, ok := changeStoreTest[origChangeHash]
	assert.Assert(t, ok)
	assert.Equal(t, store.B64(originalChange.ID), origChangeHash)
	assert.Equal(t, len(originalChange.Config), 3)
	assert.Equal(t, originalChange.Config[2].Path, Test1Cont1ACont2ALeaf2A)
	assert.Equal(t, originalChange.Config[2].Type, change.ValueTypeFLOAT)
	assert.Equal(t, (*change.TypedFloat)(&originalChange.Config[2].TypedValue).Float32(), float32(ValueLeaf2B159))

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	// Making change
	updates[Test1Cont1ACont2ALeaf2A] = (*change.TypedValue)(change.CreateTypedValueFloat(ValueLeaf2B314))
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)
	err := mgrTest.ValidateNetworkConfig("Device1", "1.0.0", "", updates, deletes)
	assert.NilError(t, err, "ValidateTargetConfig error")
	changeID, configName, err := mgrTest.SetNetworkConfig("Device1", "1.0.0", "", updates, deletes, "Test_SetNetworkConfig")
	assert.NilError(t, err, "SetTargetConfig error")
	testUpdate := configurationStoreTest["Device1-1.0.0"]
	changeIDTest := testUpdate.Changes[len(testUpdate.Changes)-1]
	assert.Equal(t, store.B64(changeID), store.B64(changeIDTest), "Change Ids should correspond")
	change1, okChange := changeStoreTest[store.B64(changeIDTest)]
	assert.Assert(t, okChange, "Expecting change", store.B64(changeIDTest))
	assert.Equal(t, change1.Description, "Originally created as part of Test_SetNetworkConfig")

	assert.Equal(t, string(*configName), "Device1-1.0.0")

	//Done in this order because ordering on a path base in the store.
	updatedVals := changeStoreTest[store.B64(changeID)].Config
	assert.Equal(t, len(updatedVals), 2)

	//Asserting change
	assert.Equal(t, updatedVals[0].Path, Test1Cont1ACont2ALeaf2A)
	assert.Equal(t, (*change.TypedFloat)(&updatedVals[0].TypedValue).Float32(), float32(ValueLeaf2B314))
	assert.Equal(t, updatedVals[0].Remove, false)

	// Checking original is still alright
	originalChange, ok = changeStoreTest[origChangeHash]
	assert.Assert(t, ok)
	assert.Equal(t, (*change.TypedFloat)(&originalChange.Config[2].TypedValue).Float32(), float32(ValueLeaf2B159))

	//Asserting deletion of 2C
	assert.Equal(t, updatedVals[1].Path, Test1Cont1ACont2ALeaf2C)
	assert.Equal(t, (*change.TypedEmpty)(&updatedVals[1].TypedValue).String(), "")
	assert.Equal(t, updatedVals[1].Remove, true)

}

// When device type is given it is like extension 102 and allows a never heard of before config to be created
func Test_SetNetworkConfig_NewConfig(t *testing.T) {
	mgrTest, changeStoreTest, configurationStoreTest := setUp()

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	// Making change
	updates[Test1Cont1ACont2ALeaf2A] = (*change.TypedValue)(change.CreateTypedValueFloat(ValueLeaf2B314))
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	changeID, configName, err := mgrTest.SetNetworkConfig("Device5", "1.0.0", "Devicesim", updates, deletes, "Test_SetNetworkConfig_NewConfig")
	assert.NilError(t, err, "SetTargetConfig error")
	testUpdate := configurationStoreTest["Device5-1.0.0"]
	changeIDTest := testUpdate.Changes[len(testUpdate.Changes)-1]
	assert.Equal(t, store.B64(changeID), store.B64(changeIDTest), "Change Ids should correspond")
	change1, okChange := changeStoreTest[store.B64(changeIDTest)]
	assert.Assert(t, okChange, "Expecting change", store.B64(changeIDTest))
	assert.Equal(t, change1.Description, "Originally created as part of Test_SetNetworkConfig_NewConfig")

	assert.Equal(t, string(*configName), "Device5-1.0.0")
}

// When device type is given it is like extension 102 and allows a never heard of before config to be created - here it is missing
func Test_SetNetworkConfig_NewConfig102Missing(t *testing.T) {
	mgrTest, _, configurationStoreTest := setUp()

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	// Making change
	updates[Test1Cont1ACont2ALeaf2A] = (*change.TypedValue)(change.CreateTypedValueFloat(ValueLeaf2B314))
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	_, _, err := mgrTest.SetNetworkConfig("Device6", "1.0.0", "", updates, deletes, "Test_SetNetworkConfig_NewConfig")
	assert.ErrorContains(t, err, "no configuration found matching 'Device6-1.0.0' and no device type () given Please specify version and device type in extensions 101 and 102")
	_, okupdate := configurationStoreTest["Device6-1.0.0"]
	assert.Assert(t, !okupdate, "Expecting not to find Device6-1.0.0")
}

func Test_SetBadNetworkConfig(t *testing.T) {

	mgrTest, _, _ := setUp()

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = change.CreateTypedValueFloat(ValueLeaf2B159)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	_, _, err := mgrTest.SetNetworkConfig("DeviceXXXX", "", "", updates, deletes, "Testing")
	assert.ErrorContains(t, err, "no configuration found")
}

func Test_SetMultipleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _, configurationStoreTest := setUp()

	device2config, err := store.CreateConfiguration("Device1", "1.0.1", "TestDevice",
		[]change.ID{})
	assert.NilError(t, err)
	configurationStoreTest[device2config.Name] = *device2config

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = change.CreateTypedValueFloat(ValueLeaf2B159)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	_, _, err = mgrTest.SetNetworkConfig("Device1", "", "", updates, deletes, "Testing")
	assert.ErrorContains(t, err, "configurations found for")
}

func Test_SetSingleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _, _ := setUp()

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = change.CreateTypedValueFloat(ValueLeaf2B159)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	changeID, configName, err := mgrTest.SetNetworkConfig("Device1", "", "", updates, deletes, "Testing")
	assert.NilError(t, err, "Similar config not found")
	assert.Assert(t, changeID != nil)
	assert.Assert(t, configName != nil)
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

	result, err := mgrTest.GetTargetConfig("No Such Device", "Running", "/*", 0)
	assert.Assert(t, len(result) == 0, "Get of bad device does not return empty array")
	assert.ErrorContains(t, err, "no Configuration found")
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

	result, err := mgrTest.GetTargetConfig("Device1", "Running", "/*", 0)
	assert.Assert(t, len(result) == 3, "Get of device all paths does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a"), "/cont1a not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a"), "/cont1a/cont2a not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetOneConfig(t *testing.T) {
	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetTargetConfig("Device1", "Running", "/cont1a/cont2a/leaf2a", 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetWildcardConfig(t *testing.T) {
	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetTargetConfig("Device1", "Running", "/*/*/leaf2a", 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetConfigNoTarget(t *testing.T) {
	mgrTest, _, _ := setUp()

	result, err := mgrTest.GetTargetConfig("", "Running", "/cont1a/cont2a/leaf2a", 0)
	assert.Assert(t, len(result) == 0, "Get of device one path does not return proper array")
	assert.ErrorContains(t, err, "no Configuration found for Running")
}

func TestManager_GetManager(t *testing.T) {
	mgrTest, _, _ := setUp()
	assert.Equal(t, mgrTest, GetManager())
	GetManager().Close()
	assert.Equal(t, mgrTest, GetManager())
}

func TestManager_ComputeRollbackDelete(t *testing.T) {
	mgrTest, _, _ := setUp()

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = change.CreateTypedValueFloat(ValueLeaf2B159)
	err := mgrTest.ValidateNetworkConfig("Device1", "1.0.0", "", updates, deletes)
	assert.NilError(t, err, "ValidateTargetConfig error")
	_, _, err = mgrTest.SetNetworkConfig("Device1", "1.0.0", "", updates, deletes, "Testing rollback")

	assert.NilError(t, err, "Can't create change", err)

	updates[Test1Cont1ACont2ALeaf2B] = change.CreateTypedValueFloat(ValueLeaf2B314)
	updates[Test1Cont1ACont2ALeaf2D] = change.CreateTypedValueFloat(ValueLeaf2D314)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)

	err = mgrTest.ValidateNetworkConfig("Device1", "1.0.0", "", updates, deletes)
	assert.NilError(t, err, "ValidateTargetConfig error")
	changeID, configName, err := mgrTest.SetNetworkConfig("Device1", "1.0.0", "", updates, deletes, "Testing rollback 2")

	assert.NilError(t, err, "Can't create change")

	_, changes, deletesRoll, errRoll := computeRollback(mgrTest, "Device1", *configName)
	assert.NilError(t, errRoll, "Can't ExecuteRollback", errRoll)
	config := mgrTest.ConfigStore.Store[*configName]
	assert.Check(t, !bytes.Equal(config.Changes[len(config.Changes)-1], changeID), "Did not remove last change")
	assert.Equal(t, changes[Test1Cont1ACont2ALeaf2B].Type, change.ValueTypeFLOAT, "Wrong value to set after rollback")
	assert.Equal(t, (*change.TypedFloat)(changes[Test1Cont1ACont2ALeaf2B]).Float32(), float32(ValueLeaf2B159), "Wrong value to set after rollback")

	assert.Equal(t, changes[Test1Cont1ACont2ALeaf2A].Type, change.ValueTypeFLOAT, "Wrong value to set after rollback")
	assert.Equal(t, (*change.TypedFloat)(changes[Test1Cont1ACont2ALeaf2A]).Float32(), float32(ValueLeaf2B159), "Wrong value to set after rollback")

	assert.Equal(t, deletesRoll[0], Test1Cont1ACont2ALeaf2D, "Path should be deleted")
}
