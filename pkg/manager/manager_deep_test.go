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
	"fmt"
	"github.com/atomix/atomix-go-client/pkg/atomix"
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	nbutils "github.com/onosproject/onos-config/pkg/northbound/utils"
	testify "github.com/stretchr/testify/assert"
	"io/ioutil"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	changetypes "github.com/onosproject/onos-api/go/onos/config/change"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/southbound"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/device/state"
	networkstore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
	southboundmocks "github.com/onosproject/onos-config/pkg/test/mocks/southbound"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
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
func setUpDeepTest(t *testing.T, client atomix.Client) (*Manager, *AllMocks) {
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
		Protocols: []*topo.ProtocolState{
			{
				Protocol:          topo.Protocol_GNMI,
				ConnectivityState: topo.ConnectivityState_REACHABLE,
				ChannelState:      topo.ChannelState_CONNECTED,
				ServiceState:      topo.ServiceState_AVAILABLE,
			},
		},
	}
	mockDeviceStore := mockstore.NewMockDeviceStore(ctrl)
	mockDeviceStore.EXPECT().Watch(gomock.Any()).DoAndReturn(
		func(ch chan<- *topodevice.ListResponse) error {
			go func() {
				ch <- &topodevice.ListResponse{
					Type:   topodevice.ListResponseNONE,
					Device: device1Topo,
				}
			}()
			return nil
		}).Times(3)
	mockDeviceStore.EXPECT().Get(topodevice.ID(device1)).Return(device1Topo, nil).AnyTimes()
	mockDeviceStore.EXPECT().Get(topodevice.ID(deviceConfigOnly)).Return(nil, fmt.Errorf("device not found")).AnyTimes()
	mockDeviceStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(updated *topodevice.Device) (*topodevice.Device, error) {
		return updated, nil
	})

	networkChangesStore, err := networkstore.NewAtomixStore(client)
	assert.NilError(t, err)
	deviceChangesStore, err := devicechanges.NewAtomixStore(client)
	assert.NilError(t, err)
	networkSnapshotStore, err := networksnapstore.NewAtomixStore(client)
	assert.NilError(t, err)
	deviceSnapshotStore, err := devicesnapstore.NewAtomixStore(client)
	assert.NilError(t, err)
	deviceStateStore, err := state.NewStore(networkChangesStore, deviceSnapshotStore)
	assert.NilError(t, err)

	deviceCache, err := cache.NewCache(networkChangesStore, deviceSnapshotStore)
	assert.NilError(t, err)

	leadershipStore, err := leadership.NewAtomixStore(client)
	assert.NilError(t, err)
	mastershipStore, err := mastership.NewAtomixStore(client, "test")
	assert.NilError(t, err)

	modelRegistry, err := modelregistry.NewModelRegistry(modelregistry.Config{
		ModPath:      "test/data/" + t.Name() + "/mod",
		RegistryPath: "test/data/" + t.Name() + "/registry",
		PluginPath:   "test/data/" + t.Name() + "/plugins",
		ModTarget:    "github.com/onosproject/onos-config",
	})
	assert.NilError(t, err)

	cfg := Config{
		GRPCPort:               5150,
		AllowUnvalidatedConfig: true,
	}

	mgrTest = NewManager(cfg)

	mgrTest.LeadershipStore = leadershipStore
	mgrTest.MastershipStore = mastershipStore
	mgrTest.DeviceChangesStore = deviceChangesStore
	mgrTest.DeviceStateStore = deviceStateStore
	mgrTest.DeviceStore = mockDeviceStore
	mgrTest.DeviceCache = deviceCache
	mgrTest.NetworkChangesStore = networkChangesStore
	mgrTest.NetworkSnapshotStore = networkSnapshotStore
	mgrTest.DeviceSnapshotStore = deviceSnapshotStore
	mgrTest.ModelRegistry = modelRegistry
	mgrTest.OperationalStateCacheLock = &sync.RWMutex{}

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
		southbound.NewTargetItem(devicetype.NewVersionedID(device1, deviceVersion1), mockTarget)

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

	time.Sleep(10 * time.Millisecond)
	assert.Assert(t, networkChange1 != nil)
	err = mgrTest.NetworkChangesStore.Create(networkChange1)
	assert.NilError(t, err, "Unexpected failure when creating new NetworkChange")
	nwChangeUpdates := make(chan stream.Event)
	ctx, err := mgrTest.NetworkChangesStore.Watch(nwChangeUpdates, networkstore.WithChangeID(networkChange1.ID))
	assert.NilError(t, err)
	defer ctx.Close()

	breakout := false
	for { // 3 responses are expected PENDING, RUNNING and COMPLETE
		select {
		case eventObj := <-nwChangeUpdates: //Blocks until event from NW change
			event := eventObj.Object.(*networkchange.NetworkChange)
			t.Logf("Event received %v", event)
			if event.Status.State == changetypes.State_COMPLETE {
				breakout = true
			}
		case <-time.After(10 * time.Second):
			t.FailNow()
		}
		if breakout {
			break
		}
	}

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
	t.Skip() // Were causing intermittent errors on travis TODO Find root cause

	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	testify.NoError(t, test.Start())
	defer test.Stop()

	atomixClient, err := test.NewClient("test")
	testify.NoError(t, err)

	mgrTest, _ := setUpDeepTest(t, atomixClient)

	result, err := nbutils.GetTargetConfig(device1, deviceVersion1, deviceTypeTd, "/*", 0, nil, mgrTest.DeviceStateStore, mgr.ModelRegistry, mgr.Config.AllowUnvalidatedConfig)
	assert.NilError(t, err, "GetTargetConfig error")

	assert.Equal(t, len(result), 1, "Unexpected result element count")

	assert.Equal(t, result[0].Path, test1Cont1ACont2ALeaf2A, "result %s is different")
}

