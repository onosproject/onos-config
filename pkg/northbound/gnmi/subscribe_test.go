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

package gnmi

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/store/change"
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/assert"
	log "k8s.io/klog"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"
)

const subscribeDelay = 100 * time.Millisecond

type gNMISubscribeServerFake struct {
	Request   *gnmi.SubscribeRequest
	Responses chan *gnmi.SubscribeResponse
	Signal    chan struct{}
	grpc.ServerStream
}

func (x gNMISubscribeServerFake) Send(m *gnmi.SubscribeResponse) error {
	x.Responses <- m
	if m.GetSyncResponse() {
		close(x.Responses)
	}
	return nil
}

func (x gNMISubscribeServerFake) Recv() (*gnmi.SubscribeRequest, error) {
	<-x.Signal
	return x.Request, nil
}

type gNMISubscribeServerPollFake struct {
	Request     *gnmi.SubscribeRequest
	PollRequest *gnmi.SubscribeRequest
	Responses   chan *gnmi.SubscribeResponse
	Signal      chan struct{}
	grpc.ServerStream
	first bool
}

func (x gNMISubscribeServerPollFake) Send(m *gnmi.SubscribeResponse) error {
	x.Responses <- m
	return nil
}

func (x gNMISubscribeServerPollFake) Recv() (*gnmi.SubscribeRequest, error) {
	<-x.Signal
	if x.first {
		return x.Request, nil
	}
	return x.PollRequest, nil

}

// Test_SubscribeLeafOnce tests subscribing with mode ONCE and then immediately receiving the subscription for a specific leaf.
func Test_SubscribeLeafOnce(t *testing.T) {
	server, mgr, mockStore, _, _ := setUp(t)
	mockStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found"))

	var wg sync.WaitGroup
	defer tearDown(mgr, &wg)

	path, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})

	assert.NilError(t, err, "Unexpected error doing parsing")

	path.Target = "Device1"

	request := buildRequest(path, gnmi.SubscriptionList_ONCE)

	responsesChan := make(chan *gnmi.SubscribeResponse)
	serverFake := gNMISubscribeServerFake{
		Request:   request,
		Responses: responsesChan,
		Signal:    make(chan struct{}),
	}
	go func() {
		err = server.Subscribe(serverFake)
	}()

	serverFake.Signal <- struct{}{}
	assert.NilError(t, err, "Unexpected error doing Subscribe")

	device1 := "Device1"
	path1Once := "cont1a"
	path2Once := "cont2a"
	path3Once := "leaf2a"
	value := uint(13)

	// Wait for the Update response with Update
	assertUpdateResponse(t, responsesChan, device1, path1Once, path2Once, path3Once, value, true)

}

// Test_SubscribeLeafDelete tests subscribing with mode STREAM and then issuing a set request with updates for that path
func Test_SubscribeLeafStream(t *testing.T) {
	server, mgr, mockDeviceStore, mockNetworkChangesStore, _ := setUp(t)
	mockDeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(2)
	mockNetworkChangesStore.EXPECT().Create(gomock.Any())

	var wg sync.WaitGroup
	defer tearDown(mgr, &wg)

	path, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf4a"})

	assert.NilError(t, err, "Unexpected error doing parsing")

	path.Target = "Device1"

	request := buildRequest(path, gnmi.SubscriptionList_STREAM)

	responsesChan := make(chan *gnmi.SubscribeResponse, 1)
	serverFake := gNMISubscribeServerFake{
		Request:   request,
		Responses: responsesChan,
		Signal:    make(chan struct{}),
	}

	go func() {
		err = server.Subscribe(serverFake)
		assert.NilError(t, err, "Unexpected error doing Subscribe")
	}()

	//FIXME Waiting for subscribe to finish properly --> when event is issued assuring state consistency we can remove
	time.Sleep(subscribeDelay)

	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)
	//augmenting pre-existing value by one
	typedValue := gnmi.TypedValue_UintVal{UintVal: 14}
	value := gnmi.TypedValue{Value: &typedValue}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: path, Val: &value})
	setRequest := &gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}
	//Sending set request
	go func() {
		_, err = server.Set(context.Background(), setRequest)
		assert.NilError(t, err, "Unexpected error doing Set")
		serverFake.Signal <- struct{}{}
	}()

	device1 := "Device1"
	path1Stream := "cont1a"
	path2Stream := "cont2a"
	path3Stream := "leaf4a"
	valueReply := uint(14)

	//Expecting 1 Update response
	assertUpdateResponse(t, responsesChan, device1, path1Stream, path2Stream, path3Stream, valueReply, true)
	//And one sync response
	assertSyncResponse(responsesChan, t)

}

