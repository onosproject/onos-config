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
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	networkstore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	typeschange "github.com/onosproject/onos-config/pkg/types/change"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/types/change/network"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-config/pkg/types/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
	log "k8s.io/klog"
	"os"
	"strings"
	"testing"
	"time"
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

const (
	Device1                   = "Device1"
	DeviceVersion             = "1.0.0"
	DeviceType                = "TestDevice"
	NetworkChange1 network.ID = "NetworkChange1"
)

func setUp(t *testing.T) (*Manager, *mockstore.MockDeviceStore, *mockstore.MockNetworkChangesStore, *mockstore.MockDeviceChangesStore) {
	log.SetOutput(os.Stdout)
	var (
		mgrTest *Manager
		err     error
	)

	config1Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1A, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2A, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value03, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2A, devicechangetypes.NewTypedValueFloat(ValueLeaf2B159), false)

	ctrl := gomock.NewController(t)

	mockDeviceCache := devicestore.NewMockCache(ctrl)

	// Data for default configuration

	change1 := types.Change{
		Values: []*types.ChangeValue{
			config1Value01, config1Value02, config1Value03},
		DeviceID:      Device1,
		DeviceVersion: DeviceVersion,
	}
	deviceChange1 := &devicechange.DeviceChange{
		Change: &change1,
		ID:     "Change1",
		Status: typeschange.Status{State: typeschange.State_COMPLETE},
	}

	now := time.Now()
	networkChange1 := &network.NetworkChange{
		ID:      NetworkChange1,
		Changes: []*types.Change{&change1},
		Updated: now,
		Created: now,
		Status:  typeschange.Status{State: typeschange.State_COMPLETE},
	}

	// Mocks for stores

	// Mock Leadership Store
	mockLeadershipStore := mockstore.NewMockLeadershipStore(ctrl)
	mockLeadershipStore.EXPECT().Watch(gomock.Any()).AnyTimes()
	mockLeadershipStore.EXPECT().IsLeader().AnyTimes()

	// Mock Mastership Store
	mockMastershipStore := mockstore.NewMockMastershipStore(ctrl)
	mockMastershipStore.EXPECT().Watch(gomock.Any(), gomock.Any()).AnyTimes()

	// Mock Network changes store
	networkChangesList := make([]*network.NetworkChange, 0)
	mockNetworkChangesStore := mockstore.NewMockNetworkChangesStore(ctrl)
	mockNetworkChangesStore.EXPECT().Create(gomock.Any()).DoAndReturn(
		func(networkChange *networkchangetypes.NetworkChange) error {
			networkChangesList = append(networkChangesList, networkChange)
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Update(gomock.Any()).DoAndReturn(
		func(networkChange *networkchangetypes.NetworkChange) error {
			networkChangesList = append(networkChangesList, networkChange)
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(id networkchangetypes.ID) (*networkchangetypes.NetworkChange, error) {
			var found *networkchangetypes.NetworkChange
			for _, networkChange := range networkChangesList {
				if networkChange.ID == id {
					found = networkChange
				}
			}
			if found != nil {
				return found, nil
			}
			return nil, nil
		}).AnyTimes()
	_ = mockNetworkChangesStore.Create(networkChange1)

	mockNetworkChangesStore.EXPECT().List(gomock.Any()).DoAndReturn(
		func(c chan<- *network.NetworkChange) error {
			go func() {
				for _, networkChange := range networkChangesList {
					c <- networkChange
				}
				close(c)
			}()
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(c chan<- stream.Event, o ...networkstore.WatchOption) (stream.Context, error) {
			go func() {
				lastChange := networkChangesList[len(networkChangesList)-1]
				if lastChange.Status.Phase == changetypes.Phase_ROLLBACK {
					lastChange.Status.State = changetypes.State_COMPLETE
				}
				event := stream.Event{
					Type:   "",
					Object: lastChange,
				}
				c <- event
				close(c)
			}()
			return stream.NewContext(func() {}), nil
		}).AnyTimes()

	// Mock Device Changes Store
	mockDeviceChangesStore := mockstore.NewMockDeviceChangesStore(ctrl)
	mockDeviceChangesStore.EXPECT().Watch(gomock.Any(), gomock.Any()).AnyTimes()
	deviceChanges := make(map[devicechange.ID]*devicechange.DeviceChange)

	mockDeviceChangesStore.EXPECT().Get(deviceChange1.ID).DoAndReturn(
		func(deviceID devicechange.ID) (*devicechange.DeviceChange, error) {
			return deviceChanges[deviceID], nil
		}).AnyTimes()

	mockDeviceChangesStore.EXPECT().Create(gomock.Any()).DoAndReturn(
		func(config *devicechange.DeviceChange) error {
			deviceChanges[config.ID] = config
			return nil
		}).AnyTimes()

	mockDeviceChangesStore.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(deviceID device.VersionedID, c chan<- *devicechange.DeviceChange) (stream.Context, error) {
			changes := make([]*devicechange.DeviceChange, 0)
			for _, deviceChange := range deviceChanges {
				if device.NewVersionedID(deviceChange.Change.DeviceID, deviceChange.Change.DeviceVersion).GetID() == deviceID.GetID() {
					changes = append(changes, deviceChange)
				}
			}
			go func() {
				for _, deviceChange := range changes {
					c <- deviceChange
				}
				close(c)
			}()
			ctx := stream.NewContext(func() {})
			if len(changes) != 0 {
				return ctx, nil
			}
			return ctx, errors.New("no Configuration found")
		}).AnyTimes()
	_ = mockDeviceChangesStore.Create(deviceChange1)

	//  Mock Network Snapshot Store
	mockNetworkSnapshotStore := mockstore.NewMockNetworkSnapshotStore(ctrl)
	mockNetworkSnapshotStore.EXPECT().Watch(gomock.Any()).AnyTimes()

	// Mock Device Snapshot Store
	mockDeviceSnapshotStore := mockstore.NewMockDeviceSnapshotStore(ctrl)
	mockDeviceSnapshotStore.EXPECT().Watch(gomock.Any()).AnyTimes()

	// Mock Device Store
	mockDeviceStore := mockstore.NewMockDeviceStore(ctrl)
	mockDeviceStore.EXPECT().Watch(gomock.Any()).AnyTimes()

	mgrTest, err = NewManager(
		&store.ConfigurationStore{
			Version:   "1.0",
			Storetype: "config",
			Store:     make(map[store.ConfigName]store.Configuration),
		},
		mockLeadershipStore,
		mockMastershipStore,
		mockDeviceChangesStore,
		&store.ChangeStore{
			Version:   "1.0",
			Storetype: "change",
			Store:     make(map[string]*change.Change),
		},
		mockDeviceStore,
		mockDeviceCache,
		nil,
		mockNetworkChangesStore,
		mockNetworkSnapshotStore,
		mockDeviceSnapshotStore,
		make(chan *devicetopo.ListResponse, 10))
	if err != nil {
		log.Warning(err)
		os.Exit(-1)
	}
	mgrTest.Run()
	return mgrTest, mockDeviceStore, mockNetworkChangesStore, mockDeviceChangesStore
}

func makeDeviceChanges(device string, updates devicechangetypes.TypedValueMap, deletes []string) (
	map[string]devicechangetypes.TypedValueMap, map[string][]string, map[devicetopo.ID]TypeVersionInfo) {
	var deviceInfo map[devicetopo.ID]TypeVersionInfo

	updatesForDevice := make(map[string]devicechangetypes.TypedValueMap)
	updatesForDevice[device] = updates
	deletesForDevice := make(map[string][]string)
	deletesForDevice[device] = deletes
	return updatesForDevice, deletesForDevice, deviceInfo
}

func Test_LoadManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sample.json",
		"../../configs/networkStore-sample.json",
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		devicestore.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl))
	assert.NilError(t, err, "failed to load manager")
}

