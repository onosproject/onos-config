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
	"fmt"
	"testing"
	"time"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/openconfig/gnmi/client"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	ocgnmi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

const (
	timeOut            = 5
	expectedNumUpdates = 2
	expectedNumSyncs   = 2
)

// TestSubscribeOnce tests subscription ONCE mode
func (s *TestSuite) TestSubscribeOnce(t *testing.T) {
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

	q, errQuery := buildQueryRequest(subReq)
	assert.NoError(t, errQuery, "Can't build Query")

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)
	// Set a value using gNMI client
	devicePath := gnmi.GetDevicePathWithValue(simulator.Name(), subTzPath, subTzValue, proto.StringVal)
	gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, gnmi.NoExtensions)
	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, subTzValue, 0, "Query after set returned the wrong value")

	q.ProtoHandler = func(msg protobuf.Message) error {
		fmt.Println("proto handler is called")
		resp, ok := msg.(*ocgnmi.SubscribeResponse)
		if !ok {
			return fmt.Errorf("failed to type assert message %#v", msg)
		}
		validateResponse(t, resp, simulator.Name(), false)
		return nil
	}

	err = subC.Subscribe(gnmi.MakeContext(), *q, "gnmi")
	defer subC.Close()
	assert.NoError(t, err)
	gnmi.DeleteSimulator(t, simulator)
}

// TestSubscribe tests a stream subscription to updates to a device
func (s *TestSuite) TestSubscribe(t *testing.T) {
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
		subStreamMode: gpb.SubscriptionMode_ON_CHANGE,
	}

	q, errQuery := buildQueryRequest(subReq)
	assert.NoError(t, errQuery, "Can't build Query")

	updateCount := 0
	syncCount := 0

	done := make(chan bool, 1)

	q.ProtoHandler = func(msg protobuf.Message) error {
		resp, ok := msg.(*ocgnmi.SubscribeResponse)
		if !ok {
			return fmt.Errorf("failed to type assert message %#v", msg)
		}

		switch v := resp.Response.(type) {
		case *gpb.SubscribeResponse_Update:
			// validate update response
			if len(resp.GetUpdate().Update) == 1 {
				validateResponse(t, resp, simulator.Name(), false)
			}

			// validate delete response
			if len(resp.GetUpdate().Delete) == 1 {
				validateResponse(t, resp, simulator.Name(), true)
			}
			s.mux.Lock()
			updateCount++
			s.mux.Unlock()

		case *gpb.SubscribeResponse_Error:
			return fmt.Errorf("error in response: %s", v)
		case *gpb.SubscribeResponse_SyncResponse:
			// validate sync response
			validateResponse(t, resp, simulator.Name(), false)
			s.mux.Lock()
			syncCount++
			s.mux.Unlock()
		default:
			return fmt.Errorf("unknown response %T: %s", v, v)
		}

		if syncCount == expectedNumSyncs && updateCount == expectedNumUpdates {
			done <- true
		}
		return nil
	}

	go func() {
		_ = subC.Subscribe(gnmi.MakeContext(), *q, "gnmi")
		defer subC.Close()

	}()

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Set a value using gNMI client
	devicePath := gnmi.GetDevicePathWithValue(simulator.Name(), subTzPath, subTzValue, proto.StringVal)
	gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, gnmi.NoExtensions)

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, subTzValue, 0, "Query after set returned the wrong value")

	// Remove the path we added
	gnmi.SetGNMIValueOrFail(t, gnmiClient, gnmi.NoPaths, devicePath, gnmi.NoExtensions)

	//  Make sure it got removed
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")

	select {
	case <-done:
		gnmi.DeleteSimulator(t, simulator)
	case <-time.After(timeOut * time.Second):
		assert.FailNow(t, "timed out; the request should be processed under %d:", timeOut)
	}

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
