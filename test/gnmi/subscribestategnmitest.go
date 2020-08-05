// Copyright 2020-present Open Networking Foundation.
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
	"os"
	"testing"
	"time"

	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/client"
	gpb "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/onosproject/onos-config/test/utils/gnmi"
	device "github.com/onosproject/onos-topo/api/topo"
	"github.com/stretchr/testify/assert"
)

// TestSubscribeStateGnmi tests a stream subscription to updates to a device using the diags API
func (s *TestSuite) TestSubscribeStateGnmi(t *testing.T) {
	t.Skip()
	const dateTimePath = "/system/state/current-datetime"

	previousTime = time.Now().Add(-5 * time.Second)

	// Bring up a new simulated device
	simulator := gnmi.CreateSimulator(t)
	deviceName := simulator.Name()
	deviceID := device.ID(deviceName)

	// Wait for config to connect to the device
	gnmi.WaitForDeviceAvailable(t, deviceID, 10*time.Second)
	time.Sleep(250 * time.Millisecond)

	// Make a GNMI client to use for subscribe
	subC := client.BaseClient{}

	path, err := utils.ParseGNMIElements(utils.SplitPath(dateTimePath))

	assert.NoError(t, err, "Unexpected error doing parsing")

	path.Target = simulator.Name()

	subReq := subscribeRequest{
		path:          path,
		subListMode:   gpb.SubscriptionList_STREAM,
		subStreamMode: gpb.SubscriptionMode_TARGET_DEFINED,
	}

	q, respChan, errQuery := buildQueryRequest(subReq, simulator)
	assert.NoError(t, errQuery, "Can't build Query")

	var response *gpb.SubscribeResponse

	// Subscription has to be spawned into a separate thread as it is blocking.
	go func() {
		errSubscribe := subC.Subscribe(gnmi.MakeContext(), *q, "gnmi")
		fmt.Fprintf(os.Stderr, "Subscription Error %v", errSubscribe)
		assert.NoError(t, errSubscribe, "Subscription Error %v", errSubscribe)
	}()

	// Sleeping in order to make sure the subscribe request is properly stored and processed.
	time.Sleep(100000)

	i := 1
	for i <= 10 {
		select {
		case response = <-respChan:
			validateGnmiStateResponse(t, response, simulator.Name())
			i++
		case <-time.After(10 * time.Second):
			assert.FailNow(t, "Expected Update Response")
		}
	}
}

func validateGnmiStateResponse(t *testing.T, resp *gpb.SubscribeResponse, device string) {
	t.Helper()

	switch v := resp.Response.(type) {
	case *gpb.SubscribeResponse_Update:
		validateGnmiStateUpdateResponse(t, v, device)

	case *gpb.SubscribeResponse_SyncResponse:
		validateGnmiStateSyncResponse(t, v)

	default:
		assert.Fail(t, "Unknown GNMI state response type")
	}
}

func validateGnmiStateUpdateResponse(t *testing.T, update *gpb.SubscribeResponse_Update, device string) {
	t.Helper()
	assertUpdateResponse(t, update, device, subDateTimePath, "")
	updatedTimeString := update.Update.GetUpdate()[0].Val.GetStringVal()
	updatedTime, timeParseError := time.Parse("2006-01-02T15:04:05Z-07:00", updatedTimeString)
	assert.NoError(t, timeParseError)

	assert.True(t, previousTime.Before(updatedTime), "Path time value is not in the future")
	previousTime = updatedTime
}

func validateGnmiStateSyncResponse(t *testing.T, sync *gpb.SubscribeResponse_SyncResponse) {
	t.Helper()
	assertSyncResponse(t, sync)
}
