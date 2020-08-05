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

	"github.com/onosproject/onos-config/test/utils/proto"

	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	device "github.com/onosproject/onos-topo/api/topo"
	"github.com/openconfig/gnmi/client"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

// TestSubscribeOnce tests subscription ONCE mode
func (s *TestSuite) TestSubscribeOnce(t *testing.T) {
	t.Skip()
	// Create a simulated device
	simulator := gnmi.CreateSimulator(t)

	// Wait for config to connect to the device
	gnmi.WaitForDeviceAvailable(t, device.ID(simulator.Name()), 10*time.Second)

	// Make a GNMI client to use for subscribe
	subC := client.BaseClient{}

	path, err := utils.ParseGNMIElements(utils.SplitPath(subTzPath))

	assert.NoError(t, err, "Unexpected error doing parsing")

	path.Target = simulator.Name()

	subReq := subscribeRequest{
		path:        path,
		subListMode: gpb.SubscriptionList_ONCE,
	}

	q, respChan, errQuery := buildQueryRequest(subReq, simulator)
	assert.NoError(t, errQuery, "Can't build Query")

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)
	// Set a value using gNMI client
	devicePath := gnmi.GetDevicePathWithValue(simulator.Name(), subTzPath, subTzValue, proto.StringVal)
	gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, gnmi.NoExtensions)
	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, subTzValue, 0, "Query after set returned the wrong value")

	var response *gpb.SubscribeResponse

	go func() {
		_ = subC.Subscribe(gnmi.MakeContext(), *q, "gnmi")
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
	gnmi.DeleteSimulator(t, simulator)
}

func awaitResponse(t *testing.T, respChan chan *gpb.SubscribeResponse, name string, delete bool, responseType string) {
	select {
	case response := <-respChan:
		validateResponse(t, response, name, delete)
	case <-time.After(10 * time.Second):
		assert.FailNow(t, "Expected "+responseType+" Response")
	}
}

// TestSubscribe tests a stream subscription to updates to a device
func (s *TestSuite) TestSubscribe(t *testing.T) {
	t.Skip()
	// Create a simulated device
	simulator := gnmi.CreateSimulator(t)

	// Wait for config to connect to the device
	gnmi.WaitForDeviceAvailable(t, device.ID(simulator.Name()), 10*time.Second)

	// Make a GNMI client to use for subscribe
	subC := client.BaseClient{}

	path, err := utils.ParseGNMIElements(utils.SplitPath(subTzPath))

	assert.NoError(t, err, "Unexpected error doing parsing")

	name := simulator.Name()
	path.Target = name

	subReq := subscribeRequest{
		path:          path,
		subListMode:   gpb.SubscriptionList_STREAM,
		subStreamMode: gpb.SubscriptionMode_TARGET_DEFINED,
	}

	q, respChan, errQuery := buildQueryRequest(subReq, simulator)
	assert.NoError(t, errQuery, "Can't build Query")

	// Subscription has to be spawned into a separate thread as it is blocking.
	go func() {
		_ = subC.Subscribe(gnmi.MakeContext(), *q, "gnmi")
	}()

	// Sleeping in order to make sure the subscribe request is properly stored and processed.
	time.Sleep(100000)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Set a value using gNMI client
	devicePath := gnmi.GetDevicePathWithValue(simulator.Name(), subTzPath, subTzValue, proto.StringVal)
	gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, gnmi.NoExtensions)

	const deleted = true
	const notDeleted = false

	// Wait for the Update response with Update
	awaitResponse(t, respChan, name, notDeleted, "Update")

	// Wait for the Sync response
	awaitResponse(t, respChan, name, notDeleted, "Sync")

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, subTzValue, 0, "Query after set returned the wrong value")

	// Remove the path we added
	gnmi.SetGNMIValueOrFail(t, gnmiClient, gnmi.NoPaths, devicePath, gnmi.NoExtensions)

	// Wait for the Update response with delete
	awaitResponse(t, respChan, name, deleted, "Update")

	// Wait for the Sync response
	awaitResponse(t, respChan, name, notDeleted, "Sync")

	//  Make sure it got removed
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")

	gnmi.DeleteSimulator(t, simulator)
}

func validateResponse(t *testing.T, resp *gpb.SubscribeResponse, device string, delete bool) {
	//No extension should be provided since the device should be connected.
	assert.Equal(t, 0, len(resp.Extension))

	switch v := resp.Response.(type) {
	default:
		assert.Fail(t, "Unknown type", v)
	case *gpb.SubscribeResponse_Error:
		assert.Fail(t, "Error ", v)
	case *gpb.SubscribeResponse_SyncResponse:
		assert.Equal(t, v.SyncResponse, true, "Sync should be true")
	case *gpb.SubscribeResponse_Update:
		if delete {
			assertDeleteResponse(t, v, device, subTzPath)
		} else {
			assertUpdateResponse(t, v, device, subTzPath, subTzValue)
		}
	}
}
