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
	"testing"
	"time"

	"fmt"

	protobuf "github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/client"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	ocgnmi "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/stretchr/testify/assert"
)

var (
	previousTimeState time.Time
)

// TestSubscribeStateGnmi tests a stream subscription to updates to a device
func (s *TestSuite) TestSubscribeStateGnmi(t *testing.T) {
	const dateTimePath = "/system/state/current-datetime"

	previousTimeState = time.Now().Add(-5 * time.Minute)

	// Bring up a new simulated device
	simulator := gnmi.CreateSimulator(t)
	deviceName := simulator.Name()
	deviceID := device.ID(deviceName)

	// Wait for config to connect to the device
	gnmi.WaitForDeviceAvailable(t, deviceID, time.Minute)

	// Make a GNMI client to use for subscribe
	subC := client.BaseClient{}

	path, err := utils.ParseGNMIElements(utils.SplitPath(dateTimePath))

	assert.NoError(t, err, "Unexpected error doing parsing")

	path.Target = simulator.Name()

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
			validateGnmiStateResponse(t, resp, simulator.Name())
			s.mux.Lock()
			updateCount++
			s.mux.Unlock()

		case *gpb.SubscribeResponse_Error:
			return fmt.Errorf("error in response: %s", v)
		case *gpb.SubscribeResponse_SyncResponse:
			s.mux.Lock()
			syncCount++
			s.mux.Unlock()
			validateGnmiStateResponse(t, resp, simulator.Name())

		default:
			return fmt.Errorf("unknown response %T: %s", v, v)
		}

		if updateCount == expectedNumUpdates && syncCount == expectedNumSyncs {
			done <- true
		}
		return nil
	}

	// Subscription has to be spawned into a separate thread as it is blocking.

	go func() {
		_ = subC.Subscribe(gnmi.MakeContext(), *q, "gnmi")
		defer subC.Close()

	}()

	select {
	case <-done:
		gnmi.DeleteSimulator(t, simulator)
	case <-time.After(timeOut * time.Second):
		assert.FailNow(t, "timed out; the request should be processed under %d:", timeOut)
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
		assert.Fail(t, "WidthUnknown GNMI state response type")
	}
}

func validateGnmiStateUpdateResponse(t *testing.T, update *gpb.SubscribeResponse_Update, device string) {
	t.Helper()
	assertUpdateResponse(t, update, device, subDateTimePath, "")
	updatedTimeString := update.Update.GetUpdate()[0].Val.GetStringVal()
	updatedTime, timeParseError := time.Parse("2006-01-02T15:04:05Z-07:00", updatedTimeString)
	assert.NoError(t, timeParseError)

	assert.True(t, !previousTimeState.After(updatedTime), "Path time value is not in the future. Req time %v previous time %v", updatedTime, previousTimeState)
	previousTimeState = updatedTime
}

func validateGnmiStateSyncResponse(t *testing.T, sync *gpb.SubscribeResponse_SyncResponse) {
	t.Helper()
	assertSyncResponse(t, sync)
}
