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

package synchronizer

import (
	context2 "context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/assert"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

const (
	cont1aLeaf1a              = "/cont1a/leaf1a"
	cont1aList2aTxout1        = "/cont1a/list2a[name=txout1]"
	cont1aList2aTxout1Txpower = "/cont1a/list2a[name=txout1]/tx-power"
	cont1aList2aTxout3Txpower = "/cont1a/list2a[name=txout3]/tx-power"
	leafAtTopLevel            = "leafAtTopLevel"

	cont1aCont2aLeaf2a = "/cont1a/cont2a/leaf2a"
	cont1aCont2aLeaf2b = "/cont1a/cont2a/leaf2b"
	cont1aCont2aLeaf2c = "/cont1a/cont2a/leaf2c" // State
	cont1aCont2aLeaf2d = "/cont1a/cont2a/leaf2d"
	cont1aCont2aLeaf2e = "/cont1a/cont2a/leaf2e"
	cont1aCont2aLeaf2g = "/cont1a/cont2a/leaf2g"
	leaf2d             = "/leaf2d"
	list2bWcLeaf3c     = "/list2b[index=*]/leaf3c"
	list2b100Leaf3c    = "/list2b[index=100]/leaf3c"
	list2b101Leaf3c    = "/list2b[index=101]/leaf3c"
	cont1bState        = "/cont1b-state"
	gnmiVer070         = "0.7.0"
)

func synchronizerSetUp() (*store.ChangeStore, *store.ConfigurationStore,
	chan devicepb.ListResponse, chan events.OperationalStateEvent,
	chan events.DeviceResponse, *dispatcher.Dispatcher,
	*modelregistry.ModelRegistry, modelregistry.ReadOnlyPathMap,
	change.TypedValueMap, chan events.ConfigEvent,
	error) {

	changeStore, err := store.LoadChangeStore("../../../configs/changeStore-sample.json")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	configStore, err := store.LoadConfigStore("../../../configs/configStore-sample.json")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	dispatcher := dispatcher.NewDispatcher()
	mr := new(modelregistry.ModelRegistry)
	opStateCache := make(change.TypedValueMap)
	// See modelplugin/yang/TestDevice-1.0.0/test1@2018-02-20.yang for paths
	roPathMap := make(modelregistry.ReadOnlyPathMap)
	roSubPath1 := make(modelregistry.ReadOnlySubPathMap)
	roSubPath1["/"] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeSTRING}
	roPathMap[cont1aCont2aLeaf2c] = roSubPath1
	roSubPath2 := make(modelregistry.ReadOnlySubPathMap)
	roSubPath2[leaf2d] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath2[list2bWcLeaf3c] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeSTRING}
	roPathMap[cont1bState] = roSubPath2
	return &changeStore, &configStore,
		make(chan devicepb.ListResponse),
		make(chan events.OperationalStateEvent),
		make(chan events.DeviceResponse),
		dispatcher, mr, roPathMap, opStateCache,
		make(chan events.ConfigEvent),
		nil
}

/**
 * Test the creation of a new synchronizer for a device that does not exist in
 * the config store - later we will test one that is in the config store
 * In this test we also have JSON encoding while later we test PROTO encoding
 * Also in this case we test the GetStateExplicitRoPathsExpandWildcards method of
 * getting the OpState attributes
 */