func Test_LoadManagerBadConfigStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	_, err := LoadManager(
		"../../configs/configStore-sampleX.json",
		"../../configs/changeStore-sample.json",
		"../../configs/networkStore-sample.json",
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		devicestore.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl))
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_LoadManagerBadChangeStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sampleX.json",
		"../../configs/networkStore-sample.json",
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		devicestore.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl))
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_LoadManagerBadNetworkStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	_, err := LoadManager(
		"../../configs/configStore-sample.json",
		"../../configs/changeStore-sample.json",
		"../../configs/networkStore-sampleX.json",
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		devicestore.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl))
	assert.Assert(t, err != nil, "should have failed to load manager")
}

func Test_GetNewNetworkConfig(t *testing.T) {

	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetNewConfig(Device1, DeviceVersion, "/*", 0)
	assert.NilError(t, err, "GetTargetNewConfig error")

	assert.Equal(t, len(result), 3, "Unexpected result element count")

	assert.Equal(t, result[0].Path, Test1Cont1A, "result %s is different")
}

func Test_SetNetworkConfig(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	// First verify the value beforehand
	originalChange, _ := mgrTest.NetworkChangesStore.Get(NetworkChange1)
	assert.Equal(t, len(originalChange.Changes[0].Values), 3)
	assert.Equal(t, originalChange.Changes[0].Values[2].Path, Test1Cont1ACont2ALeaf2A)
	assert.Equal(t, originalChange.Changes[0].Values[2].Value.Type, devicechangetypes.ValueType_FLOAT)
	assert.Equal(t, (*devicechangetypes.TypedFloat)(originalChange.Changes[0].Values[2].Value).Float32(), float32(ValueLeaf2B159))

	// Making change

	updates := make(devicechangetypes.TypedValueMap)
	updates[Test1Cont1ACont2ALeaf2A] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B314)
	deletes := []string{Test1Cont1ACont2ALeaf2C}
	updatesForDevice1, deletesForDevice1, deviceInfo := makeDeviceChanges(Device1, updates, deletes)

	// Verify the change
	validationError := mgrTest.ValidateNewNetworkConfig(Device1, DeviceVersion, DeviceType, updates, deletes)
	assert.NilError(t, validationError, "ValidateTargetConfig error")

	// Set the new change
	const testNetworkChange network.ID = "Test_SetNetworkConfig"
	err := mgrTest.SetNewNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, string(testNetworkChange))
	assert.NilError(t, err, "SetTargetConfig error")
	testUpdate, _ := mgrTest.NetworkChangesStore.Get(testNetworkChange)
	assert.Assert(t, testUpdate != nil)
	assert.Equal(t, testUpdate.ID, testNetworkChange, "Change Ids should correspond")

	// Check that the created change is correct
	updatedVals := testUpdate.Changes[0].Values
	assert.Equal(t, len(updatedVals), 2)

	// Asserting update to 2A
	assert.Equal(t, updatedVals[0].Path, Test1Cont1ACont2ALeaf2A)
	assert.Equal(t, (*devicechangetypes.TypedFloat)(updatedVals[0].GetValue()).Float32(), float32(ValueLeaf2B314))
	assert.Equal(t, updatedVals[0].Removed, false)

	// Asserting deletion of 2C
	assert.Equal(t, updatedVals[1].Path, Test1Cont1ACont2ALeaf2C)
	assert.Equal(t, (*devicechangetypes.TypedEmpty)(updatedVals[1].GetValue()).String(), "")
	assert.Equal(t, updatedVals[1].Removed, true)
}

