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
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	networkstore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	topodevice "github.com/onosproject/onos-topo/api/device"
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
	test1Cont1ACont2ALeaf2A = "/cont1a/cont2a/leaf2a"
	test1Cont1ACont2ALeaf2B = "/cont1a/cont2a/leaf2b"
	test1Cont1ACont2ALeaf2C = "/cont1a/cont2a/leaf2c"
	test1Cont1ACont2ALeaf2D = "/cont1a/cont2a/leaf2d"
)

const (
	valueLeaf2A789 = uint(789)
	valueLeaf2B159 = 1.579
	valueLeaf2B314 = 3.14
	valueLeaf2D123 = 1.23
)

const (
	device1                         = "Device1"
	deviceVersion1                  = "1.0.0"
	deviceTypeTd                    = "TestDevice"
	networkChange1 networkchange.ID = "NetworkChange1"
	deviceChange1  devicechange.ID  = "DeviceChange1"
)

type AllMocks struct {
	MockStores      *mockstore.MockStores
	MockDeviceCache *devicestore.MockCache
}

func setUp(t *testing.T) (*Manager, *AllMocks) {
	log.SetOutput(os.Stdout)
	var (
		mgrTest  *Manager
		err      error
		allMocks AllMocks
	)

	config1Value03, _ := devicechange.NewChangeValue(test1Cont1ACont2ALeaf2A, devicechange.NewTypedValueFloat(valueLeaf2B159), false)

	ctrl := gomock.NewController(t)

	mockDeviceCache := devicestore.NewMockCache(ctrl)
	mockDeviceCache.EXPECT().Watch(gomock.Any()).DoAndReturn(
		func(ch chan<- *devicestore.Info) {
			ch <- &devicestore.Info{
				DeviceID: device1,
				Type:     deviceTypeTd,
				Version:  deviceVersion1,
			}
		},
	)
	// Data for default configuration

	change1 := devicechange.Change{
		Values: []*devicechange.ChangeValue{
			config1Value03},
		DeviceID:      device1,
		DeviceVersion: deviceVersion1,
	}
	deviceChange1 := &devicechange.DeviceChange{
		Change: &change1,
		ID:     deviceChange1,
		Status: changetypes.Status{State: changetypes.State_COMPLETE},
	}

	now := time.Now()
	networkChange1 := &networkchange.NetworkChange{
		ID:      networkChange1,
		Changes: []*devicechange.Change{&change1},
		Updated: now,
		Created: now,
		Status:  changetypes.Status{State: changetypes.State_COMPLETE},
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
	networkChangesList := make([]*networkchange.NetworkChange, 0)
	mockNetworkChangesStore := mockstore.NewMockNetworkChangesStore(ctrl)
	mockNetworkChangesStore.EXPECT().Create(gomock.Any()).DoAndReturn(
		func(networkChange *networkchange.NetworkChange) error {
			networkChangesList = append(networkChangesList, networkChange)
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Update(gomock.Any()).DoAndReturn(
		func(networkChange *networkchange.NetworkChange) error {
			networkChangesList = append(networkChangesList, networkChange)
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(id networkchange.ID) (*networkchange.NetworkChange, error) {
			var found *networkchange.NetworkChange
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
		func(c chan<- *networkchange.NetworkChange) error {
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
		func(deviceID devicetype.VersionedID, c chan<- *devicechange.DeviceChange) (stream.Context, error) {
			changes := make([]*devicechange.DeviceChange, 0)
			for _, deviceChange := range deviceChanges {
				if devicetype.NewVersionedID(deviceChange.Change.DeviceID, deviceChange.Change.DeviceVersion).GetID() == deviceID.GetID() {
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
		mockLeadershipStore,
		mockMastershipStore,
		mockDeviceChangesStore,
		mockDeviceStore,
		mockDeviceCache,
		mockNetworkChangesStore,
		mockNetworkSnapshotStore,
		mockDeviceSnapshotStore,
		make(chan *topodevice.ListResponse, 10))
	if err != nil {
		log.Warning(err)
		os.Exit(-1)
	}
	mgrTest.Run()

	mockStores := &mockstore.MockStores{
		DeviceStore:          mockDeviceStore,
		NetworkChangesStore:  mockNetworkChangesStore,
		DeviceChangesStore:   mockDeviceChangesStore,
		NetworkSnapshotStore: mockNetworkSnapshotStore,
		DeviceSnapshotStore:  mockDeviceSnapshotStore,
		LeadershipStore:      mockLeadershipStore,
		MastershipStore:      mockMastershipStore,
	}

	allMocks.MockStores = mockStores
	allMocks.MockDeviceCache = mockDeviceCache
	return mgrTest, &allMocks
}

func makeDeviceChanges(device string, updates devicechange.TypedValueMap, deletes []string) (
	map[string]devicechange.TypedValueMap, map[string][]string, map[devicetype.ID]devicestore.Info) {
	deviceInfo := make(map[devicetype.ID]devicestore.Info)
	deviceInfo[devicetype.ID(device)] = devicestore.Info{DeviceID: devicetype.ID(device), Type: deviceTypeTd, Version: deviceVersion1}

	updatesForDevice := make(map[string]devicechange.TypedValueMap)
	updatesForDevice[device] = updates
	deletesForDevice := make(map[string][]string)
	deletesForDevice[device] = deletes
	return updatesForDevice, deletesForDevice, deviceInfo
}

func Test_GetNetworkConfig(t *testing.T) {

	mgrTest, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig(device1, deviceVersion1, "/*", 0)
	assert.NilError(t, err, "GetTargetConfig error")

	assert.Equal(t, len(result), 1, "Unexpected result element count")

	assert.Equal(t, result[0].Path, test1Cont1ACont2ALeaf2A, "result %s is different")
}

func Test_SetNetworkConfig(t *testing.T) {
	mgrTest, _ := setUp(t)

	// First verify the value beforehand
	originalChange, _ := mgrTest.NetworkChangesStore.Get(networkChange1)
	assert.Equal(t, len(originalChange.Changes[0].Values), 1)
	assert.Equal(t, originalChange.Changes[0].Values[0].Path, test1Cont1ACont2ALeaf2A)
	assert.Equal(t, originalChange.Changes[0].Values[0].Value.Type, devicechange.ValueType_FLOAT)
	assert.Equal(t, (*devicechange.TypedFloat)(originalChange.Changes[0].Values[0].Value).Float32(), float32(valueLeaf2B159))

	// Making change
	updates := make(devicechange.TypedValueMap)
	updates[test1Cont1ACont2ALeaf2A] = devicechange.NewTypedValueUint64(valueLeaf2A789)
	deletes := []string{test1Cont1ACont2ALeaf2C}
	updatesForDevice1, deletesForDevice1, deviceInfo := makeDeviceChanges(device1, updates, deletes)

	// Verify the change
	validationError := mgrTest.ValidateNetworkConfig(device1, deviceVersion1, deviceTypeTd, updates, deletes)
	assert.NilError(t, validationError, "ValidateTargetConfig error")

	// Set the new change
	const testNetworkChange networkchange.ID = "Test_SetNetworkConfig"
	err := mgrTest.SetNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, string(testNetworkChange))
	assert.NilError(t, err, "SetTargetConfig error")
	testUpdate, _ := mgrTest.NetworkChangesStore.Get(testNetworkChange)
	assert.Assert(t, testUpdate != nil)
	assert.Equal(t, testUpdate.ID, testNetworkChange, "Change Ids should correspond")

	// Check that the created change is correct
	updatedVals := testUpdate.Changes[0].Values
	assert.Equal(t, len(updatedVals), 2)

	// Asserting update to 2A
	assert.Equal(t, updatedVals[0].Path, test1Cont1ACont2ALeaf2A)
	assert.Equal(t, (*devicechange.TypedUint64)(updatedVals[0].GetValue()).Uint(), valueLeaf2A789)
	assert.Equal(t, updatedVals[0].Removed, false)

	// Asserting deletion of 2C
	assert.Equal(t, updatedVals[1].Path, test1Cont1ACont2ALeaf2C)
	assert.Equal(t, updatedVals[1].GetValue().ValueToString(), "")
	assert.Equal(t, updatedVals[1].Removed, true)
}

// When device type is given it is like extension 102 and allows a never heard of before config to be created
func Test_SetNetworkConfig_NewConfig(t *testing.T) {
	mgrTest, _ := setUp(t)

	// Making change
	const Device5 = "Device5"
	const NetworkChangeAddDevice5 = "NetworkChangeAddDevice5"

	updates := make(devicechange.TypedValueMap)
	updates[test1Cont1ACont2ALeaf2A] = devicechange.NewTypedValueUint64(valueLeaf2A789)
	deletes := []string{test1Cont1ACont2ALeaf2C}

	updatesForDevice, deletesForDevice, deviceInfo := makeDeviceChanges(Device5, updates, deletes)

	err := mgrTest.SetNetworkConfig(updatesForDevice, deletesForDevice, deviceInfo, NetworkChangeAddDevice5)
	assert.NilError(t, err, "SetTargetConfig error")
	testUpdate, _ := mgrTest.NetworkChangesStore.Get(NetworkChangeAddDevice5)
	assert.Assert(t, testUpdate != nil)
	assert.Equal(t, 1, len(testUpdate.Changes))
	change1 := testUpdate.Changes[0]
	assert.Assert(t, change1 != nil)

	// Asserting update to 2A
	value2A := change1.Values[0]
	assert.Equal(t, value2A.Path, test1Cont1ACont2ALeaf2A)
	assert.Equal(t, value2A.GetValue().ValueToString(), "789")
	assert.Equal(t, value2A.Removed, false)

	// Asserting deletion of 2C
	value2C := change1.Values[1]
	assert.Equal(t, value2C.Path, test1Cont1ACont2ALeaf2C)
	assert.Equal(t, value2C.GetValue().ValueToString(), "")
	assert.Equal(t, value2C.Removed, true)
}

func Test_SetMultipleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _ := setUp(t)

	updates := make(devicechange.TypedValueMap)
	deletes := []string{test1Cont1ACont2ALeaf2A, test1Cont1ACont2ALeaf2C}
	updates[test1Cont1ACont2ALeaf2B] = devicechange.NewTypedValueFloat(valueLeaf2B159)
	updatesForDevice1, deletesForDevice1, deviceInfo := makeDeviceChanges(device1, updates, deletes)

	err := mgrTest.SetNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, "Testing")

	// TODO - similar configs are currently not detected
	t.Skip()

	assert.ErrorContains(t, err, "configurations found for")
}

func Test_SetSingleSimilarNetworkConfig(t *testing.T) {

	mgrTest, _ := setUp(t)

	updates := make(devicechange.TypedValueMap)
	deletes := []string{test1Cont1ACont2ALeaf2A, test1Cont1ACont2ALeaf2C}
	updates[test1Cont1ACont2ALeaf2B] = devicechange.NewTypedValueFloat(valueLeaf2B159)
	updatesForDevice, deletesForDevice, deviceInfo := makeDeviceChanges(device1, updates, deletes)

	err := mgrTest.SetNetworkConfig(updatesForDevice, deletesForDevice, deviceInfo, "Testing")
	assert.NilError(t, err, "Similar config not found")
}

func matchDeviceID(deviceID string, deviceName string) bool {
	return strings.Contains(deviceID, deviceName)
}

func TestManager_GetAllDeviceIds(t *testing.T) {
	// TODO - GetAllDeviceIds() uses atomix, needs better mocking
	t.Skip("TODO - mock for atomix")
	mgrTest, mocks := setUp(t)

	updates := make(devicechange.TypedValueMap)
	updates[test1Cont1ACont2ALeaf2A] = devicechange.NewTypedValueFloat(valueLeaf2B314)
	deletes := []string{test1Cont1ACont2ALeaf2C}
	updatesForDevice2, deletesForDevice2, deviceInfo2 := makeDeviceChanges("Device2", updates, deletes)
	err := mgrTest.SetNetworkConfig(updatesForDevice2, deletesForDevice2, deviceInfo2, "Device2")
	assert.NilError(t, err, "SetTargetConfig error")
	updatesForDevice3, deletesForDevice3, deviceInfo3 := makeDeviceChanges("Device2", updates, deletes)
	err = mgrTest.SetNetworkConfig(updatesForDevice3, deletesForDevice3, deviceInfo3, "Device3")
	assert.NilError(t, err, "SetTargetConfig error")
	mocks.MockStores.DeviceStore.EXPECT().List(gomock.Any()).AnyTimes()
	deviceIds := mgrTest.GetAllDeviceIds()

	assert.Equal(t, len(*deviceIds), 3)
	assert.Assert(t, matchDeviceID((*deviceIds)[0], device1))
	assert.Assert(t, matchDeviceID((*deviceIds)[1], "Device2"))
	assert.Assert(t, matchDeviceID((*deviceIds)[2], "Device3"))
}

func TestManager_GetNoConfig(t *testing.T) {
	mgrTest, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig("No Such Device", deviceVersion1, "/*", 0)
	assert.Assert(t, len(result) == 0, "Get of bad device does not return empty array")
	assert.ErrorContains(t, err, "no Configuration found")
}

func networkConfigContainsPath(configs []*devicechange.PathValue, whichOne string) bool {
	for _, config := range configs {
		if config.Path == whichOne {
			return true
		}
	}
	return false
}

func TestManager_GetAllConfig(t *testing.T) {
	mgrTest, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig(device1, deviceVersion1, "/*", 0)
	assert.Assert(t, len(result) == 1, "Get of device all paths does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, test1Cont1ACont2ALeaf2A), test1Cont1ACont2ALeaf2A+" not found")
}

func TestManager_GetOneConfig(t *testing.T) {
	mgrTest, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig(device1, deviceVersion1, test1Cont1ACont2ALeaf2A, 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, test1Cont1ACont2ALeaf2A), test1Cont1ACont2ALeaf2A+" not found")
}

func TestManager_GetWildcardConfig(t *testing.T) {
	mgrTest, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig(device1, deviceVersion1, "/*/*/leaf2a", 0)
	assert.Assert(t, len(result) == 1, "Get of device one path does not return proper array")
	assert.NilError(t, err, "Configuration not found")
	assert.Assert(t, networkConfigContainsPath(result, test1Cont1ACont2ALeaf2A), test1Cont1ACont2ALeaf2A+" not found")
}

func TestManager_GetConfigNoTarget(t *testing.T) {
	mgrTest, _ := setUp(t)

	result, err := mgrTest.GetTargetConfig("", deviceVersion1, test1Cont1ACont2ALeaf2A, 0)
	assert.Assert(t, len(result) == 0, "Get of device one path does not return proper array")
	assert.ErrorContains(t, err, "no Configuration found")
}

func TestManager_GetManager(t *testing.T) {
	mgrTest, _ := setUp(t)
	assert.Equal(t, mgrTest, GetManager())
	GetManager().Close()
	assert.Equal(t, mgrTest, GetManager())
}

func TestManager_ComputeRollbackDelete(t *testing.T) {
	mgrTest, _ := setUp(t)

	updates := make(devicechange.TypedValueMap)
	deletes := make([]string, 0)
	updates[test1Cont1ACont2ALeaf2B] = devicechange.NewTypedValueFloat(valueLeaf2B159)

	updatesForDevice1, deletesForDevice1, deviceInfo := makeDeviceChanges(device1, updates, deletes)

	err := mgrTest.ValidateNetworkConfig(device1, deviceVersion1, deviceTypeTd, updates, deletes)
	assert.NilError(t, err, "ValidateTargetConfig error")
	err = mgrTest.SetNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, "TestingRollback")
	assert.NilError(t, err, "Can't create change", err)

	updates[test1Cont1ACont2ALeaf2B] = devicechange.NewTypedValueFloat(valueLeaf2B314)
	updates[test1Cont1ACont2ALeaf2D] = devicechange.NewTypedValueFloat(valueLeaf2D123)
	deletes = append(deletes, test1Cont1ACont2ALeaf2A)

	err = mgrTest.ValidateNetworkConfig(device1, deviceVersion1, deviceTypeTd, updates, deletes)
	assert.NilError(t, err, "ValidateTargetConfig error")

	updatesForDevice1, deletesForDevice1, deviceInfo = makeDeviceChanges(device1, updates, deletes)
	err = mgrTest.SetNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo, "TestingRollback2")
	assert.NilError(t, err, "Can't create change")

	err = mgrTest.RollbackTargetConfig("TestingRollback2")
	assert.NilError(t, err, "Can't roll back change")
	rbChange, _ := mgrTest.NetworkChangesStore.Get("TestingRollback2")
	assert.Assert(t, rbChange != nil)

	assert.Assert(t, len(rbChange.Changes) == 1)
	assert.Assert(t, strings.Contains(rbChange.Status.Message, "requested rollback"))
	assert.Assert(t, len(rbChange.Changes[0].Values) == 3)
	for _, v := range rbChange.Changes[0].Values {
		switch v.Path {
		case test1Cont1ACont2ALeaf2A:
			assert.Assert(t, v.Removed)
			assert.Equal(t, "", v.Value.ValueToString())
		case test1Cont1ACont2ALeaf2B:
			assert.Assert(t, !v.Removed)
			assert.Equal(t, "3.140000", v.Value.ValueToString())
		case test1Cont1ACont2ALeaf2D:
			assert.Assert(t, !v.Removed)
			assert.Equal(t, "1.230000", v.Value.ValueToString())
		default:
			t.Errorf("Unexpected path %s", v.Path)
		}
	}
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
	mgrTest, _ := setUp(t)

	//  Create an op state map with test data
	device1ValueMap := make(map[string]*devicechange.TypedValue)
	change1 := devicechange.NewTypedValueString(value1)
	change2 := devicechange.NewTypedValueString(value2)
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
	mgrTest, mocks := setUp(t)
	const (
		device1 = "device1"
	)

	deviceDisconnected := &topodevice.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  deviceVersion1,
	}

	device1Connected := &topodevice.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  deviceVersion1,
	}

	mocks.MockStores.DeviceStore.EXPECT().Get("device1")

	protocolState := new(topodevice.ProtocolState)
	protocolState.Protocol = topodevice.Protocol_GNMI
	protocolState.ConnectivityState = topodevice.ConnectivityState_REACHABLE
	protocolState.ChannelState = topodevice.ChannelState_CONNECTED
	protocolState.ServiceState = topodevice.ServiceState_AVAILABLE
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)

	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(deviceDisconnected, nil)
	mocks.MockStores.DeviceStore.EXPECT().Update(gomock.Any()).Return(device1Connected, nil)

	deviceConnected, err := mgrTest.DeviceConnected(device1)

	assert.NilError(t, err)
	assert.Equal(t, deviceConnected.ID, device1Connected.ID)
	assert.Equal(t, deviceConnected.Protocols[0].Protocol, topodevice.Protocol_GNMI)
	assert.Equal(t, deviceConnected.Protocols[0].ConnectivityState, topodevice.ConnectivityState_REACHABLE)
	assert.Equal(t, deviceConnected.Protocols[0].ChannelState, topodevice.ChannelState_CONNECTED)
	assert.Equal(t, deviceConnected.Protocols[0].ServiceState, topodevice.ServiceState_AVAILABLE)
}

func TestManager_DeviceDisconnected(t *testing.T) {
	mgrTest, mocks := setUp(t)
	const (
		device1 = "device1"
	)

	deviceDisconnected := &topodevice.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  deviceVersion1,
	}

	device1Connected := &topodevice.Device{
		ID:       "device1",
		Revision: 1,
		Address:  "device1:1234",
		Version:  deviceVersion1,
	}

	mocks.MockStores.DeviceStore.EXPECT().Get("device1")

	protocolState := new(topodevice.ProtocolState)
	protocolState.Protocol = topodevice.Protocol_GNMI
	protocolState.ConnectivityState = topodevice.ConnectivityState_REACHABLE
	protocolState.ChannelState = topodevice.ChannelState_CONNECTED
	protocolState.ServiceState = topodevice.ServiceState_AVAILABLE
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)

	protocolStateDisconnected := new(topodevice.ProtocolState)
	protocolStateDisconnected.Protocol = topodevice.Protocol_GNMI
	protocolStateDisconnected.ConnectivityState = topodevice.ConnectivityState_UNREACHABLE
	protocolStateDisconnected.ChannelState = topodevice.ChannelState_DISCONNECTED
	protocolStateDisconnected.ServiceState = topodevice.ServiceState_UNAVAILABLE
	deviceDisconnected.Protocols = append(device1Connected.Protocols, protocolState)

	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(device1Connected, nil)
	mocks.MockStores.DeviceStore.EXPECT().Update(gomock.Any()).Return(deviceDisconnected, nil)

	deviceUpdated, err := mgrTest.DeviceDisconnected(device1, errors.New("device reported disconnection"))

	assert.NilError(t, err)
	assert.Equal(t, deviceUpdated.ID, device1Connected.ID)
	assert.Equal(t, deviceUpdated.Protocols[0].Protocol, topodevice.Protocol_GNMI)
	assert.Equal(t, deviceUpdated.Protocols[0].ConnectivityState, topodevice.ConnectivityState_UNREACHABLE)
	assert.Equal(t, deviceUpdated.Protocols[0].ChannelState, topodevice.ChannelState_DISCONNECTED)
	assert.Equal(t, deviceUpdated.Protocols[0].ServiceState, topodevice.ServiceState_UNAVAILABLE)
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
	mgrTest, _ := setUp(t)

	plugin := MockModelPlugin{}
	mgrTest.ModelRegistry.ModelPlugins["TestDevice-1.0.0"] = plugin

	roPathMap := make(modelregistry.ReadOnlyPathMap)
	roSubPath1 := make(modelregistry.ReadOnlySubPathMap)
	roPathMap[test1Cont1ACont2ALeaf2A] = roSubPath1

	mgr.ModelRegistry.ModelReadOnlyPaths["TestDevice-1.0.0"] = roPathMap

	// TODO - Not implemented on Atomix stores yet
	t.Skip()
	//assert.ErrorContains(t, validationError,
	//	"error read only path in configuration /cont1a/cont2a/leaf2a matches /cont1a/cont2a/leaf2a for TestDevice-1.0.0")
}