func TestNew(t *testing.T) {
	changeStore, configStore, topoChan, opstateChan, responseChan, dispatcher,
		models, roPathMap, opstateCache, configChan, err := synchronizerSetUp()
	assert.NilError(t, err, "Error in factorySetUp()")
	assert.Assert(t, changeStore != nil)
	assert.Assert(t, configStore != nil)
	assert.Assert(t, topoChan != nil)
	assert.Assert(t, opstateChan != nil)
	assert.Assert(t, responseChan != nil)
	assert.Assert(t, configChan != nil)
	assert.Assert(t, dispatcher != nil)
	assert.Assert(t, models != nil)
	assert.Assert(t, roPathMap != nil)
	assert.Assert(t, opstateCache != nil)

	_, textValue, _, _ := setUpStatePaths(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockTarget := NewMockTargetIf(ctrl)
	modelData1 := gnmi.ModelData{
		Name:         "test1",
		Organization: "Open Networking Foundation",
		Version:      "2018-02-20",
	}
	timeout := time.Millisecond * 200
	mock1NameStr := "mockTd"
	mockDevice1 := device.Device{
		ID:          device.ID(mock1NameStr),
		Revision:    0,
		Address:     "1.2.3.4:11161",
		Target:      "",
		Version:     "1.0.0",
		Timeout:     &timeout,
		Credentials: device.Credentials{},
		TLS:         device.TlsConfig{},
		Type:        "TestDevice",
		Role:        "leaf",
		Attributes:  nil,
	}

	mockTarget.EXPECT().ConnectTarget(
		gomock.Any(),
		mockDevice1,
	).Return(southbound.DeviceID{DeviceID: mock1NameStr}, nil)

	mockTarget.EXPECT().CapabilitiesWithString(
		gomock.Any(),
		"",
	).Return(&gnmi.CapabilityResponse{
		SupportedModels:    []*gnmi.ModelData{&modelData1},
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_JSON},
		GNMIVersion:        gnmiVer070,
	}, nil)

	go func() {
		// Handles any errors coming back from functions
		for resp := range responseChan {
			assert.NilError(t, resp.Error(), "Expecting no error response")
			// If all is running cleanly we should only get a response after Set
			assert.Assert(t, strings.Contains(resp.Response(), `<path:<elem:<name:"cont1a" > elem:<name:"cont2a" >`))
		}
	}()

	s, err := New(context2.Background(), changeStore, configStore, &mockDevice1, configChan,
		opstateChan, responseChan, opstateCache, roPathMap, mockTarget,
		modelregistry.GetStateExplicitRoPaths)
	assert.NilError(t, err, "Creating s")
	assert.Equal(t, string(s.ID), mock1NameStr)
	assert.Equal(t, string(s.Device.ID), mock1NameStr)
	assert.Equal(t, s.encoding, gnmi.Encoding_JSON) // Should be default

	time.Sleep(100 * time.Millisecond) // Wait for response message

	// called asynchronously as it hangs until it gets a config event
	go s.syncConfigEventsToDevice(mockTarget, responseChan)

	// Listen for OpState updates
	go func() {
		for o := range opstateChan {
			fmt.Println("OpState cache subscribe event received", o.Path(), o.EventType(), o.ItemAction())
			assert.Equal(t, o.Subject(), mock1NameStr)
		}
	}()

	// All values are taken from testdata/sample-testdevice-opstate.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("../../modelregistry/jsonvalues/testdata/sample-testdevice-opstate.json")
	assert.NilError(t, err)
	opstateResponseJSON := gnmi.TypedValue_JsonVal{JsonVal: sampleTree}

	//opStatePathWc, err := utils.ParseGNMIElements(utils.SplitPath(cont1bState + list2bWcLeaf3c))
	assert.NilError(t, err, "Path for wildcard get")

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
	}, nil)

	mockTarget.EXPECT().Subscribe(
		gomock.Any(),
		gomock.AssignableToTypeOf(&gnmi.SubscribeRequest{}),
		gomock.Any(),
	).Return(nil).MinTimes(1)

	// Called asynchronously as after building up the opStateCache it subscribes and waits
	go s.syncOperationalStateByPaths(context2.Background(), mockTarget, responseChan)

	time.Sleep(200 * time.Millisecond) // Wait for response message
	os1, ok := opstateCache[cont1bState+leaf2d]
	assert.Assert(t, ok, "Retrieving 1st path from Op State cache")
	assert.Equal(t, os1.Type, change.ValueTypeUINT)
	assert.Equal(t, os1.String(), "10001")
	os2, ok := opstateCache[cont1aCont2aLeaf2c]
	assert.Assert(t, ok, "Retrieving 2nd path from Op State cache")
	assert.Equal(t, os2.Type, change.ValueTypeSTRING)
	assert.Equal(t, os2.String(), "Mock leaf2c value")
	os3, ok := opstateCache[cont1bState+list2b100Leaf3c]
	assert.Assert(t, ok, "Retrieving 3rd path from Op State cache")
	assert.Equal(t, os3.Type, change.ValueTypeSTRING)
	assert.Equal(t, os3.String(), "mock Value in JSON")
	os4, ok := opstateCache[cont1bState+list2b101Leaf3c]
	assert.Assert(t, ok, "Retrieving 4th path from Op State cache")
	assert.Equal(t, os4.Type, change.ValueTypeSTRING)
	assert.Equal(t, os4.String(), "Second mock Value")

	opStatePath1, err := utils.ParseGNMIElements(utils.SplitPath(cont1bState + list2b100Leaf3c))
	assert.NilError(t, err, "Path for wildcard get")
	opStatePath2, err := utils.ParseGNMIElements(utils.SplitPath(cont1bState + list2b101Leaf3c))
	assert.NilError(t, err, "Path for wildcard get")

	// Send a message to the Subscribe request
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message
	subscribeResp1 := gnmi.SubscribeResponse_Update{
		Update: &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update: []*gnmi.Update{
				{Path: opStatePath1, Val: textValue},
				{Path: opStatePath2, Val: textValue},
			},
		},
	}
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeResp1,
	})
	assert.NilError(t, err, "Subscribe response test 1st time")

	// Send it again message to the Subscribe request
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeResp1,
	})
	assert.NilError(t, err, "Subscribe response test 2nd time")

	// Even try closing the subscription
	subscribeSync := gnmi.SubscribeResponse_SyncResponse{SyncResponse: true}
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeSync,
	})
	assert.NilError(t, err)
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message
}