// When device type is given it is like extension 102 and allows a never heard of before config to be created
func Test_SetNetworkConfig_NewConfig(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	// Making change

	const Device5 = "Device5"
	const NetworkChangeAddDevice5 = "NetworkChangeAddDevice5"

	updates := make(devicechangetypes.TypedValueMap)
	updates[Test1Cont1ACont2ALeaf2A] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B314)
	deletes := []string{Test1Cont1ACont2ALeaf2C}

	updatesForDevice, deletesForDevice, deviceInfo := makeDeviceChanges(Device5, updates, deletes)

	err := mgrTest.SetNewNetworkConfig(updatesForDevice, deletesForDevice, deviceInfo, NetworkChangeAddDevice5)
	assert.NilError(t, err, "SetTargetConfig error")
	testUpdate, _ := mgrTest.NetworkChangesStore.Get(NetworkChangeAddDevice5)
	assert.Assert(t, testUpdate != nil)
	assert.Assert(t, len(testUpdate.Changes) == 2)
	change1 := testUpdate.Changes[len(testUpdate.Changes)-2]
	change2 := testUpdate.Changes[len(testUpdate.Changes)-1]
	assert.Assert(t, change1 != nil)
	assert.Assert(t, change2 != nil)

	// Asserting update to 2A
	value2A := change1.Values[0]
	assert.Equal(t, value2A.Path, Test1Cont1ACont2ALeaf2A)
	assert.Equal(t, (*devicechangetypes.TypedFloat)(value2A.GetValue()).Float32(), float32(ValueLeaf2B314))
	assert.Equal(t, value2A.Removed, false)

	// Asserting deletion of 2C
	value2C := change2.Values[0]
	assert.Equal(t, value2C.Path, Test1Cont1ACont2ALeaf2C)
	assert.Equal(t, (*devicechangetypes.TypedEmpty)(value2C.GetValue()).String(), "")
	assert.Equal(t, value2C.Removed, true)
}

