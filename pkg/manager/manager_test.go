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
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
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

func setUp(t *testing.T) (*Manager, map[string]*change.Change, map[store.ConfigName]store.Configuration, *MockStore) {

	var change1 *change.Change
	var device1config *store.Configuration
	var mgrTest *Manager

	var (
		changeStoreTest        map[string]*change.Change
		configurationStoreTest map[store.ConfigName]store.Configuration
		networkStoreTest       []store.NetworkConfiguration
	)

	var err error
	config1Value01, _ := change.NewChangeValue(Test1Cont1A, change.NewTypedValueEmpty(), false)
	config1Value02, _ := change.NewChangeValue(Test1Cont1ACont2A, change.NewTypedValueEmpty(), false)
	config1Value03, _ := change.NewChangeValue(Test1Cont1ACont2ALeaf2A, change.NewTypedValueFloat(ValueLeaf2B159), false)
	change1, err = change.NewChange(change.ValueCollections{
		config1Value01, config1Value02, config1Value03}, "Original Config for test switch")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
	changeStoreTest = make(map[string]*change.Change)
	changeStoreTest[store.B64(change1.ID)] = change1

	device1config, err = store.NewConfiguration("Device1", "1.0.0", "TestDevice",
		[]change.ID{change1.ID})
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	configurationStoreTest = make(map[store.ConfigName]store.Configuration)
	configurationStoreTest[device1config.Name] = *device1config

	ctrl := gomock.NewController(t)
	mockStore := NewMockStore(ctrl)

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
		//TODO mock
		mockStore,
		&store.NetworkStore{
			Version:   "1.0",
			Storetype: "network",
			Store:     networkStoreTest,
		},
		make(chan *devicepb.ListResponse, 10))
	if err != nil {
		log.Warning(err)
		os.Exit(-1)
	}
	mgrTest.Run()
	return mgrTest, changeStoreTest, configurationStoreTest, mockStore
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

	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig("Device1", "Running", "/*", 0)
	assert.NilError(t, err, "GetTargetConfig error")

	assert.Equal(t, len(result), 3, "Unexpected result element count")

	assert.Equal(t, result[0].Path, Test1Cont1A, "result %s is different")
}

func Test_SetNetworkConfig(t *testing.T) {
	// First verify the value beforehand
	const origChangeHash = "65MwjiKdV5lnh19sKeapnlAUxGw="
	mgrTest, changeStoreTest, configurationStoreTest, _ := setUp(t)
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
	updates[Test1Cont1ACont2ALeaf2A] = (*change.TypedValue)(change.NewTypedValueFloat(ValueLeaf2B314))
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
	mgrTest, changeStoreTest, configurationStoreTest, _ := setUp(t)

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	// Making change
	updates[Test1Cont1ACont2ALeaf2A] = (*change.TypedValue)(change.NewTypedValueFloat(ValueLeaf2B314))
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
	mgrTest, _, configurationStoreTest, _ := setUp(t)

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	// Making change
	updates[Test1Cont1ACont2ALeaf2A] = (*change.TypedValue)(change.NewTypedValueFloat(ValueLeaf2B314))
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	_, _, err := mgrTest.SetNetworkConfig("Device6", "1.0.0", "", updates, deletes, "Test_SetNetworkConfig_NewConfig")
	assert.ErrorContains(t, err, "no configuration found matching 'Device6-1.0.0' and no device type () given Please specify version and device type in extensions 101 and 102")
	_, okupdate := configurationStoreTest["Device6-1.0.0"]
	assert.Assert(t, !okupdate, "Expecting not to find Device6-1.0.0")
}

func Test_SetBadNetworkConfig(t *testing.T) {

	mgrTest, _, _, _ := setUp(t)

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = change.NewTypedValueFloat(ValueLeaf2B159)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	_, _, err := mgrTest.SetNetworkConfig("DeviceXXXX", "", "", updates, deletes, "Testing")
	assert.ErrorContains(t, err, "no configuration found")
}

func Test_SetMultipleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _, configurationStoreTest, _ := setUp(t)

	device2config, err := store.NewConfiguration("Device1", "1.0.1", "TestDevice",
		[]change.ID{})
	assert.NilError(t, err)
	configurationStoreTest[device2config.Name] = *device2config

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = change.NewTypedValueFloat(ValueLeaf2B159)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2C)

	_, _, err = mgrTest.SetNetworkConfig("Device1", "", "", updates, deletes, "Testing")
	assert.ErrorContains(t, err, "configurations found for")
}