func Test_WrongDevice(t *testing.T) {
	_, mgr, mockStore, _, _ := setUp(t)
	mockStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found"))

	path, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf4a"})

	assert.NilError(t, err, "Unexpected error doing parsing")

	path.Target = "Device1"

	request := buildRequest(path, gnmi.SubscriptionList_STREAM)

	changeChan := make(chan events.ConfigEvent)
	responsesChan := make(chan *gnmi.SubscribeResponse, 1)
	serverFake := gNMISubscribeServerFake{
		Request:   request,
		Responses: responsesChan,
		Signal:    make(chan struct{}),
	}

	targets := make(map[string]struct{})
	subs := make([]*regexp.Regexp, 0)
	resChan := make(chan result)
	targets["Device2"] = struct{}{}
	go listenForUpdates(changeChan, serverFake, mgr, targets, subs, resChan)
	changeChan <- events.NewConfigEvent("Device1", []byte("test"), true)
	var response *gnmi.SubscribeResponse
	select {
	case response = <-responsesChan:
		log.Error("Should not be receiving response ", response)
		t.FailNow()
	case <-time.After(50 * time.Millisecond):
	}
	targets["Device1"] = struct{}{}
	subs = append(subs, utils.MatchWildcardRegexp("/cont1a/*/leaf3c"))
	go listenForUpdates(changeChan, serverFake, mgr, targets, subs, resChan)
	config1Value05, _ := types.NewChangeValue("/cont1a/cont2a/leaf2c", types.NewTypedValueString("def"), false)
	config1Value09, _ := types.NewChangeValue("/cont1a/list2a[name=txout2]", types.NewTypedValueEmpty(), true)
	change1, _ := change.NewChange([]*types.ChangeValue{config1Value05, config1Value09}, "Remove txout 2")
	changeChan <- events.NewConfigEvent("Device1", change1.ID, true)
	select {
	case response = <-responsesChan:
		log.Error("Should not be receiving response ", response)
		t.FailNow()
	case <-time.After(50 * time.Millisecond):
	}

}

func Test_WrongPath(t *testing.T) {
	_, mgr, mockStore, _, _ := setUp(t)
	mockStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found"))

	path, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf4a"})

	assert.NilError(t, err, "Unexpected error doing parsing")

	path.Target = "Device1"

	request := buildRequest(path, gnmi.SubscriptionList_STREAM)

	changeChan := make(chan events.ConfigEvent)
	responsesChan := make(chan *gnmi.SubscribeResponse, 1)
	serverFake := gNMISubscribeServerFake{
		Request:   request,
		Responses: responsesChan,
		Signal:    make(chan struct{}),
	}

	targets := make(map[string]struct{})
	resChan := make(chan result)
	var response *gnmi.SubscribeResponse
	targets["Device1"] = struct{}{}
	subscriptionPathStr := "/test1:cont1a/cont2a/leaf3c"
	subsStr := make([]*regexp.Regexp, 0)
	subsStr = append(subsStr, utils.MatchWildcardRegexp(subscriptionPathStr))
	go listenForUpdates(changeChan, serverFake, mgr, targets, subsStr, resChan)
	config1Value05, _ := types.NewChangeValue("/test1:cont1a/cont2a/leaf2c", types.NewTypedValueString("def"), false)
	config1Value09, _ := types.NewChangeValue("/test1:cont1a/list2a[name=txout2]", types.NewTypedValueEmpty(), true)
	change1, _ := change.NewChange([]*types.ChangeValue{config1Value05, config1Value09}, "Remove txout 2")
	changeChan <- events.NewConfigEvent("Device1", change1.ID, true)
	select {
	case response = <-responsesChan:
		log.Error("Should not be receiving response ", response)
		t.FailNow()
	case <-time.After(50 * time.Millisecond):
	}

}