func TestManager_ValidateStores(t *testing.T) {
	t.Skip("TODO re-enable when validation is done on atomix stores")
	mgrTest, _ := setUp(t)

	plugin := MockModelPlugin{}
	mgrTest.ModelRegistry.ModelPlugins["TestDevice-1.0.0"] = plugin

	//assert.NilError(t, validationError)
}

func TestManager_CheckCacheForDevice(t *testing.T) {
	mgrTest, mocks := setUp(t)

	const (
		deviceTest1 = "DeviceTest1"
		v1          = "1.0.0"
		v2          = "2.0.0"
		v3          = "3.0.0"
		tdType      = "TestDevice"
		deviceTest2 = "DeviceTest2"
		dsType      = "Devicesim"
		deviceTest3 = "DeviceTest3"
	)

	deviceInfos := []*devicestore.Info{
		{
			DeviceID: deviceTest1,
			Version:  v1,
			Type:     tdType,
		},
		{
			DeviceID: deviceTest1,
			Version:  v2,
			Type:     tdType,
		},
		{
			DeviceID: deviceTest2,
			Version:  v1,
			Type:     dsType,
		},
	}

	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID(deviceTest1)).Return(deviceInfos[:2]).AnyTimes()
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID(deviceTest2)).Return(deviceInfos[2:]).AnyTimes()
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(gomock.Any()).Return(make([]*devicestore.Info, 0)).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(topodevice.ID(deviceTest1)).Return(&topodevice.Device{
		ID:      deviceTest1,
		Address: "1.2.3.4",
		Version: v1,
		Type:    tdType,
	}, nil).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(topodevice.ID(deviceTest2)).Return(&topodevice.Device{
		ID:      deviceTest2,
		Address: "1.2.3.4",
		Version: v1,
		Type:    dsType,
	}, nil).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(topodevice.ID(deviceTest3)).Return(nil, nil).AnyTimes()

	/********************************************************************
	 * deviceTest1 v1.0.0
	 *****************************************************************/
	// Definite - valid type and version given
	ty, v, err := mgrTest.CheckCacheForDevice(deviceTest1, tdType, v1)
	assert.NilError(t, err, "testing cache access")
	assert.Equal(t, tdType, string(ty))
	assert.Equal(t, v1, string(v))

	// Slightly ambiguous - valid version given
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest1, "", v1)
	assert.NilError(t, err, "testing cache access")
	assert.Equal(t, tdType, string(ty))
	assert.Equal(t, v1, string(v))

	// Fully ambiguous - no version or type given
	_, _, err = mgrTest.CheckCacheForDevice(deviceTest1, "", "")
	assert.ErrorContains(t, err, "DeviceTest1 has 2 versions. Specify 1 version with extension 102")

	/********************************************************************
	 * deviceTest1 v2.0.0
	 *****************************************************************/
	// Definite - valid type and version given
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest1, tdType, v2)
	assert.NilError(t, err, "testing cache access")
	assert.Equal(t, tdType, string(ty))
	assert.Equal(t, v2, string(v))

	// Slightly ambiguous - valid version given
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest1, "", v2)
	assert.NilError(t, err, "testing cache access")
	assert.Equal(t, tdType, string(ty))
	assert.Equal(t, v2, string(v))

	// Fully ambiguous - no version or type given
	_, _, err = mgrTest.CheckCacheForDevice(deviceTest1, "", "")
	assert.ErrorContains(t, err, "DeviceTest1 has 2 versions. Specify 1 version with extension 102")

	// Wrong device type given - ignored
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest1, dsType, v1)
	assert.NilError(t, err, "testing wrong type")
	assert.Equal(t, tdType, string(ty))
	assert.Equal(t, v1, string(v))

	/********************************************************************
	 * deviceTest2 v3.0.0
	 *****************************************************************/
	// New version given with valid type - accepted
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest1, tdType, v3)
	assert.NilError(t, err, "testing new version")
	assert.Equal(t, tdType, string(ty))
	assert.Equal(t, v3, string(v))

	// New version given but invalid type - rejected
	_, _, err = mgrTest.CheckCacheForDevice(deviceTest1, dsType, v3)
	assert.ErrorContains(t, err, "DeviceTest1 type given Devicesim does not match expected TestDevice")

	/********************************************************************
	 * deviceTest2 v1.0.0
	 *****************************************************************/
	// Definite - valid type and version given
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest2, dsType, v1)
	assert.NilError(t, err, "testing cache access")
	assert.Equal(t, dsType, string(ty))
	assert.Equal(t, v1, string(v))

	// Slightly ambiguous - valid version given
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest2, "", v1)
	assert.NilError(t, err, "testing cache access")
	assert.Equal(t, dsType, string(ty))
	assert.Equal(t, v1, string(v))

	// Fully ambiguous - no version or type given - but there's only 1 so it's OK
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest2, "", "")
	assert.NilError(t, err, "testing cache access")
	assert.Equal(t, dsType, string(ty))
	assert.Equal(t, v1, string(v))

	/********************************************************************
	 * deviceTest3 v1.0.0
	 *****************************************************************/
	// Definite - valid type and version given
	ty, v, err = mgrTest.CheckCacheForDevice(deviceTest3, dsType, v1)
	assert.NilError(t, err, "testing cache access")
	assert.Equal(t, dsType, string(ty))
	assert.Equal(t, v1, string(v))

	// Slightly ambiguous - valid version given
	_, _, err = mgrTest.CheckCacheForDevice(deviceTest3, "", v1)
	assert.ErrorContains(t, err, "DeviceTest3 is not known. Need to supply a type and version through Extensions 101 and 102")

	// Fully ambiguous - no version or type given - but there's only 1 so it's OK
	_, _, err = mgrTest.CheckCacheForDevice(deviceTest3, "", "")
	assert.ErrorContains(t, err, "DeviceTest3 is not known. Need to supply a type and version through Extensions 101 and 102")
}