func synchronizerBootstrap(t *testing.T) (*MockTargetIf, *device.Device, *gnmi.CapabilityResponse) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockTarget := NewMockTargetIf(ctrl)
	modelData1 := gnmi.ModelData{
		Name:         "test1",
		Organization: "Open Networking Foundation",
		Version:      "2018-02-20",
	}
	capabilitiesResp := gnmi.CapabilityResponse{
		SupportedModels:    []*gnmi.ModelData{&modelData1},
		SupportedEncodings: []gnmi.Encoding{}, // Defaults to PROTO
		GNMIVersion:        gnmiVer070,
	}

	timeout := time.Millisecond * 200
	device1NameStr := "Device1" // Exists in configStore-sample.json
	device1 := device.Device{
		ID:          device.ID(device1NameStr),
		Revision:    0,
		Address:     "1.2.3.4:11161",
		Target:      "",
		Version:     "1.0.0",
		Timeout:     &timeout,
		Credentials: device.Credentials{},
		TLS:         device.TlsConfig{},
		Type:        "TestDevice",
		Role:        "leaf",
		Attributes:  nil,
	}

	return mockTarget, &device1, &capabilitiesResp
}

func setUpStatePaths(t *testing.T) (*gnmi.Path, *gnmi.TypedValue, *gnmi.Path, *gnmi.TypedValue) {
	statePath, err := utils.ParseGNMIElements(utils.SplitPath(cont1aCont2aLeaf2c))
	assert.NilError(t, err)
	stateValue, err := values.NativeTypeToGnmiTypedValue(change.NewTypedValueString("mock Value"))
	assert.NilError(t, err)
	opPath, err := utils.ParseGNMIElements(utils.SplitPath(cont1bState + leaf2d))
	assert.NilError(t, err)
	opValue, err := values.NativeTypeToGnmiTypedValue(change.NewTypedValueUint64(10002))
	assert.NilError(t, err)
	return statePath, stateValue, opPath, opValue
}

/**
 * Test the creation of a new synchronizer for a device that does exist in
 * the config store Device1
 * In this test we also have PROTO encoding
 * Also in this case we test the GetState_OpState for getting the OpState attribs
 */