func Test_ErrorDoubleSubscription(t *testing.T) {
	server, mgr, mockStore, _, _ := setUp(t)
	mockStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found"))
	var wg sync.WaitGroup
	defer tearDown(mgr, &wg)

	path, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf4a"})

	assert.NilError(t, err, "Unexpected error doing parsing")

	path.Target = "Device1"

	request := buildRequest(path, gnmi.SubscriptionList_STREAM)

	responsesChan := make(chan *gnmi.SubscribeResponse, 1)
	serverFake := gNMISubscribeServerFake{
		Request:   request,
		Responses: responsesChan,
		Signal:    make(chan struct{}),
	}

	go func() {
		err = server.Subscribe(serverFake)
		assert.NilError(t, err, "Unexpected error doing Subscribe")
		var response *gnmi.SubscribeResponse
		select {
		case response = <-responsesChan:
			log.Error("Should not be receiving response ", response)
			t.Fail()
		case <-time.After(50 * time.Millisecond):
		}
	}()
	//FIXME Waiting for subscribe to finish properly --> when event is issued assuring state consistency we can remove
	time.Sleep(subscribeDelay)

	err = server.Subscribe(serverFake)
	assert.ErrorContains(t, err, "is already registered")
}

func Test_Poll(t *testing.T) {
	server, mgr, mockStore, _, _ := setUp(t)
	mockStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(2)

	var wg sync.WaitGroup
	defer tearDown(mgr, &wg)

	path, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})

	assert.NilError(t, err, "Unexpected error doing parsing")

	path.Target = "Device1"

	request := buildRequest(path, gnmi.SubscriptionList_POLL)

	pollrequest := &gnmi.SubscribeRequest{
		Request: &gnmi.SubscribeRequest_Poll{
			Poll: &gnmi.Poll{},
		},
	}

	responsesChan := make(chan *gnmi.SubscribeResponse, 1)
	serverFake := gNMISubscribeServerPollFake{
		Request:     request,
		Responses:   responsesChan,
		Signal:      make(chan struct{}),
		first:       true,
		PollRequest: pollrequest,
	}

	go func() {
		err = server.Subscribe(serverFake)
		assert.NilError(t, err, "Unexpected error doing Subscribe")
	}()

	serverFake.Signal <- struct{}{}
	time.Sleep(subscribeDelay) //waiting before sending fake poll
	serverFake.first = false
	serverFake.Signal <- struct{}{}

	device1 := "Device1"
	path1Poll := "cont1a"
	path2Poll := "cont2a"
	path3Poll := "leaf2a"
	value := uint(13)

	//Expecting first Update response
	assertUpdateResponse(t, responsesChan, device1, path1Poll, path2Poll, path3Poll, value, true)
	//And first sync response
	assertSyncResponse(responsesChan, t)

	//Expecting second Update response
	assertUpdateResponse(t, responsesChan, device1, path1Poll, path2Poll, path3Poll, value, true)
	//And second sync response
	assertSyncResponse(responsesChan, t)

	close(serverFake.Responses)

}

