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
	"context"
	"github.com/golang/mock/gomock"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/pkg/southbound"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	networkstore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	southboundmocks "github.com/onosproject/onos-config/pkg/test/mocks/southbound"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/onosproject/onos-config/pkg/utils"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"io/ioutil"
	log "k8s.io/klog"
	"os"
	"testing"
	"time"
)

const (
	deviceConfigOnly = "device-config-only"
	gnmiVer070       = "0.7.0"
)

// setUpDeepTest uses local atomix stores like mocks at a deep level. It allows
// all of the interactions between the manager and the networkchange stores, controllers
// watchers and reconcilers, and the devicechange stores, controllers, watchers,
// reconcilers, filters and their interaction with the synchronizer to become
// part of the test. With this only the DeviceStore (a proxy for onos-topo) and
// the 'target' (the gnmi client to the device) are mocked.
func setUpDeepTest(t *testing.T) (*Manager, *AllMocks) {
	log.SetOutput(os.Stdout)
	var (
		mgrTest *Manager
		err     error
	)

	config1Value03, _ := devicechange.NewChangeValue(test1Cont1ACont2ALeaf2A, devicechange.NewTypedValueFloat(valueLeaf2B159), false)

	ctrl := gomock.NewController(t)
	// Data for default configuration

	change1 := devicechange.Change{
		Values: []*devicechange.ChangeValue{
			config1Value03},
		DeviceID:      device1,
		DeviceVersion: deviceVersion1,
		DeviceType:    deviceTypeTd,
	}

	now := time.Now()
	networkChange1 := &networkchange.NetworkChange{
		ID:      networkChange1,
		Changes: []*devicechange.Change{&change1},
		Updated: now,
		Created: now,
		Status:  changetypes.Status{State: changetypes.State_PENDING},
	}

	// Mock Device Store
	timeout := time.Millisecond * 10
	device1Topo := &topodevice.Device{
		ID:      device1,
		Address: "1.2.3.4:1234",
		Version: deviceVersion1,
		Timeout: &timeout,
		Type:    deviceTypeTd,
		Protocols: []*topodevice.ProtocolState{
			{
				Protocol:          topodevice.Protocol_GNMI,
				ConnectivityState: topodevice.ConnectivityState_REACHABLE,
				ChannelState:      topodevice.ChannelState_CONNECTED,
				ServiceState:      topodevice.ServiceState_AVAILABLE,
			},
		},
	}
	mockDeviceStore := mockstore.NewMockDeviceStore(ctrl)
	mockDeviceStore.EXPECT().Watch(gomock.Any()).DoAndReturn(
		func(ch chan<- *topodevice.ListResponse) error {
			go func() {
				ch <- &topodevice.ListResponse{
					Type:   topodevice.ListResponse_NONE,
					Device: device1Topo,
				}
			}()
			return nil
		}).Times(3)
	mockDeviceStore.EXPECT().Get(topodevice.ID(device1)).Return(device1Topo, nil).AnyTimes()
	mockDeviceStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(updated *topodevice.Device) (*topodevice.Device, error) {
		return updated, nil
	})

	networkChangesStore, err := networkstore.NewLocalStore()
	assert.NilError(t, err)
	deviceChangesStore, err := devicechanges.NewLocalStore()
	assert.NilError(t, err)
	networkSnapshotStore, err := networksnapstore.NewLocalStore()
	assert.NilError(t, err)
	deviceSnapshotStore, err := devicesnapstore.NewLocalStore()
	assert.NilError(t, err)

	deviceCache, err := devicestore.NewCache(networkChangesStore)
	assert.NilError(t, err)

	leadershipStore, err := leadership.NewLocalStore("test", cluster.NodeID("node1"))
	assert.NilError(t, err)
	mastershipStore, err := mastership.NewLocalStore("test", cluster.NodeID("node1"))
	assert.NilError(t, err)

	mgrTest, err = LoadManager(leadershipStore, mastershipStore, deviceChangesStore, deviceCache,
		networkChangesStore, networkSnapshotStore, deviceSnapshotStore, mockDeviceStore)
	if err != nil {
		t.Fatalf("could not load manager %v", err)
	}

	modelData1 := gnmi.ModelData{
		Name:         "test1",
		Organization: "Open Networking Foundation",
		Version:      "2018-02-20",
	}
	// All values are taken from testdata/sample-testdevice-opstate.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("../modelregistry/jsonvalues/testdata/sample-testdevice-opstate.json")
	assert.NilError(t, err)
	opstateResponseJSON := gnmi.TypedValue_JsonVal{JsonVal: sampleTree}
	changePath1, _ := utils.ParseGNMIElements(utils.SplitPath(test1Cont1ACont2ALeaf2A))

	device1Context := context.Background()

	mockTargetCreationfn := func() southbound.TargetIf {
		mockTarget := southboundmocks.NewMockTargetIf(ctrl)
		southbound.Targets[topodevice.ID(device1)] = mockTarget

		mockTarget.EXPECT().CapabilitiesWithString(
			gomock.Any(),
			"",
		).Return(&gnmi.CapabilityResponse{
			SupportedModels:    []*gnmi.ModelData{&modelData1},
			SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_JSON},
			GNMIVersion:        gnmiVer070,
		}, nil)
		mockTarget.EXPECT().ConnectTarget(gomock.Any(), gomock.Any()).DoAndReturn(
			func(ctx context.Context, device topodevice.Device) (topodevice.ID, error) {
				return device.ID, nil
			})

		mockTarget.EXPECT().Get(
			gomock.Any(),
			// There's only 1 GetRequest in this test, so we're not fussed about contents
			gomock.AssignableToTypeOf(&gnmi.GetRequest{}),
		).Return(&gnmi.GetResponse{
			Notification: []*gnmi.Notification{
				{
					Timestamp: time.Now().Unix(),
					Update: []*gnmi.Update{
						{Path: &gnmi.Path{}, Val: &gnmi.TypedValue{
							Value: &opstateResponseJSON,
						}},
					},
				},
			},
		}, nil).Times(2)

		mockTarget.EXPECT().Subscribe(
			gomock.Any(),
			gomock.AssignableToTypeOf(&gnmi.SubscribeRequest{}),
			gomock.Any(),
		).Return(nil).MinTimes(1)

		mockTarget.EXPECT().Context().Return(&device1Context).AnyTimes()

		mockTarget.EXPECT().Set(
			gomock.Any(),
			gomock.AssignableToTypeOf(&gnmi.SetRequest{}),
		).Return(&gnmi.SetResponse{
			Response: []*gnmi.UpdateResult{
				{Path: changePath1},
			},
			Timestamp: time.Now().Unix(),
		}, nil).AnyTimes()

		return southbound.TargetIf(mockTarget)
	}

	mgrTest.setTargetGenerator(mockTargetCreationfn)
	mgrTest.Run()

	time.Sleep(time.Millisecond * 100)

	assert.Assert(t, networkChange1 != nil)
	err = mgrTest.NetworkChangesStore.Create(networkChange1)
	assert.NilError(t, err, "Unexpected failure when creating new NetworkChange")
	time.Sleep(time.Millisecond * 100)

	return mgrTest, &AllMocks{
		MockStores: &mockstore.MockStores{
			DeviceStore:          mockDeviceStore,
			NetworkChangesStore:  nil,
			DeviceChangesStore:   nil,
			NetworkSnapshotStore: nil,
			DeviceSnapshotStore:  nil,
			LeadershipStore:      nil,
			MastershipStore:      nil,
		},
	}
}