// When device type is given it is like extension 102 and allows a never heard of before config to be created - here it is missing
func Test_SetNetworkConfig_NewConfig102Missing(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	// Making change
	updates := make(devicechangetypes.TypedValueMap)
	deletes := []string{Test1Cont1ACont2ALeaf2C}
	updates[Test1Cont1ACont2ALeaf2A] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B314)
	updatesForDevice, deletesForDevice, deviceInfo := makeDeviceChanges("Device6", updates, deletes)

	err := mgrTest.SetNewNetworkConfig(updatesForDevice, deletesForDevice, deviceInfo, "Test_SetNetworkConfig_NewConfig")
	// TODO - the error for the missing type is currently not detecting the error of the unknown device type
	t.Skip()
	assert.ErrorContains(t, err, "no configuration found matching 'Device6-1.0.0' and no device type () given Please specify version and device type in extensions 101 and 102")
	//_, okupdate := configurationStoreTest["Device6-1.0.0"]
	//assert.Assert(t, !okupdate, "Expecting not to find Device6-1.0.0")
}

func Test_SetBadNetworkConfig(t *testing.T) {

	mgrTest, _, _, _ := setUp(t)

	updates := make(devicechangetypes.TypedValueMap)
	deletes := []string{Test1Cont1ACont2ALeaf2A, Test1Cont1ACont2ALeaf2C}
	updates[Test1Cont1ACont2ALeaf2B] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B159)
	updatesForDevice1, deletesForDevice1, deviceInfo := makeDeviceChanges(Device1, updates, deletes)

	err := mgrTest.SetNewNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, "Testing")
	// TODO - Storing a new device without extensions 101 and 102 set does not currently throw an error
	// enable test when an error can be seen
	t.Skip()
	assert.ErrorContains(t, err, "no configuration found")
}

func Test_SetMultipleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _, _, _ := setUp(t)

	updates := make(devicechangetypes.TypedValueMap)
	deletes := []string{Test1Cont1ACont2ALeaf2A, Test1Cont1ACont2ALeaf2C}
	updates[Test1Cont1ACont2ALeaf2B] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B159)
	updatesForDevice1, deletesForDevice1, deviceInfo := makeDeviceChanges(Device1, updates, deletes)

	err := mgrTest.SetNewNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, "Testing")

	// TODO - similar configs are currently not detected
	t.Skip()

	assert.ErrorContains(t, err, "configurations found for")
}