func TestNewWithExistingConfig(t *testing.T) {
	changeStore, configStore, _, opstateChan, responseChan, _,
		_, roPathMap, opstateCache, configChan, err := synchronizerSetUp()
	assert.NilError(t, err, "Error in factorySetUp()")

	mockTarget, device1, capabilitiesResp := synchronizerBootstrap(t)

	statePath1, stateValue1, opPath2, opValue2 := setUpStatePaths(t)

	mockTarget.EXPECT().ConnectTarget(
		gomock.Any(),
		*device1,
	).Return(southbound.DeviceID{DeviceID: string(device1.ID)}, nil)

	mockTarget.EXPECT().CapabilitiesWithString(
		gomock.Any(),
		"",
	).Return(capabilitiesResp, nil)

	changePath1, err := utils.ParseGNMIElements(utils.SplitPath(cont1aCont2aLeaf2a))
	assert.NilError(t, err)
	changePath2, err := utils.ParseGNMIElements(utils.SplitPath(cont1aCont2aLeaf2b))
	assert.NilError(t, err)
	changePath3, err := utils.ParseGNMIElements(utils.SplitPath(cont1aCont2aLeaf2d))
	assert.NilError(t, err)
	changePath4, err := utils.ParseGNMIElements(utils.SplitPath(cont1aCont2aLeaf2e))
	assert.NilError(t, err)
	changePath5, err := utils.ParseGNMIElements(utils.SplitPath(cont1aCont2aLeaf2g))
	assert.NilError(t, err)
	changePath6, err := utils.ParseGNMIElements(utils.SplitPath(cont1aLeaf1a))
	assert.NilError(t, err)
	changePath7, err := utils.ParseGNMIElements(utils.SplitPath(cont1aList2aTxout1))
	assert.NilError(t, err)
	changePath8, err := utils.ParseGNMIElements(utils.SplitPath(cont1aList2aTxout1Txpower))
	assert.NilError(t, err)
	changePath10, err := utils.ParseGNMIElements(utils.SplitPath(cont1aList2aTxout3Txpower))
	assert.NilError(t, err)
	changePath11, err := utils.ParseGNMIElements(utils.SplitPath(leafAtTopLevel))
	assert.NilError(t, err)

	mockTarget.EXPECT().Set(
		gomock.Any(),
		gomock.AssignableToTypeOf(&gnmi.SetRequest{}),
	).Return(&gnmi.SetResponse{
		Response: []*gnmi.UpdateResult{
			{Path: changePath1},
			{Path: changePath2},
			{Path: changePath3},
			{Path: changePath4},
			{Path: changePath5},
			{Path: changePath6},
			{Path: changePath7},
			{Path: changePath8},
			{Path: changePath10},
			{Path: changePath11},
		},
		Timestamp: time.Now().Unix(),
	}, nil).Times(2)

	mockTarget.EXPECT().Get(
		gomock.Any(),
		&gnmi.GetRequest{
			Type:     2, // GetRequest_STATE
			Encoding: gnmi.Encoding_PROTO,
		},
	).Return(&gnmi.GetResponse{
		Notification: []*gnmi.Notification{
			{
				Timestamp: time.Now().Unix(),
				Update: []*gnmi.Update{
					{Path: statePath1, Val: stateValue1},
				},
			},
		},
	}, nil)

	mockTarget.EXPECT().Get(
		gomock.Any(),
		&gnmi.GetRequest{
			Type:     3, // GetRequest_OPERATIONAL
			Encoding: gnmi.Encoding_PROTO,
		},
	).Return(&gnmi.GetResponse{
		Notification: []*gnmi.Notification{
			{
				Timestamp: time.Now().Unix(),
				Update: []*gnmi.Update{
					{Path: opPath2, Val: opValue2},
				},
			},
		},
	}, nil)

	mockTarget.EXPECT().Subscribe(
		gomock.Any(),
		gomock.AssignableToTypeOf(&gnmi.SubscribeRequest{}),
		gomock.Any(),
	).Return(nil).MinTimes(1)

	go func() {
		// Handles any errors coming back from functions
		for resp := range responseChan {
			assert.NilError(t, resp.Error(), "Expecting no error response")
			assert.Assert(t, strings.Contains(resp.Response(), `<path:<elem:<name:"cont1a" > elem:<name:"cont2a" >`))
		}
	}()

	s, err := New(context2.Background(), changeStore, configStore, device1, configChan,
		opstateChan, responseChan, opstateCache, roPathMap, mockTarget, modelregistry.GetStateOpState)
	assert.NilError(t, err, "Creating synchronizer")
	assert.Equal(t, s.ID, device1.ID)
	assert.Equal(t, s.Device.ID, device1.ID)
	assert.Equal(t, s.encoding, gnmi.Encoding_PROTO) // Should be default

	time.Sleep(100 * time.Millisecond) // Wait for response message

	// called asynchronously as it hangs until it gets a config event
	go s.syncConfigEventsToDevice(mockTarget, responseChan)

	//Create a change that we can send down to device
	value1, err := change.NewChangeValue(cont1aCont2aLeaf2a, change.NewTypedValueUint64(12), false)
	assert.NilError(t, err)
	change1, err := change.NewChange([]*change.Value{value1}, "mock test change")
	assert.NilError(t, err)
	s.ChangeStore.Store[store.B64(change1.ID)] = change1

	// Send a device config event
	go func() {
		configEvent := events.NewConfigEvent(string(device1.ID), change1.ID, true)
		configChan <- configEvent
	}()

	time.Sleep(100 * time.Millisecond) // Wait for response message

	// Listen for OpState updates
	go func() {
		for o := range opstateChan {
			fmt.Println("OpState cache subscribe event received", o.Path(), o.EventType(), o.ItemAction())
			assert.Equal(t, o.Subject(), string("Device1"))
		}
	}()

	// Called asynchronously as after building up the opStateCache it subscribes and waits
	go s.syncOperationalStateByPartition(context2.Background(), mockTarget, responseChan)
	subscribeResp1 := gnmi.SubscribeResponse_Update{
		Update: &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update: []*gnmi.Update{
				{
					Path: statePath1,
					Val:  stateValue1,
				},
			},
		},
	}
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeResp1,
	})
	assert.NilError(t, err)
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message
	subscribeResp2 := gnmi.SubscribeResponse_Update{
		Update: &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update: []*gnmi.Update{
				{
					Path: opPath2,
					Val:  opValue2,
				},
			},
		},
	}
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeResp2,
	})
	assert.NilError(t, err)
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message

	// Test out all the switch cases for opStateSubHandler()
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: nil,
	})
	assert.ErrorContains(t, err, "unknown response")

	subscribeError := gnmi.SubscribeResponse_Error{Error: nil}
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeError,
	})
	assert.ErrorContains(t, err, "error in response")

	// Try with a delete
	subscribeResp3 := gnmi.SubscribeResponse_Update{
		Update: &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Delete:    []*gnmi.Path{opPath2},
		},
	}
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeResp3,
	})
	assert.NilError(t, err, "trying op state delete")

	// Even try closing the subscription
	subscribeSync := gnmi.SubscribeResponse_SyncResponse{SyncResponse: true}
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeSync,
	})
	assert.NilError(t, err)
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message
}

