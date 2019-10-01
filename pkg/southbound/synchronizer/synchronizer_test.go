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
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/assert"
	"strings"
	"testing"
	"time"
)

func synchronizerSetUp() (*store.ChangeStore, *store.ConfigurationStore,
	chan events.TopoEvent, chan events.OperationalStateEvent,
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
	roPathMap := make(modelregistry.ReadOnlyPathMap)
	roSubPath1 := make(modelregistry.ReadOnlySubPathMap)
	roSubPath1["/"] = change.ValueTypeSTRING
	roPathMap["/cont1a/cont1b/leaf2c"] = roSubPath1
	roSubPath2 := make(modelregistry.ReadOnlySubPathMap)
	roSubPath2["/leaf2d"] = change.ValueTypeUINT
	roPathMap["/cont1b-state"] = roSubPath2
	return &changeStore, &configStore,
		make(chan events.TopoEvent),
		make(chan events.OperationalStateEvent),
		make(chan events.DeviceResponse),
		dispatcher, mr, roPathMap, opStateCache,
		make(chan events.ConfigEvent),
		nil
}

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

	statePath1, _, opPath2, _ := setUpStatePaths(t)

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
		GNMIVersion:        "0.7.0",
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
		opstateChan, responseChan, opstateCache, roPathMap, mockTarget)
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
			assert.Equal(t, o.Subject(), string("Device1"))
		}
	}()

	stateResponseJSON := gnmi.TypedValue_JsonVal{JsonVal: []byte(`{"cont1a":{"cont2a":{"leaf2c":"mock Value in JSON"}}}`)}
	mockTarget.EXPECT().Get(
		gomock.Any(),
		&gnmi.GetRequest{
			Type:     2, // GetRequest_STATE
			Encoding: gnmi.Encoding_JSON,
		},
	).Return(&gnmi.GetResponse{
		Notification: []*gnmi.Notification{
			{
				Timestamp: time.Now().Unix(),
				Update: []*gnmi.Update{
					{Path: statePath1, Val: &gnmi.TypedValue{
						Value: &stateResponseJSON,
					}},
				},
			},
		},
	}, nil)

	opResponseJSON := gnmi.TypedValue_JsonVal{JsonVal: []byte(`{"cont1b-state":{"leaf2d":10002}}`)}
	mockTarget.EXPECT().Get(
		gomock.Any(),
		&gnmi.GetRequest{
			Type:     3, // GetRequest_OPERATIONAL
			Encoding: gnmi.Encoding_JSON,
		},
	).Return(&gnmi.GetResponse{
		Notification: []*gnmi.Notification{
			{
				Timestamp: time.Now().Unix(),
				Update: []*gnmi.Update{
					{Path: opPath2, Val: &gnmi.TypedValue{
						Value: &opResponseJSON,
					}},
				},
			},
		},
	}, nil)

	mockTarget.EXPECT().Subscribe(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return(nil).MinTimes(1)

	// Called asynchronously as after building up the opStateCache it subscribes and waits
	go s.syncOperationalState(context2.Background(), mockTarget, responseChan)

	time.Sleep(100 * time.Millisecond) // Wait for response message
	os1, ok := opstateCache["/cont1b-state/leaf2d"]
	assert.Assert(t, ok, "Retrieving first path from Op State cache")
	assert.Equal(t, os1.Type, change.ValueTypeUINT)
	assert.Equal(t, os1.String(), "10002")
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
		GNMIVersion:        "0.7.0",
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
	statePath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2c"})
	assert.NilError(t, err)
	stateValue, err := values.NativeTypeToGnmiTypedValue(change.NewTypedValueString("mock Value"))
	assert.NilError(t, err)
	opPath, err := utils.ParseGNMIElements([]string{"cont1b-state", "leaf2d"})
	assert.NilError(t, err)
	opValue, err := values.NativeTypeToGnmiTypedValue(change.NewTypedValueUint64(10001))
	assert.NilError(t, err)
	return statePath, stateValue, opPath, opValue
}

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

	changePath1, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	assert.NilError(t, err)
	changePath2, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	assert.NilError(t, err)
	changePath3, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2d"})
	assert.NilError(t, err)
	changePath4, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2e"})
	assert.NilError(t, err)
	changePath5, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2g"})
	assert.NilError(t, err)
	changePath6, err := utils.ParseGNMIElements([]string{"cont1a", "leaf1a"})
	assert.NilError(t, err)
	changePath7, err := utils.ParseGNMIElements([]string{"cont1a", "list2a[name=txout1]"})
	assert.NilError(t, err)
	changePath8, err := utils.ParseGNMIElements([]string{"cont1a", "list2a[name=txout1]", "tx-power"})
	assert.NilError(t, err)
	changePath10, err := utils.ParseGNMIElements([]string{"cont1a", "list2a[name=txout3]", "tx-power"})
	assert.NilError(t, err)
	changePath11, err := utils.ParseGNMIElements([]string{"leafAtTopLevel"})
	assert.NilError(t, err)

	mockTarget.EXPECT().Set(
		gomock.Any(),
		gomock.Any(),
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
		gomock.Any(),
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
		opstateChan, responseChan, opstateCache, roPathMap, mockTarget)
	assert.NilError(t, err, "Creating synchronizer")
	assert.Equal(t, s.ID, device1.ID)
	assert.Equal(t, s.Device.ID, device1.ID)
	assert.Equal(t, s.encoding, gnmi.Encoding_PROTO) // Should be default

	time.Sleep(100 * time.Millisecond) // Wait for response message

	// called asynchronously as it hangs until it gets a config event
	go s.syncConfigEventsToDevice(mockTarget, responseChan)

	//Create a change that we can send down to device
	value1, err := change.NewChangeValue("/cont1a/cont2a/leaf2a", change.NewTypedValueUint64(12), false)
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
	go s.syncOperationalState(context2.Background(), mockTarget, responseChan)
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
	err = s.handler(&gnmi.SubscribeResponse{
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
	err = s.handler(&gnmi.SubscribeResponse{
		Response: &subscribeResp2,
	})
	assert.NilError(t, err)
	time.Sleep(10 * time.Millisecond) // Wait for before sending a subscribe message

	// Test out all the switch cases for handler()
	err = s.handler(&gnmi.SubscribeResponse{
		Response: nil,
	})
	assert.ErrorContains(t, err, "unknown response")

	subscribeError := gnmi.SubscribeResponse_Error{Error: nil}
	err = s.handler(&gnmi.SubscribeResponse{
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
	err = s.handler(&gnmi.SubscribeResponse{
		Response: &subscribeResp3,
	})
	assert.NilError(t, err, "trying op state delete")

	// Even try closing the subscription
	subscribeSync := gnmi.SubscribeResponse_SyncResponse{SyncResponse: true}
	err = s.handler(&gnmi.SubscribeResponse{
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
		gomock.Any(), gomock.Any(),
	).Return(nil, status.Errorf(codes.Internal, "test, desc = error generated by mock"))

	go func() {
		for resp := range responseChan {
			assert.ErrorContains(t, resp.Error(), "error generated by mock")
		}
	}()

	s, err := New(context2.Background(), changeStore, configStore, device1, configChan,
		opstateChan, responseChan, opstateCache, roPathMap, mockTarget)
	assert.NilError(t, err, "Creating synchronizer")
	assert.Equal(t, s.ID, device1.ID)
	assert.Equal(t, s.Device.ID, device1.ID)
	assert.Equal(t, s.encoding, gnmi.Encoding_PROTO) // Should be default

	time.Sleep(100 * time.Millisecond) // Wait for response message
}