func Test_GetNetworkConfig_Deep(t *testing.T) {

	mgrTest, _ := setUpDeepTest(t)

	result, err := mgrTest.GetTargetConfig(device1, deviceVersion1, "/*", 0)
	assert.NilError(t, err, "GetTargetConfig error")
	time.Sleep(time.Millisecond * 100)

	assert.Equal(t, len(result), 1, "Unexpected result element count")

	assert.Equal(t, result[0].Path, test1Cont1ACont2ALeaf2A, "result %s is different")
}

// Test a change on Device 1 which was already created and mocked in the setUp
func Test_SetNetworkConfig_Deep(t *testing.T) {
	mgrTest, _ := setUpDeepTest(t)

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
	time.Sleep(time.Millisecond * 100) // Have to wait for change to be reconciled

	testUpdate, _ := mgrTest.NetworkChangesStore.Get(testNetworkChange)
	assert.Assert(t, testUpdate != nil)
	assert.Equal(t, testUpdate.ID, testNetworkChange, "Change Ids should correspond")
	assert.Equal(t, changetypes.Phase_CHANGE, testUpdate.Status.Phase)
	assert.Equal(t, changetypes.State_COMPLETE, testUpdate.Status.State)

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

// Test a change on Device 2 - this device does not exist on Topo - a config-only device at this stage
func Test_SetNetworkConfig_ConfigOnly_Deep(t *testing.T) {
	mgrTest, _ := setUpDeepTest(t)

	// Making change
	updates := make(devicechange.TypedValueMap)
	updates[test1Cont1ACont2ALeaf2A] = devicechange.NewTypedValueUint64(valueLeaf2A789)
	updates[test1Cont1ACont2ALeaf2B] = devicechange.NewTypedValueDecimal64(1590, 3)
	deletes := []string{}
	updatesForConfigOnlyDevice, deletesForConfigOnlyDevice, deviceInfo := makeDeviceChanges(deviceConfigOnly, updates, deletes)

	// Set the new change
	const testNetworkChange networkchange.ID = "ConfigOnly_SetNetworkConfig"
	err := mgrTest.SetNetworkConfig(updatesForConfigOnlyDevice, deletesForConfigOnlyDevice, deviceInfo, string(testNetworkChange))
	assert.NilError(t, err, "ConfigOnly_SetNetworkConfig error")
	time.Sleep(time.Millisecond * 10) // Have to wait for change to be reconciled

	testUpdate, _ := mgrTest.NetworkChangesStore.Get(testNetworkChange)
	assert.Assert(t, testUpdate != nil)
	assert.Equal(t, testUpdate.ID, testNetworkChange, "Change Ids should correspond")
	assert.Equal(t, changetypes.Phase_CHANGE, testUpdate.Status.Phase)

	// The following is the problem in Issue https://github.com/onosproject/onos-config/issues/866
	// The Device Change controller never gets created and hence the Network Change controller
	// just sits there doing nothing and stays in RUNNING
	//assert.Equal(t, changetypes.State_COMPLETE, testUpdate.Status.State)
}