func TestNewWithExistingConfigError(t *testing.T) {
	changeStore, configStore, _, opstateChan, responseChan, _,
		_, roPathMap, opstateCache, configChan, err := synchronizerSetUp()
	assert.NilError(t, err, "Error in factorySetUp()")

	mockTarget, device1, capabilitiesResp := synchronizerBootstrap(t)

	mockTarget.EXPECT().ConnectTarget(
		gomock.Any(),
		*device1,
	).Return(southbound.DeviceID{DeviceID: string(device1.ID)}, nil)

	mockTarget.EXPECT().CapabilitiesWithString(
		gomock.Any(),
		"",
	).Return(capabilitiesResp, nil)

	mockTarget.EXPECT().Set(
		gomock.Any(),
		gomock.AssignableToTypeOf(&gnmi.SetRequest{}),
	).Return(nil, status.Errorf(codes.Internal, "test, desc = error generated by mock"))

	go func() {
		for resp := range responseChan {
			assert.ErrorContains(t, resp.Error(), "error generated by mock")
		}
	}()

	s, err := New(context2.Background(), changeStore, configStore, device1, configChan,
		opstateChan, responseChan, opstateCache, roPathMap, mockTarget, modelregistry.GetStateOpState)

	assert.NilError(t, err, "Creating synchronizer")
	assert.Equal(t, s.ID, device1.ID)
	assert.Equal(t, s.Device.ID, device1.ID)
	assert.Equal(t, s.encoding, gnmi.Encoding_PROTO) // Should be default

	time.Sleep(100 * time.Millisecond) // Wait for response message
}

/**
 * Test the creation of a new synchronizer for a stratum type device that does
 * not exist in the config store
 * PROTO encoding with  GetStateExplicitRoPathsExpandWildcards method
 */
