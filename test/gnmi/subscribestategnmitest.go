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
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
)

// TestSubscribeStateGnmi tests a stream subscription to updates to a device using the diags API
func (s *TestSuite) TestSubscribeStateGnmi(t *testing.T) {
	const dateTimePath = "/system/state/current-datetime"

	previousTime = time.Now().Add(-5 * time.Second)

	// Bring up a new simulated device
	simulator := env.NewSimulator().AddOrDie()
	deviceName := simulator.Name()
	deviceID := device.ID(deviceName)

	// Wait for config to connect to the device
	testutils.WaitForDeviceAvailable(t, deviceID, 10*time.Second)
	time.Sleep(250 * time.Millisecond)

	// Make a GNMI client to use for subscribe
	subC := client.BaseClient{}

	path, err := utils.ParseGNMIElements(utils.SplitPath(dateTimePath))

	assert.NoError(t, err, "Unexpected error doing parsing")

	path.Target = simulator.Name()

	subReq := subscribeRequest{
		path:          path,
		subListMode:   gnmi.SubscriptionList_STREAM,
		subStreamMode: gnmi.SubscriptionMode_TARGET_DEFINED,
	}

	request, errReq := buildRequest(subReq)

	assert.NoError(t, errReq, "Can't build Request")

	q, errQuery := client.NewQuery(request)

	assert.NoError(t, errQuery, "Can't build Query")

	dest := env.Config().Destination()

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

	var response *gnmi.SubscribeResponse

	// Subscription has to be spawned into a separate thread as it is blocking.
	go func() {
		errSubscribe := subC.Subscribe(testutils.MakeContext(), q, "gnmi")
		assert.NoError(t, errSubscribe, "Subscription Error")
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

func validateGnmiStateResponse(t *testing.T, resp *gnmi.SubscribeResponse, device string) {
	t.Helper()

	switch v := resp.Response.(type) {
	case *gnmi.SubscribeResponse_Update:
		validateGnmiStateUpdateResponse(t, v, device)

	case *gnmi.SubscribeResponse_SyncResponse:
		validateGnmiStateSyncResponse(t, v)

	default:
		assert.Fail(t, "Unknown GNMI state response type")
	}
}

func validateGnmiStateUpdateResponse(t *testing.T, update *gnmi.SubscribeResponse_Update, device string) {
	t.Helper()

	assert.True(t, update.Update != nil, "Update should not be nil")
	assert.Equal(t, 1, len(update.Update.GetUpdate()))
	assert.NotNil(t, update.Update.GetUpdate()[0])

	pathResponse := update.Update.GetUpdate()[0].Path
	assert.Equal(t, pathResponse.Target, device)
	assert.Equal(t, 3, len(pathResponse.Elem), "Expected 3 path elements")
	assert.Equal(t, pathResponse.Elem[0].Name, "system")
	assert.Equal(t, pathResponse.Elem[1].Name, "state")
	assert.Equal(t, pathResponse.Elem[2].Name, "current-datetime")
	updatedTimeString := update.Update.GetUpdate()[0].Val.GetStringVal()
	updatedTime, timeParseError := time.Parse("2006-01-02T15:04:05Z-07:00", updatedTimeString)
	assert.NoError(t, timeParseError)

	assert.True(t, previousTime.Before(updatedTime), "Path time value is not in the future")
	previousTime = updatedTime
}

func validateGnmiStateSyncResponse(t *testing.T, sync *gnmi.SubscribeResponse_SyncResponse) {
	t.Helper()
	assert.True(t, sync.SyncResponse)
}