// Test a change on Device 1 which was already created and mocked in the setUp
func Test_SetNetworkConfig_Deep(t *testing.T) {
	t.Skip() // Were causing intermittent errors on travis TODO Find root cause

	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	testify.NoError(t, test.Start())
	defer test.Stop()

	atomixClient, err := test.NewClient("test")
	testify.NoError(t, err)

	mgrTest, _ := setUpDeepTest(t, atomixClient)

	// First verify the value beforehand
	originalChange, _ := mgrTest.NetworkChangesStore.Get(networkChange1)
	assert.Equal(t, len(originalChange.Changes[0].Values), 1)
	assert.Equal(t, originalChange.Changes[0].Values[0].Path, test1Cont1ACont2ALeaf2A)
	assert.Equal(t, originalChange.Changes[0].Values[0].Value.Type, devicechange.ValueType_FLOAT)
	assert.Equal(t, (*devicechange.TypedFloat)(originalChange.Changes[0].Values[0].Value).Float32(), float32(valueLeaf2B159))

	// Making change
	updates := make(devicechange.TypedValueMap)
	updates[test1Cont1ACont2ALeaf2A] = devicechange.NewTypedValueUint(valueLeaf2A789, 16)
	deletes := []string{test1Cont1ACont2ALeaf2C}
	updatesForDevice1, deletesForDevice1, deviceInfo := makeDeviceChanges(device1, updates, deletes)

	// Verify the change
	validationError := nbutils.ValidateNetworkConfig(device1, deviceVersion1, deviceTypeTd, updates, deletes, 0, mgrTest.ModelRegistry, mgrTest.DeviceStateStore, mgrTest.Config.AllowUnvalidatedConfig)
	assert.NilError(t, validationError, "ValidateTargetConfig error")

	// Set the new change
	const testNetworkChange networkchange.ID = "Test_SetNetworkConfig"
	_, err = nbutils.SetNetworkConfig(updatesForDevice1, deletesForDevice1, deviceInfo,
		string(testNetworkChange), "testuser", mgrTest.NetworkChangesStore)
	assert.NilError(t, err, "SetTargetConfig error")

	nwChangeUpdates := make(chan stream.Event)
	ctx, err := mgrTest.NetworkChangesStore.Watch(nwChangeUpdates, networkstore.WithChangeID(testNetworkChange))
	assert.NilError(t, err)
	defer ctx.Close()

	breakout := false
	for { // 3 responses are expected PENDING, RUNNING and COMPLETE
		select {
		case eventObj := <-nwChangeUpdates: //Blocks until event from NW change
			event := eventObj.Object.(*networkchange.NetworkChange)
			t.Logf("Event received %v", event)
			if event.Status.State == changetypes.State_COMPLETE {
				breakout = true
			}
		case <-time.After(10 * time.Second):
			t.FailNow()
		}
		if breakout {
			break
		}
	}

	testUpdate, _ := mgrTest.NetworkChangesStore.Get(testNetworkChange)
	assert.Assert(t, testUpdate != nil)
	assert.Equal(t, testUpdate.ID, testNetworkChange, "Change Ids should correspond")
	assert.Equal(t, changetypes.Phase_CHANGE, testUpdate.Status.Phase)
	assert.Equal(t, changetypes.State_COMPLETE, testUpdate.Status.State)

	// Check that the created change is correct
	updatedVals := testUpdate.Changes[0].Values
	assert.Equal(t, len(updatedVals), 2)

	for _, updatedVal := range updatedVals {
		switch updatedVal.Path {
		case test1Cont1ACont2ALeaf2A:
			assert.Equal(t, (*devicechange.TypedUint)(updatedVal.GetValue()).Uint(), valueLeaf2A789)
			assert.Equal(t, updatedVal.Removed, false)
		case test1Cont1ACont2ALeaf2C:
			assert.Equal(t, updatedVal.GetValue().ValueToString(), "")
			assert.Equal(t, updatedVal.Removed, true)
		default:
			t.Errorf("Unexpected path: %s", updatedVal.Path)
		}
	}
}