func Test_LikeStratum(t *testing.T) {
	const (
		s1Eth1                     = "s1-eth1"
		s1Eth2                     = "s1-eth2"
		interfacesInterfaceWcState = "/interfaces/interface[name=*]/state"
		ifName                     = "/name"
		ifIndex                    = "/ifindex"
		adminStatus                = "/admin-status"
		hardwarePort               = "/hardware-port"
		healthIndicator            = "/health-indicator"
		lastChange                 = "/last-change"
		operStatus                 = "/oper-status"
		countersInBroadcastPkts    = "/counters/in-broadcast-pkts"
		countersInDiscards         = "/counters/in-discards"
		countersInErrors           = "/counters/in-errors"
		countersInFcsErrors        = "/counters/in-fcs-errors"
		countersInMcastPkts        = "/counters/in-multicast-pkts"
		countersInOctets           = "/counters/in-octets"
		countersInUnicastPkts      = "/counters/in-unicast-pkts"
		countersInUnknPkts         = "/counters/in-unknown-protos"
		countersInBcastPkts        = "/counters/out-broadcast-pkts"
		countersOutDiscards        = "/counters/out-discards"
		countersOutErrs            = "/counters/out-errors"
		countersOutMcastPkts       = "/counters/out-multicast-pkts"
		countersOutOctets          = "/counters/out-octets"
		countersOutUcastPkts       = "/counters/out-unicast-pkts"

		interfacesInterfaceEth1State        = "/interfaces/interface[name=s1-eth1]/state"
		interfacesInterfaceEth1StateIfindex = interfacesInterfaceEth1State + ifIndex
		interfacesInterfaceEth2State        = "/interfaces/interface[name=s1-eth2]/state"
		interfacesInterfaceEth2StateIfindex = interfacesInterfaceEth2State + ifIndex
	)

	changeStore := store.ChangeStore{
		Version:   "1.0.0",
		Storetype: "changes",
		Store:     make(map[string]*change.Change),
	}

	configStore := store.ConfigurationStore{
		Version:   "1.0.0",
		Storetype: "configs",
		Store:     make(map[store.ConfigName]store.Configuration),
	}

	opStateCache := make(change.TypedValueMap)
	// See modelplugin/yang/TestDevice-1.0.0/test1@2018-02-20.yang for paths
	roPathMap := make(modelregistry.ReadOnlyPathMap)
	roSubPath1 := make(modelregistry.ReadOnlySubPathMap)
	roSubPath1[ifIndex] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[ifName] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeSTRING}
	roSubPath1[adminStatus] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeSTRING}
	roSubPath1[hardwarePort] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeSTRING}
	roSubPath1[healthIndicator] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeSTRING}
	roSubPath1[lastChange] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeSTRING}
	roSubPath1[operStatus] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeSTRING}
	roSubPath1[countersInBroadcastPkts] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersInDiscards] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersInErrors] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersInFcsErrors] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersInMcastPkts] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersInOctets] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersInUnicastPkts] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersInUnknPkts] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersInBcastPkts] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersOutDiscards] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersOutErrs] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersOutMcastPkts] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersOutOctets] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roSubPath1[countersOutUcastPkts] = modelregistry.ReadOnlyAttrib{Datatype: change.ValueTypeUINT}
	roPathMap[interfacesInterfaceWcState] = roSubPath1

	opstateChan := make(chan events.OperationalStateEvent)
	responseChan := make(chan events.DeviceResponse)
	configChan := make(chan events.ConfigEvent)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockTarget := NewMockTargetIf(ctrl)
	modelData1 := gnmi.ModelData{
		Name:         "openconfig-interfaces",
		Organization: "OpenConfig working group",
		Version:      "2.4.1",
	} // And many many more
	timeout := time.Millisecond * 200
	mock1NameStr := "stratum-1"
	mockDevice1 := device.Device{
		ID:          device.ID(mock1NameStr),
		Revision:    0,
		Address:     "1.2.3.4:50001",
		Target:      "",
		Version:     "1.0.0",
		Timeout:     &timeout,
		Credentials: device.Credentials{},
		TLS:         device.TlsConfig{},
		Type:        "Stratum",
		Role:        "leaf",
		Attributes:  nil,
	}

	mockTarget.EXPECT().ConnectTarget(
		gomock.Any(),
		mockDevice1,
	).Return(southbound.DeviceID{DeviceID: mock1NameStr}, nil)

	mockTarget.EXPECT().CapabilitiesWithString(
		gomock.Any(),
		"",
	).Return(&gnmi.CapabilityResponse{
		SupportedModels:    []*gnmi.ModelData{&modelData1},
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_PROTO},
		GNMIVersion:        "0.7.0",
	}, nil)

	go func() {
		// Handles any errors coming back from functions
		for resp := range responseChan {
			assert.NilError(t, resp.Error(), "Expecting no error response")
		}
	}()

	s, err := New(context2.Background(), &changeStore, &configStore, &mockDevice1, configChan,
		opstateChan, responseChan, opStateCache, roPathMap, mockTarget,
		modelregistry.GetStateExplicitRoPathsExpandWildcards)
	assert.NilError(t, err, "Creating s")
	assert.Equal(t, string(s.ID), mock1NameStr)
	assert.Equal(t, string(s.Device.ID), mock1NameStr)
	assert.Equal(t, s.encoding, gnmi.Encoding_PROTO) // Should be default

	time.Sleep(100 * time.Millisecond) // Wait for response message

	// called asynchronously as it hangs until it gets a config event
	go s.syncConfigEventsToDevice(mockTarget, responseChan)

	// Listen for OpState updates
	go func() {
		for o := range opstateChan {
			fmt.Println("OpState cache subscribe event received", o.Path(), o.EventType(), o.ItemAction())
			assert.Equal(t, o.Subject(), mock1NameStr)
		}
	}()

	wcPath, err := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceWcState))
	wcResult1, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth1StateIfindex))
	wcResultValue1 := gnmi.TypedValue_UintVal{UintVal: 1}
	wcResult2, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth2StateIfindex))
	wcResultValue2 := gnmi.TypedValue_UintVal{UintVal: 2}

	assert.NilError(t, err, "Path for wildcard get")
	mockTarget.EXPECT().Get(
		gomock.Any(),
		&gnmi.GetRequest{
			Path:     []*gnmi.Path{wcPath},
			Encoding: gnmi.Encoding_PROTO,
		},
	).Return(&gnmi.GetResponse{
		Notification: []*gnmi.Notification{
			{
				Timestamp: time.Now().Unix(),
				Update: []*gnmi.Update{
					{Path: wcResult1, Val: &gnmi.TypedValue{
						Value: &wcResultValue1,
					}},
					{Path: wcResult2, Val: &gnmi.TypedValue{
						Value: &wcResultValue2,
					}},
				},
			},
		},
	}, nil)

	ewPath1, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth1State))
	ewPath2, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth2State))
	if1Name, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth1State + ifName))
	if1NameValue := gnmi.TypedValue_StringVal{StringVal: s1Eth1}
	if2Name, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth2State + ifName))
	if2NameValue := gnmi.TypedValue_StringVal{StringVal: s1Eth2}
	if1AdminStatus, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth1State + adminStatus))
	if2AdminStatus, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth2State + adminStatus))
	adminStatusUp := gnmi.TypedValue_StringVal{StringVal: "UP"}
	if1Inoctets, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth1State + countersInOctets))
	value11111 := gnmi.TypedValue_UintVal{UintVal: 11111}
	if2Inoctets, _ := utils.ParseGNMIElements(utils.SplitPath(interfacesInterfaceEth2State + countersInOctets))
	value22222 := gnmi.TypedValue_UintVal{UintVal: 22222}

	mockTarget.EXPECT().Get(
		gomock.Any(),
		&gnmi.GetRequest{
			Path:     []*gnmi.Path{ewPath1, ewPath2},
			Encoding: gnmi.Encoding_PROTO,
		},
	).Return(&gnmi.GetResponse{
		Notification: []*gnmi.Notification{
			{
				Timestamp: time.Now().Unix(),
				Update: []*gnmi.Update{
					{Path: wcResult1, Val: &gnmi.TypedValue{
						Value: &wcResultValue1,
					}},
					{Path: wcResult2, Val: &gnmi.TypedValue{
						Value: &wcResultValue2,
					}},
					{Path: if1Name, Val: &gnmi.TypedValue{
						Value: &if1NameValue,
					}},
					{Path: if2Name, Val: &gnmi.TypedValue{
						Value: &if2NameValue,
					}},
					{Path: if1AdminStatus, Val: &gnmi.TypedValue{
						Value: &adminStatusUp,
					}},
					{Path: if2AdminStatus, Val: &gnmi.TypedValue{
						Value: &adminStatusUp,
					}},
					{Path: if1Inoctets, Val: &gnmi.TypedValue{
						Value: &value11111,
					}},
					{Path: if2Inoctets, Val: &gnmi.TypedValue{
						Value: &value22222,
					}},
				},
			},
		},
	}, nil)

	mockTarget.EXPECT().Subscribe(
		gomock.Any(),
		gomock.AssignableToTypeOf(&gnmi.SubscribeRequest{}),
		gomock.Any(),
	).Return(nil).MinTimes(1)

	// Called asynchronously as after building up the opStateCache it subscribes and waits
	go s.syncOperationalStateByPaths(context2.Background(), mockTarget, responseChan)

	time.Sleep(200 * time.Millisecond) // Wait for response message
	os1, ok := opStateCache[interfacesInterfaceEth1StateIfindex]
	assert.Assert(t, ok, "Retrieving 1st path from Op State cache")
	assert.Equal(t, os1.Type, change.ValueTypeUINT)
	assert.Equal(t, os1.String(), "1")
	os2, ok := opStateCache[interfacesInterfaceEth2StateIfindex]
	assert.Assert(t, ok, "Retrieving 2nd path from Op State cache")
	assert.Equal(t, os2.Type, change.ValueTypeUINT)
	assert.Equal(t, os2.String(), "2")
	os3, ok := opStateCache[interfacesInterfaceEth1State+ifName]
	assert.Assert(t, ok, "Retrieving 3rd path from Op State cache")
	assert.Equal(t, os3.Type, change.ValueTypeSTRING)
	assert.Equal(t, os3.String(), s1Eth1)
	os4, ok := opStateCache[interfacesInterfaceEth2State+ifName]
	assert.Assert(t, ok, "Retrieving 4th path from Op State cache")
	assert.Equal(t, os4.Type, change.ValueTypeSTRING)
	assert.Equal(t, os4.String(), s1Eth2)
	os5, ok := opStateCache[interfacesInterfaceEth1State+adminStatus]
	assert.Assert(t, ok, "Retrieving 5th path from Op State cache")
	assert.Equal(t, os5.Type, change.ValueTypeSTRING)
	assert.Equal(t, os5.String(), "UP")
	os6, ok := opStateCache[interfacesInterfaceEth2State+adminStatus]
	assert.Assert(t, ok, "Retrieving 6th path from Op State cache")
	assert.Equal(t, os6.Type, change.ValueTypeSTRING)
	assert.Equal(t, os6.String(), "UP")
	os7, ok := opStateCache[interfacesInterfaceEth1State+countersInOctets]
	assert.Assert(t, ok, "Retrieving 7th path from Op State cache")
	assert.Equal(t, os7.Type, change.ValueTypeUINT)
	assert.Equal(t, os7.String(), "11111")
	os8, ok := opStateCache[interfacesInterfaceEth2State+countersInOctets]
	assert.Assert(t, ok, "Retrieving 8th path from Op State cache")
	assert.Equal(t, os8.Type, change.ValueTypeUINT)
	assert.Equal(t, os8.String(), "22222")

	// Send a message to the Subscribe request
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message
	subscribeResp1 := gnmi.SubscribeResponse_Update{
		Update: &gnmi.Notification{
			Timestamp: time.Now().Unix(),
			Update: []*gnmi.Update{
				{Path: if1Inoctets, Val: &gnmi.TypedValue{
					Value: &value22222,
				}},
			},
		},
	}
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeResp1,
	})
	assert.NilError(t, err, "Subscribe response test 1st time")

	// Even try closing the subscription
	subscribeSync := gnmi.SubscribeResponse_SyncResponse{SyncResponse: true}
	err = s.opStateSubHandler(&gnmi.SubscribeResponse{
		Response: &subscribeSync,
	})
	assert.NilError(t, err)
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message
}

