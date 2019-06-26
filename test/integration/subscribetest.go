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

package integration

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/test/env"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	log "k8s.io/klog"
	"testing"
	"time"
)

const (
	subValue = "Europe/Dublin"
	subPath  = "/system/clock/config/timezone-name"
)

func init() {
	Registry.Register("subscribe-test", TestSubscribe)
}

func TestSubscribe(t *testing.T) {

	// Get the first configured device from the environment.
	device := env.GetDevices()[0]

	// Make a GNMI client to use for subscribe
	subC := client.BaseClient{}

	path, err := utils.ParseGNMIElements(utils.SplitPath(subPath))

	assert.NoError(t, err, "Unexpected error doing parsing")

	path.Target = device

	request, errReq := buildRequest(path, gnmi.SubscriptionList_STREAM)

	assert.NoError(t, errReq, "Can't build Request")

	q, errQuery := client.NewQuery(request)

	assert.NoError(t, errQuery, "Can't build Query")

	dest, errDest := env.GetDestination("")

	assert.NoError(t, errDest, "Can't build Destination")

	q.Addrs = dest.Addrs
	q.Timeout = dest.Timeout
	q.Target = dest.Target
	q.Credentials = dest.Credentials
	q.TLS = dest.TLS

	respChan := make(chan *gnmi.SubscribeResponse)

	q.ProtoHandler = func(msg proto.Message) error {
		resp, ok := msg.(*gnmi.SubscribeResponse)
		if !ok {
			return fmt.Errorf("failed to type assert message %#v", msg)
		}
		respChan <- resp
		return nil
	}

	// Subscription has to be spawned into a separate thread as it is blocking.
	go func() {
		errSubscribe := subC.Subscribe(MakeContext(), q, "gnmi")
		assert.NoError(t, errSubscribe, "Subscription Error")
	}()

	// Make a GNMI client to use for requests
	c, err := env.NewGnmiClient(MakeContext(), "")
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Set a value using gNMI client
	setPath := makeDevicePath(device, subPath)
	setPath[0].value = subValue
	_, errorSet := GNMISet(MakeContext(), c, setPath)
	assert.NoError(t, errorSet)
	var response *gnmi.SubscribeResponse

	// Wait for the Update response with Update
	select {
	case response = <-respChan:
		valiedateResponse(t, response, device, false)
	case <-time.After(50 * time.Millisecond):
		assert.FailNow(t, "Expected Update Response")
	}

	// Wait for the Sync response
	select {
	case response = <-respChan:
		valiedateResponse(t, response, device, false)
	case <-time.After(50 * time.Millisecond):
		assert.FailNow(t, "Expected Sync Response")
	}

	// Check that the value was set correctly
	valueAfter, errorAfter := GNMIGet(MakeContext(), c, makeDevicePath(device, subPath))
	assert.NoError(t, errorAfter)
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, subValue, valueAfter[0].value,
		"Query after set returned the wrong value: %s\n", valueAfter)

	// Remove the path we added
	errorDelete := GNMIDelete(MakeContext(), c, makeDevicePath(device, subPath))
	assert.NoError(t, errorDelete)

	// Wait for the Update response with delete
	select {
	case response = <-respChan:
		valiedateResponse(t, response, device, true)
	case <-time.After(50 * time.Millisecond):
		assert.FailNow(t, "Expected Delete Response")
	}

	// Wait for the Sync response
	select {
	case response = <-respChan:
		valiedateResponse(t, response, device, false)
	case <-time.After(50 * time.Millisecond):
		assert.FailNow(t, "Expected Sync Response")
	}

	//  Make sure it got removed
	valueAfterDelete, errorAfterDelete := GNMIGet(MakeContext(), c, makeDevicePath(device, subPath))
	assert.NoError(t, errorAfterDelete)
	assert.Equal(t, valueAfterDelete[0].value, "",
		"incorrect value found for path /system/clock/config/timezone-name after delete")
}

func buildRequest(path *gnmi.Path, mode gnmi.SubscriptionList_Mode) (*gnmi.SubscribeRequest, error) {
	prefixPath, err := utils.ParseGNMIElements(utils.SplitPath(""))
	if err != nil {
		return nil, err
	}
	subscription := &gnmi.Subscription{
		Path: path,
		Mode: gnmi.SubscriptionMode_TARGET_DEFINED,
	}
	subscriptions := make([]*gnmi.Subscription, 0)
	subscriptions = append(subscriptions, subscription)
	subList := &gnmi.SubscriptionList{
		Subscription: subscriptions,
		Mode:         mode,
		UpdatesOnly:  true,
		Prefix:       prefixPath,
	}
	request := &gnmi.SubscribeRequest{
		Request: &gnmi.SubscribeRequest_Subscribe{
			Subscribe: subList,
		},
	}
	return request, nil
}

func valiedateResponse(t *testing.T, resp *gnmi.SubscribeResponse, device string, delete bool) error {
	switch v := resp.Response.(type) {
	default:
		assert.Fail(t, "Unknown type", v)
	case *gnmi.SubscribeResponse_Error:
		assert.Fail(t, "Error ", v)
	case *gnmi.SubscribeResponse_SyncResponse:
		assert.Equal(t, v.SyncResponse, true, "Sync should be true")
	case *gnmi.SubscribeResponse_Update:
		if delete {
			assertDeleteResponse(t, v, device)
		} else {
			assertUpdateResponse(t, v, device)
		}
	}
	return nil
}

func assertUpdateResponse(t *testing.T, response *gnmi.SubscribeResponse_Update, device string) {
	assert.True(t, response.Update != nil, "Update should not be nil")
	assert.Equal(t, len(response.Update.GetUpdate()), 1)
	if response.Update.GetUpdate()[0] == nil {
		log.Error("response should contain at least one update and that should not be nil")
		t.FailNow()
	}
	pathResponse := response.Update.GetUpdate()[0].Path
	assert.Equal(t, pathResponse.Target, device)
	assert.Equal(t, len(pathResponse.Elem), 4, "Expected 4 path elements")
	assert.Equal(t, pathResponse.Elem[0].Name, "system")
	assert.Equal(t, pathResponse.Elem[1].Name, "clock")
	assert.Equal(t, pathResponse.Elem[2].Name, "config")
	assert.Equal(t, pathResponse.Elem[3].Name, "timezone-name")
	assert.Equal(t, response.Update.GetUpdate()[0].Val.GetStringVal(), subValue)
}

func assertDeleteResponse(t *testing.T, response *gnmi.SubscribeResponse_Update, device string) {
	assert.True(t, response.Update != nil, "Delete should not be nil")
	assert.Equal(t, len(response.Update.GetDelete()), 1)
	if response.Update.GetDelete()[0] == nil {
		log.Error("response should contain at least one delete path")
		t.FailNow()
	}
	pathResponse := response.Update.GetDelete()[0]
	assert.Equal(t, pathResponse.Target, device)
	assert.Equal(t, len(pathResponse.Elem), 4, "Expected 3 path elements")
	assert.Equal(t, pathResponse.Elem[0].Name, "system")
	assert.Equal(t, pathResponse.Elem[1].Name, "clock")
	assert.Equal(t, pathResponse.Elem[2].Name, "config")
	assert.Equal(t, pathResponse.Elem[3].Name, "timezone-name")
}