// Test a change on device-config-only - this device does not exist on Topo - a config-only device at this stage
func Test_SetNetworkConfig_ConfigOnly_Deep(t *testing.T) {
	t.Skip() // Were causing intermittent errors on travis TODO Find root cause

	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	testify.NoError(t, test.Start())
	defer test.Stop()

	atomixClient, err := test.NewClient("test")
	testify.NoError(t, err)

	mgrTest, allMocks := setUpDeepTest(t, atomixClient)

	allMocks.MockStores.DeviceStore.EXPECT().Get(topodevice.ID("device-config-only")).Return(nil, nil).AnyTimes()

	// Making change
	updates := make(devicechange.TypedValueMap)
	updates[test1Cont1ACont2ALeaf2A] = devicechange.NewTypedValueUint(valueLeaf2A789, 16)
	updates[test1Cont1ACont2ALeaf2B] = devicechange.NewTypedValueDecimal(1590, 3)
	deletes := []string{}
	updatesForConfigOnlyDevice, deletesForConfigOnlyDevice, deviceInfo := makeDeviceChanges(deviceConfigOnly, updates, deletes)

	// Set the new change
	const testNetworkChange networkchange.ID = "ConfigOnly_SetNetworkConfig"
	_, err = nbutils.SetNetworkConfig(updatesForConfigOnlyDevice, deletesForConfigOnlyDevice, deviceInfo,
		string(testNetworkChange), "testuser", mgrTest.NetworkChangesStore)
	assert.NilError(t, err, "ConfigOnly_SetNetworkConfig error")

	nwChangeUpdates := make(chan stream.Event)
	ctx, err := mgrTest.NetworkChangesStore.Watch(nwChangeUpdates, networkstore.WithChangeID(testNetworkChange))
	assert.NilError(t, err)
	defer ctx.Close()

	breakout := false
	for { // 3 responses are expected PENDING, RUNNING and COMPLETE
		select {
		case eventObj := <-nwChangeUpdates: //Blocks until event from NW change
			event := eventObj.Object.(*networkchange.NetworkChange)
			t.Logf("Event received %v", event)
			if event.Status.State == changetypes.State_COMPLETE {
				breakout = true
			}
		case <-time.After(10 * time.Second):
			t.FailNow()
		}
		if breakout {
			break
		}
	}

	testUpdate, _ := mgrTest.NetworkChangesStore.Get(testNetworkChange)
	assert.Assert(t, testUpdate != nil)
	assert.Equal(t, testUpdate.ID, testNetworkChange, "Change Ids should correspond")
	assert.Equal(t, changetypes.Phase_CHANGE, testUpdate.Status.Phase)
	assert.Equal(t, changetypes.State_COMPLETE, testUpdate.Status.State)

	// Check that the created change is correct
	updatedVals := testUpdate.Changes[0].Values
	assert.Equal(t, len(updatedVals), 2)

	for _, updatedVal := range updatedVals {
		switch updatedVal.Path {
		case test1Cont1ACont2ALeaf2A:
			assert.Equal(t, (*devicechange.TypedUint)(updatedVal.GetValue()).Uint(), valueLeaf2A789)
			assert.Equal(t, updatedVal.Removed, false)
		case test1Cont1ACont2ALeaf2B:
			assert.Equal(t, updatedVal.GetValue().ValueToString(), "1.590")
			assert.Equal(t, updatedVal.Removed, false)
		default:
			t.Errorf("Unexpected path: %s", updatedVal.Path)
		}
	}
}