func Test_SetSingleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _, _, _ := setUp(t)

	updates := make(devicechangetypes.TypedValueMap)
	deletes := []string{Test1Cont1ACont2ALeaf2A, Test1Cont1ACont2ALeaf2C}
	updates[Test1Cont1ACont2ALeaf2B] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B159)
	updatesForDevice, deletesForDevice, deviceInfo := makeDeviceChanges(Device1, updates, deletes)

	err := mgrTest.SetNewNetworkConfig(updatesForDevice, deletesForDevice, deviceInfo, "Testing")
	assert.NilError(t, err, "Similar config not found")
}

func matchDeviceID(deviceID string, deviceName string) bool {
	return strings.Contains(deviceID, deviceName)
}

func TestManager_GetAllDeviceIds(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	updates := make(devicechangetypes.TypedValueMap)
	updates[Test1Cont1ACont2ALeaf2A] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B314)
	deletes := []string{Test1Cont1ACont2ALeaf2C}
	updatesForDevice2, deletesForDevice2, deviceInfo2 := makeDeviceChanges("Device2", updates, deletes)
	err := mgrTest.SetNewNetworkConfig(updatesForDevice2, deletesForDevice2, deviceInfo2, "Device2")
	assert.NilError(t, err, "SetTargetConfig error")
	updatesForDevice3, deletesForDevice3, deviceInfo3 := makeDeviceChanges("Device2", updates, deletes)
	err = mgrTest.SetNewNetworkConfig(updatesForDevice3, deletesForDevice3, deviceInfo3, "Device3")
	assert.NilError(t, err, "SetTargetConfig error")

	deviceIds := mgrTest.GetAllDeviceIds()
	// TODO - GetAllDeviceIds() doesn't use the Atomix based stores yet
	t.Skip()
	assert.Equal(t, len(*deviceIds), 3)
	assert.Assert(t, matchDeviceID((*deviceIds)[0], "Device1"))
	assert.Assert(t, matchDeviceID((*deviceIds)[1], "Device2"))
	assert.Assert(t, matchDeviceID((*deviceIds)[2], "Device3"))
}

func TestManager_GetNoConfig(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetNewConfig("No Such Device", DeviceVersion, "/*", 0)
	assert.Assert(t, len(result) == 0, "Get of bad device does not return empty array")
	assert.ErrorContains(t, err, "no Configuration found")
}

func networkConfigContainsPath(configs []*devicechangetypes.PathValue, whichOne string) bool {
	for _, config := range configs {
		if config.Path == whichOne {
			return true
		}
	}
	return false
}

func TestManager_GetAllConfig(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetNewConfig("Device1", DeviceVersion, "/*", 0)
	assert.Assert(t, len(result) == 3, "Get of device all paths does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a"), "/cont1a not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a"), "/cont1a/cont2a not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetOneConfig(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetNewConfig("Device1", DeviceVersion, "/cont1a/cont2a/leaf2a", 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetWildcardConfig(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetNewConfig("Device1", DeviceVersion, "/*/*/leaf2a", 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, "/cont1a/cont2a/leaf2a"), "/cont1a/cont2a/leaf2a not found")
}

func TestManager_GetConfigNoTarget(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	result, err := mgrTest.GetTargetNewConfig("", DeviceVersion, "/cont1a/cont2a/leaf2a", 0)
	assert.Assert(t, len(result) == 0, "Get of device one path does not return proper array")
	assert.ErrorContains(t, err, "no Configuration found")
}

func TestManager_GetManager(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)
	assert.Equal(t, mgrTest, GetManager())
	GetManager().Close()
	assert.Equal(t, mgrTest, GetManager())
}

func valuesContainsPath(path string, values []*types.ChangeValue) bool {
	for _, value := range values {
		if value.Path == path {
			return true
		}
	}
	return false
}