// Test_SubscribeLeafDelete tests subscribing with mode STREAM and then issuing a set request with delete paths
func Test_SubscribeLeafStreamDelete(t *testing.T) {
	server, mgr, mockDeviceStore, mockNetworkChangesStore, _ := setUp(t)
	mockDeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(2)
	mockNetworkChangesStore.EXPECT().Create(gomock.Any())

	var wg sync.WaitGroup
	defer tearDown(mgr, &wg)

	path, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})

	assert.NilError(t, err, "Unexpected error doing parsing")

	path.Target = "Device1"

	request := buildRequest(path, gnmi.SubscriptionList_STREAM)

	responsesChan := make(chan *gnmi.SubscribeResponse, 1)
	serverFake := gNMISubscribeServerFake{
		Request:   request,
		Responses: responsesChan,
		Signal:    make(chan struct{}),
	}

	go func() {
		err = server.Subscribe(serverFake)
		assert.NilError(t, err, "Unexpected error doing Subscribe")
	}()

	//FIXME Waiting for subscribe to finish properly --> when event is issued assuring state consistency we can remove
	time.Sleep(subscribeDelay)

	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)

	deletePaths = append(deletePaths, path)

	var setRequest = &gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}
	//Sending set request
	go func() {
		_, err = server.Set(context.Background(), setRequest)
		assert.NilError(t, err, "Unexpected error doing Set")
		serverFake.Signal <- struct{}{}
	}()

	device1 := "Device1"
	path1Stream := "cont1a"
	path2Stream := "cont2a"
	path3Stream := "leaf2a"

	//Expecting one delete response
	assertDeleteResponse(t, responsesChan, device1, path1Stream, path2Stream, path3Stream)
	// and one sync response
	assertSyncResponse(responsesChan, t)

}

// Test_SubscribeLeafStreamWithDeviceLoaded tests subscribing with mode STREAM for an existing device
// and then issuing a set request with updates for that path
func Test_SubscribeLeafStreamWithDeviceLoaded(t *testing.T) {
	server, mgr, mockDeviceStore, mockNetworkChangesStore, _ := setUp(t)

	targetStr := "Device1"
	target := device.ID(targetStr)
	presentDevice := &device.Device{
		ID: target,
	}

	mockDeviceStore.EXPECT().Get(gomock.Any()).Return(presentDevice, nil)
	mockDeviceStore.EXPECT().Get(gomock.Any()).Return(presentDevice, nil)
	mockNetworkChangesStore.EXPECT().Create(gomock.Any())

	configChan, respChan, err := mgr.Dispatcher.RegisterDevice(target)

	path, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf4a"})

	assert.NilError(t, err, "Unexpected error doing parsing")

	path.Target = targetStr

	request := buildRequest(path, gnmi.SubscriptionList_STREAM)

	responsesChan := make(chan *gnmi.SubscribeResponse, 1)
	serverFake := gNMISubscribeServerFake{
		Request:   request,
		Responses: responsesChan,
		Signal:    make(chan struct{}),
	}

	go func() {
		err = server.Subscribe(serverFake)
		assert.NilError(t, err, "Unexpected error doing Subscribe")
	}()
	serverFake.Signal <- struct{}{}
	//FIXME Waiting for subscribe to finish properly --> when event is issued assuring state consistency we can remove
	time.Sleep(subscribeDelay)
	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)
	//augmenting pre-existing value by one
	typedValue := gnmi.TypedValue_UintVal{UintVal: 14}
	value := gnmi.TypedValue{Value: &typedValue}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: path, Val: &value})
	setRequest := &gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	go func() {
		cfg := <-configChan
		assert.Assert(t, cfg.Applied())
		go func() {
			respChan <- events.NewResponseEvent(events.EventTypeAchievedSetConfig, targetStr, []byte(cfg.ChangeID()), "")
		}()
	}()
	//Sending set request
	go func() {
		_, err = server.Set(context.Background(), setRequest)
		assert.NilError(t, err, "Unexpected error doing Set")
	}()

	device1 := "Device1"
	path1Stream := "cont1a"
	path2Stream := "cont2a"
	path3Stream := "leaf4a"
	valueReply := uint(14)

	//Expecting 1 Update response
	assertUpdateResponse(t, responsesChan, device1, path1Stream, path2Stream, path3Stream, valueReply, false)
	//And one sync response
	assertSyncResponse(responsesChan, t)
	var wg sync.WaitGroup
	tearDown(mgr, &wg)
}