func Test_SetSingleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _, _, _ := setUp(t)

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = change.NewTypedValueFloat(ValueLeaf2B159)
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
	mgrTest, _, configurationStoreTest, _ := setUp(t)

	device2config, err := store.NewConfiguration("Device2", "1.0.0", "TestDevice",
		[]change.ID{})
	assert.NilError(t, err)
	configurationStoreTest[device2config.Name] = *device2config
	device3config, err := store.NewConfiguration("Device3", "1.0.0", "TestDevice",
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
	mgrTest, _, _, _ := setUp(t)

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
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig("Device1", "Running", "/*", 0)
	assert.Assert(t, len(result) == 3, "Get of device all paths does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a"), "/cont1a not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a"), "/cont1a/cont2a not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetOneConfig(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig("Device1", "Running", "/cont1a/cont2a/leaf2a", 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetWildcardConfig(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig("Device1", "Running", "/*/*/leaf2a", 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetConfigNoTarget(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig("", "Running", "/cont1a/cont2a/leaf2a", 0)
	assert.Assert(t, len(result) == 0, "Get of device one path does not return proper array")
	assert.ErrorContains(t, err, "no Configuration found for Running")
}

func TestManager_GetManager(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)
	assert.Equal(t, mgrTest, GetManager())
	GetManager().Close()
	assert.Equal(t, mgrTest, GetManager())
}

func TestManager_ComputeRollbackDelete(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	updates := make(change.TypedValueMap)
	deletes := make([]string, 0)

	updates[Test1Cont1ACont2ALeaf2B] = change.NewTypedValueFloat(ValueLeaf2B159)
	err := mgrTest.ValidateNetworkConfig("Device1", "1.0.0", "", updates, deletes)
	assert.NilError(t, err, "ValidateTargetConfig error")
	_, _, err = mgrTest.SetNetworkConfig("Device1", "1.0.0", "", updates, deletes, "Testing rollback")

	assert.NilError(t, err, "Can't create change", err)

	updates[Test1Cont1ACont2ALeaf2B] = change.NewTypedValueFloat(ValueLeaf2B314)
	updates[Test1Cont1ACont2ALeaf2D] = change.NewTypedValueFloat(ValueLeaf2D314)
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

func TestManager_GetTargetState(t *testing.T) {
	const (
		device1 = "device1"
		path1   = "/a/b/c"
		path1WC = "/a/*/c"
		value1  = "v1"
		path2   = "/x/y/z"
		value2  = "v2"
		badPath = "/no/such/path"
	)
	mgrTest, _, _, _ := setUp(t)

	//  Create an op state map with test data
	device1ValueMap := make(map[string]*change.TypedValue)
	change1 := change.NewTypedValueString(value1)
	change2 := change.NewTypedValueString(value2)
	device1ValueMap[path1] = change1
	device1ValueMap[path2] = change2
	mgrTest.OperationalStateCache[device1] = device1ValueMap

	// Test fetching a known path from the cache
	state1 := mgrTest.GetTargetState(device1, path1WC)
	assert.Assert(t, state1 != nil, "Path 1 entry not found")
	assert.Assert(t, len(state1) == 1, "Path 1 entry has incorrect length %d", len(state1))
	assert.Assert(t, state1[0].Path == path1, "Path 1 path is incorrect")
	assert.Assert(t, string(state1[0].Value) == value1, "Path 1 value is incorrect")

	// Test fetching an unknown path from the cache
	stateBad := mgrTest.GetTargetState(device1, badPath)
	assert.Assert(t, stateBad != nil, "Bad Path Entry returns nil")
	assert.Assert(t, len(stateBad) == 0, "Bad path entry has incorrect length %d", len(stateBad))
}

func TestManager_DeviceConnected(t *testing.T) {
	mgrTest, _, _, mockStore := setUp(t)
	const (
		device1 = "device1"
	)

	deviceDisconnected := &devicepb.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  "1.0.0",
	}

	device1Connected := &devicepb.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  "1.0.0",
	}

	protocolState := new(devicepb.ProtocolState)
	protocolState.Protocol = devicepb.Protocol_GNMI
	protocolState.ConnectivityState = devicepb.ConnectivityState_REACHABLE
	protocolState.ChannelState = devicepb.ChannelState_CONNECTED
	protocolState.ServiceState = devicepb.ServiceState_AVAILABLE
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)

	mockStore.EXPECT().Get(gomock.Any()).Return(deviceDisconnected, nil)
	mockStore.EXPECT().Update(gomock.Any()).Return(device1Connected, nil)

	deviceConnected, err := mgrTest.DeviceConnected(device1)

	assert.NilError(t, err)
	assert.Equal(t, deviceConnected.ID, device1Connected.ID)
	assert.Equal(t, deviceConnected.Protocols[0].Protocol, devicepb.Protocol_GNMI)
	assert.Equal(t, deviceConnected.Protocols[0].ConnectivityState, devicepb.ConnectivityState_REACHABLE)
	assert.Equal(t, deviceConnected.Protocols[0].ChannelState, devicepb.ChannelState_CONNECTED)
	assert.Equal(t, deviceConnected.Protocols[0].ServiceState, devicepb.ServiceState_AVAILABLE)
}