func TestManager_ComputeRollbackDelete(t *testing.T) {
	mgrTest, _, _, _ := setUp(t)

	updates := make(devicechangetypes.TypedValueMap)
	deletes := make([]string, 0)
	updates[Test1Cont1ACont2ALeaf2B] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B159)

	updatesForDevice1, deletesForDevice1, deviceInfo := makeDeviceChanges(Device1, updates, deletes)

	err := mgrTest.ValidateNewNetworkConfig(Device1, DeviceVersion, DeviceType, updates, deletes)
	assert.NilError(t, err, "ValidateTargetConfig error")
	err = mgrTest.SetNewNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, "TestingRollback")
	assert.NilError(t, err, "Can't create change", err)

	updates[Test1Cont1ACont2ALeaf2B] = devicechangetypes.NewTypedValueFloat(ValueLeaf2B314)
	updates[Test1Cont1ACont2ALeaf2D] = devicechangetypes.NewTypedValueFloat(ValueLeaf2D314)
	deletes = append(deletes, Test1Cont1ACont2ALeaf2A)

	err = mgrTest.ValidateNewNetworkConfig(Device1, DeviceVersion, DeviceType, updates, deletes)
	assert.NilError(t, err, "ValidateTargetConfig error")

	updatesForDevice1, deletesForDevice1, deviceInfo = makeDeviceChanges(Device1, updates, deletes)
	err = mgrTest.SetNewNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, "TestingRollback2")
	assert.NilError(t, err, "Can't create change")

	err = mgrTest.NewRollbackTargetConfig("TestingRollback2")
	assert.NilError(t, err, "Can't roll back change")
	rbChange, _ := mgrTest.NetworkChangesStore.Get("TestingRollback2")
	assert.Assert(t, rbChange != nil)

	assert.Assert(t, len(rbChange.Changes) == 2)
	assert.Assert(t, strings.Contains(rbChange.Status.Message, "requested rollback"))
	assert.Assert(t, valuesContainsPath(Test1Cont1ACont2ALeaf2D, rbChange.Changes[0].Values))
	assert.Assert(t, !rbChange.Changes[0].Values[0].Removed)
	assert.Assert(t, valuesContainsPath(Test1Cont1ACont2ALeaf2B, rbChange.Changes[0].Values))
	assert.Assert(t, !rbChange.Changes[0].Values[0].Removed)
	assert.Assert(t, valuesContainsPath(Test1Cont1ACont2ALeaf2A, rbChange.Changes[0].Values))
	assert.Assert(t, !rbChange.Changes[0].Values[0].Removed)

	assert.Assert(t, valuesContainsPath(Test1Cont1ACont2ALeaf2A, rbChange.Changes[1].Values))
	assert.Assert(t, rbChange.Changes[1].Values[0].Removed)
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
	device1ValueMap := make(map[string]*devicechangetypes.TypedValue)
	change1 := devicechangetypes.NewTypedValueString(value1)
	change2 := devicechangetypes.NewTypedValueString(value2)
	device1ValueMap[path1] = change1
	device1ValueMap[path2] = change2
	mgrTest.OperationalStateCache[device1] = device1ValueMap

	// Test fetching a known path from the cache
	state1 := mgrTest.GetTargetState(device1, path1WC)
	assert.Assert(t, state1 != nil, "Path 1 entry not found")
	assert.Assert(t, len(state1) == 1, "Path 1 entry has incorrect length %d", len(state1))
	assert.Assert(t, state1[0].Path == path1, "Path 1 path is incorrect")
	assert.Assert(t, string(state1[0].GetValue().GetBytes()) == value1, "Path 1 value is incorrect")

	// Test fetching an unknown path from the cache
	stateBad := mgrTest.GetTargetState(device1, badPath)
	assert.Assert(t, stateBad != nil, "Bad Path Entry returns nil")
	assert.Assert(t, len(stateBad) == 0, "Bad path entry has incorrect length %d", len(stateBad))
}

