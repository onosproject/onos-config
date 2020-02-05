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
	"testing"
	"time"

	"github.com/onosproject/onos-config/pkg/utils"
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

// TestSubscribeOnce tests subscription ONCE mode
func (s *TestSuite) TestSubscribeOnce(t *testing.T) {
	simulator := env.NewSimulator().AddOrDie()

	// Wait for config to connect to the device
	testutils.WaitForDeviceAvailable(t, device.ID(simulator.Name()), 10*time.Second)

	// Make a GNMI client to use for subscribe
	subC := client.BaseClient{}

	path, err := utils.ParseGNMIElements(utils.SplitPath(subTzPath))

	assert.NoError(t, err, "Unexpected error doing parsing")

	path.Target = simulator.Name()

	subReq := subscribeRequest{
		path:        path,
		subListMode: gnmi.SubscriptionList_ONCE,
	}

	q, respChan, errQuery := buildQueryRequest(subReq)
	assert.NoError(t, errQuery, "Can't build Query")

	// Make a GNMI client to use for requests
	gnmiClient := getGNMIClientOrFail(t)
	// Set a value using gNMI client
	devicePath := getDevicePathWithValue(simulator.Name(), subTzPath, subTzValue, StringVal)
	setGNMIValueOrFail(t, gnmiClient, devicePath, noPaths, noExtensions)
	// Check that the value was set correctly
	checkGNMIValue(t, gnmiClient, devicePath, subTzValue, 0, "Query after set returned the wrong value")

	var response *gnmi.SubscribeResponse

	go func() {
		errSubscribe := subC.Subscribe(testutils.MakeContext(), *q, "gnmi")
		assert.NoError(t, errSubscribe, "Subscription Error")
	}()

	select {
	case response = <-respChan:
		validateResponse(t, response, simulator.Name(), false)
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "Expected Update Response")
	}

	// Wait for the Sync response
	select {
	case response = <-respChan:
		validateResponse(t, response, simulator.Name(), false)
	case <-time.After(1 * time.Second):
		assert.FailNow(t, "Expected Sync Response")
	}

}

// TestSubscribe tests a stream subscription to updates to a device
func (s *TestSuite) TestSubscribe(t *testing.T) {
	simulator := env.NewSimulator().AddOrDie()

	// Wait for config to connect to the device
	testutils.WaitForDeviceAvailable(t, device.ID(simulator.Name()), 10*time.Second)

	// Make a GNMI client to use for subscribe
	subC := client.BaseClient{}

	path, err := utils.ParseGNMIElements(utils.SplitPath(subTzPath))

	assert.NoError(t, err, "Unexpected error doing parsing")

	path.Target = simulator.Name()

	subReq := subscribeRequest{
		path:          path,
		subListMode:   gnmi.SubscriptionList_STREAM,
		subStreamMode: gnmi.SubscriptionMode_TARGET_DEFINED,
	}

	q, respChan, errQuery := buildQueryRequest(subReq)
	assert.NoError(t, errQuery, "Can't build Query")

	// Subscription has to be spawned into a separate thread as it is blocking.
	go func() {
		errSubscribe := subC.Subscribe(testutils.MakeContext(), *q, "gnmi")
		assert.NoError(t, errSubscribe, "Subscription Error: %v", errSubscribe)
	}()

	// Sleeping in order to make sure the subscribe request is properly stored and processed.
	time.Sleep(100000)

	// Make a GNMI client to use for requests
	gnmiClient := getGNMIClientOrFail(t)

	// Set a value using gNMI client
	devicePath := getDevicePathWithValue(simulator.Name(), subTzPath, subTzValue, StringVal)
	setGNMIValueOrFail(t, gnmiClient, devicePath, noPaths, noExtensions)
	var response *gnmi.SubscribeResponse

	// Wait for the Update response with Update
	select {
	case response = <-respChan:
		validateResponse(t, response, simulator.Name(), false)
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "Expected Update Response")
	}

	// Wait for the Sync response
	select {
	case response = <-respChan:
		validateResponse(t, response, simulator.Name(), false)
	case <-time.After(1 * time.Second):
		assert.FailNow(t, "Expected Sync Response")
	}

	// Check that the value was set correctly
	checkGNMIValue(t, gnmiClient, devicePath, subTzValue, 0, "Query after set returned the wrong value")

	// Remove the path we added
	setGNMIValueOrFail(t, gnmiClient, noPaths, devicePath, noExtensions)

	// Wait for the Update response with delete
	select {
	case response = <-respChan:
		validateResponse(t, response, simulator.Name(), true)
	case <-time.After(1 * time.Second):
		assert.FailNow(t, "Expected Delete Response")
	}

	// Wait for the Sync response
	select {
	case response = <-respChan:
		validateResponse(t, response, simulator.Name(), false)
	case <-time.After(1 * time.Second):
		assert.FailNow(t, "Expected Sync Response")
	}

	//  Make sure it got removed
	checkGNMIValue(t, gnmiClient, devicePath, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")
}

func validateResponse(t *testing.T, resp *gnmi.SubscribeResponse, device string, delete bool) {
	//No extension should be provided since the device should be connected.
	assert.Equal(t, 0, len(resp.Extension))

	switch v := resp.Response.(type) {
	default:
		assert.Fail(t, "Unknown type", v)
	case *gnmi.SubscribeResponse_Error:
		assert.Fail(t, "Error ", v)
	case *gnmi.SubscribeResponse_SyncResponse:
		assert.Equal(t, v.SyncResponse, true, "Sync should be true")
	case *gnmi.SubscribeResponse_Update:
		if delete {
			assertDeleteResponse(t, v, device, subTzPath)
		} else {
			assertUpdateResponse(t, v, device, subTzPath, subTzValue)
		}
	}
}
