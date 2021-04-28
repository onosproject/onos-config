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
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestDeviceState tests that a device is connected and available.
func (s *TestSuite) TestDeviceState(t *testing.T) {
	simulator := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator)

	assert.NotNil(t, simulator)
	found := gnmi.WaitForDevice(t, func(d *device.Device, eventType topo.EventType) bool {
		return len(d.Protocols) > 0 &&
			d.Protocols[0].Protocol == topo.Protocol_GNMI &&
			d.Protocols[0].ConnectivityState == topo.ConnectivityState_REACHABLE &&
			d.Protocols[0].ChannelState == topo.ChannelState_CONNECTED &&
			d.Protocols[0].ServiceState == topo.ServiceState_AVAILABLE
	}, 5*time.Second)
	assert.Equal(t, true, found)
}
