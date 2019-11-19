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
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestDeviceState tests that a device is connected and available.
func (s *TestSuite) TestDeviceState(t *testing.T) {
	simulator := env.NewSimulator().AddOrDie()

	conn, err := env.Topo().Connect()
	assert.NoError(t, err)
	client := device.NewDeviceServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, errGet := client.Get(ctx, &device.GetRequest{
		ID: device.ID(simulator.Name()),
	})
	assert.NoError(t, errGet)
	responseDevice := response.Device
	assert.Equal(t, responseDevice.ID, device.ID(simulator.Name()), "Wrong Device")
	assert.Equal(t, responseDevice.Protocols[0].Protocol, device.Protocol_GNMI)
	assert.Equal(t, responseDevice.Protocols[0].ConnectivityState, device.ConnectivityState_REACHABLE)
	assert.Equal(t, responseDevice.Protocols[0].ChannelState, device.ChannelState_CONNECTED)
	assert.Equal(t, responseDevice.Protocols[0].ServiceState, device.ServiceState_AVAILABLE)
}