func Test_pathMatchesWildcardExactMatch(t *testing.T) {
	wildcards := make(map[string]interface{})
	wildcards["/aa/bb[idx=*]/cc"] = nil
	wildcards["/aa/bb[idx=*]/dd"] = nil

	const testpath1 = "/aa/bb[idx=11]/cc"
	result, err := pathMatchesWildcard(wildcards, testpath1)
	assert.NilError(t, err)
	assert.Equal(t, result, testpath1)

	const testpath2 = "/aa/bb[idx=11]/dd"
	result, err = pathMatchesWildcard(wildcards, testpath2)
	assert.NilError(t, err)
	assert.Equal(t, result, testpath2)
}

func Test_pathMatchesWildcardLonger(t *testing.T) {
	wildcards := make(map[string]interface{})
	wildcards["/aa/bb[idx=*]/cc"] = nil
	wildcards["/aa/bb[idx=*]/dd"] = nil

	const testpath = "/aa/bb[idx=11]/dd/ee"
	const resultpath = "/aa/bb[idx=11]/dd"
	result, err := pathMatchesWildcard(wildcards, testpath)
	assert.NilError(t, err)
	assert.Equal(t, result, resultpath)
}

func Test_pathMatchesWildcardEmptyWc(t *testing.T) {
	wildcards := make(map[string]interface{})

	const testpath = "/aa/bb[idx=11]/dd/ee"
	_, err := pathMatchesWildcard(wildcards, testpath)
	assert.ErrorContains(t, err, "empty")
}

func Test_pathMatchesWildcardEmptyPath(t *testing.T) {
	wildcards := make(map[string]interface{})
	wildcards["/aa/bb[idx=*]/cc"] = nil
	wildcards["/aa/bb[idx=*]/dd"] = nil

	_, err := pathMatchesWildcard(wildcards, "")
	assert.ErrorContains(t, err, "empty")
}

func Test_pathMatchesWildcardNoMatch(t *testing.T) {
	wildcards := make(map[string]interface{})
	wildcards["/aa/bb[idx=*]/cc"] = nil
	wildcards["/aa/bb[idx=*]/dd"] = nil

	const testpath = "/aa/bb[idx=11]"
	_, err := pathMatchesWildcard(wildcards, testpath)
	assert.ErrorContains(t, err, "no match")
}