// Test a change on device-disconn - this device does not exist on Topo - a config-only device at this stage
func Test_SetNetworkConfig_Disconnected_Device(t *testing.T) {
	t.Skip() // Were causing intermittent errors on travis TODO Find root cause

	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	testify.NoError(t, test.Start())
	defer test.Stop()

	atomixClient, err := test.NewClient("test")
	testify.NoError(t, err)

	mgrTest, allMocks := setUpDeepTest(t, atomixClient)
	const deviceDisconn = "device-disconn"

	// Mock Device Store
	timeout := time.Millisecond * 10
	deviceDisconnTopo := &topodevice.Device{
		ID:      deviceDisconn,
		Address: "1.2.3.4:1234",
		Version: deviceVersion1,
		Timeout: &timeout,
		Type:    deviceTypeTd,
		Protocols: []*topo.ProtocolState{
			{
				Protocol:          topo.Protocol_GNMI,
				ConnectivityState: topo.ConnectivityState_UNREACHABLE,
				ChannelState:      topo.ChannelState_DISCONNECTED,
				ServiceState:      topo.ServiceState_UNAVAILABLE,
			},
		},
	}
	allMocks.MockStores.DeviceStore.EXPECT().Watch(gomock.Any()).DoAndReturn(
		func(ch chan<- *topodevice.ListResponse) error {
			go func() {
				ch <- &topodevice.ListResponse{
					Type:   topodevice.ListResponseNONE,
					Device: deviceDisconnTopo,
				}
			}()
			return nil
		}).Times(3)
	allMocks.MockStores.DeviceStore.EXPECT().Get(topodevice.ID(deviceDisconn)).Return(deviceDisconnTopo, nil).AnyTimes()
	allMocks.MockStores.DeviceStore.EXPECT().Update(gomock.Any()).DoAndReturn(func(updated *topodevice.Device) (*topodevice.Device, error) {
		return updated, nil
	})

	// Making change
	updates := make(devicechange.TypedValueMap)
	updates[test1Cont1ACont2ALeaf2A] = devicechange.NewTypedValueUint(valueLeaf2A789, 16)
	updates[test1Cont1ACont2ALeaf2B] = devicechange.NewTypedValueDecimal(1590, 3)
	deletes := []string{}
	updatesForDisconnectedDevice, deletesForDisconnectedDevice, deviceInfo := makeDeviceChanges(deviceDisconn, updates, deletes)
	assert.Assert(t, updatesForDisconnectedDevice != nil)
	assert.Assert(t, deletesForDisconnectedDevice != nil)
	assert.Assert(t, deviceInfo != nil)

	// Set the new change
	const testNetworkChange networkchange.ID = "Disconnected_SetNetworkConfig"
	_, err = nbutils.SetNetworkConfig(updatesForDisconnectedDevice, deletesForDisconnectedDevice, deviceInfo,
		string(testNetworkChange), "testuser", mgrTest.NetworkChangesStore)
	assert.NilError(t, err, "Disconnected_SetNetworkConfig error")

	nwChangeUpdates := make(chan stream.Event)
	ctx, err := mgrTest.NetworkChangesStore.Watch(nwChangeUpdates, networkstore.WithChangeID(testNetworkChange))
	assert.NilError(t, err)
	defer ctx.Close()

	breakout := false
	for { // 3 responses are expected PENDING, RUNNING and PENDING (because of failure)
		select {
		case eventObj := <-nwChangeUpdates: //Blocks until event from NW change
			event := eventObj.Object.(*networkchange.NetworkChange)
			t.Logf("Event received %v", event)
			switch event.Status.State {
			case changetypes.State_PENDING:
				breakout = true
			}
		case <-time.After(10 * time.Second):
			breakout = true
			t.FailNow()
		}
		if breakout {
			break
		}
	}
}