func TestManager_DeviceDisconnected(t *testing.T) {
	mgrTest, _, _, mockStore := setUp(t)
	const (
		device1 = "device1"
	)

	deviceDisconnected := &devicepb.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  "1.0.0",
	}

	device1Connected := &devicepb.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  "1.0.0",
	}

	protocolState := new(devicepb.ProtocolState)
	protocolState.Protocol = devicepb.Protocol_GNMI
	protocolState.ConnectivityState = devicepb.ConnectivityState_REACHABLE
	protocolState.ChannelState = devicepb.ChannelState_CONNECTED
	protocolState.ServiceState = devicepb.ServiceState_AVAILABLE
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)

	protocolStateDisconnected := new(devicepb.ProtocolState)
	protocolStateDisconnected.Protocol = devicepb.Protocol_GNMI
	protocolStateDisconnected.ConnectivityState = devicepb.ConnectivityState_UNREACHABLE
	protocolStateDisconnected.ChannelState = devicepb.ChannelState_DISCONNECTED
	protocolStateDisconnected.ServiceState = devicepb.ServiceState_UNAVAILABLE
	deviceDisconnected.Protocols = append(device1Connected.Protocols, protocolState)

	mockStore.EXPECT().Get(gomock.Any()).Return(device1Connected, nil)
	mockStore.EXPECT().Update(gomock.Any()).Return(deviceDisconnected, nil)

	deviceUpdated, err := mgrTest.DeviceDisconnected(device1, errors.New("device reported disconnection"))

	assert.NilError(t, err)
	assert.Equal(t, deviceUpdated.ID, device1Connected.ID)
	assert.Equal(t, deviceUpdated.Protocols[0].Protocol, devicepb.Protocol_GNMI)
	assert.Equal(t, deviceUpdated.Protocols[0].ConnectivityState, devicepb.ConnectivityState_UNREACHABLE)
	assert.Equal(t, deviceUpdated.Protocols[0].ChannelState, devicepb.ChannelState_DISCONNECTED)
	assert.Equal(t, deviceUpdated.Protocols[0].ServiceState, devicepb.ServiceState_UNAVAILABLE)
}

type MockModelPlugin struct{}

func (m MockModelPlugin) ModelData() (string, string, []*gnmi.ModelData, string) {
	panic("implement me")
}

func (m MockModelPlugin) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	return nil, nil
}

func (m MockModelPlugin) Validate(*ygot.ValidatedGoStruct, ...ygot.ValidationOption) error {
	return nil
}

func (m MockModelPlugin) Schema() (map[string]*yang.Entry, error) {
	panic("implement me")
}

func (m MockModelPlugin) GetStateMode() modelregistry.GetStateMode {
	panic("implement me")
}

func TestManager_ValidateStoresReadOnlyFailure(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	plugin := MockModelPlugin{}
	mgrTest.ModelRegistry.ModelPlugins["TestDevice-1.0.0"] = plugin

	roPathMap := make(modelregistry.ReadOnlyPathMap)
	roSubPath1 := make(modelregistry.ReadOnlySubPathMap)
	roPathMap["/cont1a/cont2a/leaf2a"] = roSubPath1

	mgr.ModelRegistry.ModelReadOnlyPaths["TestDevice-1.0.0"] = roPathMap

	validationError := mgrTest.ValidateStores()
	assert.ErrorContains(t, validationError,
		"error read only path in configuration /cont1a/cont2a/leaf2a matches /cont1a/cont2a/leaf2a for TestDevice-1.0.0")
}

func TestManager_ValidateStores(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	plugin := MockModelPlugin{}
	mgrTest.ModelRegistry.ModelPlugins["TestDevice-1.0.0"] = plugin

	validationError := mgrTest.ValidateStores()
	assert.NilError(t, validationError)
}