func TestManager_DeviceConnected(t *testing.T) {
	mgrTest, mockStore, _, _ := setUp(t)
	const (
		device1 = "device1"
	)

	deviceDisconnected := &devicetopo.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  DeviceVersion,
	}

	device1Connected := &devicetopo.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  DeviceVersion,
	}

	mockStore.EXPECT().Get("device1")

	protocolState := new(devicetopo.ProtocolState)
	protocolState.Protocol = devicetopo.Protocol_GNMI
	protocolState.ConnectivityState = devicetopo.ConnectivityState_REACHABLE
	protocolState.ChannelState = devicetopo.ChannelState_CONNECTED
	protocolState.ServiceState = devicetopo.ServiceState_AVAILABLE
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)

	mockStore.EXPECT().Get(gomock.Any()).Return(deviceDisconnected, nil)
	mockStore.EXPECT().Update(gomock.Any()).Return(device1Connected, nil)

	deviceConnected, err := mgrTest.DeviceConnected(device1)

	assert.NilError(t, err)
	assert.Equal(t, deviceConnected.ID, device1Connected.ID)
	assert.Equal(t, deviceConnected.Protocols[0].Protocol, devicetopo.Protocol_GNMI)
	assert.Equal(t, deviceConnected.Protocols[0].ConnectivityState, devicetopo.ConnectivityState_REACHABLE)
	assert.Equal(t, deviceConnected.Protocols[0].ChannelState, devicetopo.ChannelState_CONNECTED)
	assert.Equal(t, deviceConnected.Protocols[0].ServiceState, devicetopo.ServiceState_AVAILABLE)
}

func TestManager_DeviceDisconnected(t *testing.T) {
	mgrTest, mockStore, _, _ := setUp(t)
	const (
		device1 = "device1"
	)

	deviceDisconnected := &devicetopo.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  DeviceVersion,
	}

	device1Connected := &devicetopo.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  DeviceVersion,
	}

	mockStore.EXPECT().Get("device1")

	protocolState := new(devicetopo.ProtocolState)
	protocolState.Protocol = devicetopo.Protocol_GNMI
	protocolState.ConnectivityState = devicetopo.ConnectivityState_REACHABLE
	protocolState.ChannelState = devicetopo.ChannelState_CONNECTED
	protocolState.ServiceState = devicetopo.ServiceState_AVAILABLE
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)

	protocolStateDisconnected := new(devicetopo.ProtocolState)
	protocolStateDisconnected.Protocol = devicetopo.Protocol_GNMI
	protocolStateDisconnected.ConnectivityState = devicetopo.ConnectivityState_UNREACHABLE
	protocolStateDisconnected.ChannelState = devicetopo.ChannelState_DISCONNECTED
	protocolStateDisconnected.ServiceState = devicetopo.ServiceState_UNAVAILABLE
	deviceDisconnected.Protocols = append(device1Connected.Protocols, protocolState)

	mockStore.EXPECT().Get(gomock.Any()).Return(device1Connected, nil)
	mockStore.EXPECT().Update(gomock.Any()).Return(deviceDisconnected, nil)

	deviceUpdated, err := mgrTest.DeviceDisconnected(device1, errors.New("device reported disconnection"))

	assert.NilError(t, err)
	assert.Equal(t, deviceUpdated.ID, device1Connected.ID)
	assert.Equal(t, deviceUpdated.Protocols[0].Protocol, devicetopo.Protocol_GNMI)
	assert.Equal(t, deviceUpdated.Protocols[0].ConnectivityState, devicetopo.ConnectivityState_UNREACHABLE)
	assert.Equal(t, deviceUpdated.Protocols[0].ChannelState, devicetopo.ChannelState_DISCONNECTED)
	assert.Equal(t, deviceUpdated.Protocols[0].ServiceState, devicetopo.ServiceState_UNAVAILABLE)
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

func (m MockModelPlugin) GetStateMode() int {
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
	// TODO - Not implemented on Atomix stores yet
	t.Skip()
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