func buildRequest(path *gnmi.Path, mode gnmi.SubscriptionList_Mode) *gnmi.SubscribeRequest {
	subscription := &gnmi.Subscription{
		Path: path,
		Mode: gnmi.SubscriptionMode_TARGET_DEFINED,
	}
	subscriptions := make([]*gnmi.Subscription, 0)
	subscriptions = append(subscriptions, subscription)
	subList := &gnmi.SubscriptionList{
		Subscription: subscriptions,
		Mode:         mode,
	}
	request := &gnmi.SubscribeRequest{
		Request: &gnmi.SubscribeRequest_Subscribe{
			Subscribe: subList,
		},
	}
	return request
}

func assertSyncResponse(responsesChan chan *gnmi.SubscribeResponse, t *testing.T) {
	select {
	case response := <-responsesChan:
		log.Info("response ", response)
		assert.Equal(t, response.GetSyncResponse(), true, "Sync should be true")
	case <-time.After(1 * time.Second):
		log.Error("Expected Sync Response")
		t.FailNow()
	}
}

func assertUpdateResponse(t *testing.T, responsesChan chan *gnmi.SubscribeResponse, device1 string,
	path1 string, path2 string, path3 string, value uint, disconnected bool) {
	select {
	case response := <-responsesChan:
		assert.Assert(t, response.GetUpdate().GetUpdate() != nil, "Update should not be nil")
		assert.Equal(t, len(response.GetUpdate().GetUpdate()), 1)
		if response.GetUpdate().GetUpdate()[0] == nil {
			log.Error("response should contain at least one update and that should not be nil")
			t.FailNow()
		}
		pathResponse := response.GetUpdate().GetUpdate()[0].Path
		assert.Equal(t, pathResponse.Target, device1)
		assert.Equal(t, len(pathResponse.Elem), 3, "Expected 3 path elements")
		assert.Equal(t, pathResponse.Elem[0].Name, path1)
		assert.Equal(t, pathResponse.Elem[1].Name, path2)
		assert.Equal(t, pathResponse.Elem[2].Name, path3)
		assert.Equal(t, response.GetUpdate().GetUpdate()[0].Val.GetUintVal(), uint64(value))
		if disconnected {
			// Check that the device is currently disconnected
			assert.Equal(t, len(response.Extension), 1)

			extensionDeviceState := response.Extension[0].GetRegisteredExt()
			assert.Equal(t, extensionDeviceState.Id.String(), strconv.Itoa(GnmiExtensionDevicesNotConnected))
			assert.Equal(t, string(extensionDeviceState.Msg), device1)
		} else {
			assert.Equal(t, len(response.Extension), 0)
		}

	case <-time.After(5 * time.Second):
		log.Error("Expected Update Response")
		t.FailNow()
	}

}

func assertDeleteResponse(t *testing.T, responsesChan chan *gnmi.SubscribeResponse, device1 string,
	path1 string, path2 string, path3 string) {
	select {
	case response := <-responsesChan:
		log.Info("response ", response)
		assert.Assert(t, response.GetUpdate() != nil, "Delete should not be nil")
		assert.Equal(t, len(response.GetUpdate().GetDelete()), 1)
		if response.GetUpdate().GetDelete()[0] == nil {
			log.Error("response should contain at least one delete path")
			t.FailNow()
		}
		pathResponse := response.GetUpdate().GetDelete()[0]
		assert.Equal(t, pathResponse.Target, device1)
		assert.Equal(t, len(pathResponse.Elem), 3, "Expected 3 path elements")
		assert.Equal(t, pathResponse.Elem[0].Name, path1)
		assert.Equal(t, pathResponse.Elem[1].Name, path2)
		assert.Equal(t, pathResponse.Elem[2].Name, path3)
	case <-time.After(1 * time.Second):
		log.Error("Expected Update Response")
		t.FailNow()
	}
}
