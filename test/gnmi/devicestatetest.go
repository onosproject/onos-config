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

	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/onosproject/onos-topo/api/topo"
	"github.com/stretchr/testify/assert"
)

// TestDeviceState tests that a device is connected and available.
func (s *TestSuite) TestDeviceState(t *testing.T) {
	simulator := gnmi.CreateSimulator(t)
	assert.NotNil(t, simulator)
	found := gnmi.WaitForDevice(t, func(d *topo.Object) bool {
		return len(d.GetEntity().Protocols) > 0 &&
			d.GetEntity().Protocols[0].Protocol == device.Protocol_GNMI &&
			d.GetEntity().Protocols[0].ConnectivityState == device.ConnectivityState_REACHABLE &&
			d.GetEntity().Protocols[0].ChannelState == device.ChannelState_CONNECTED &&
			d.GetEntity().Protocols[0].ServiceState == device.ServiceState_AVAILABLE
	}, 5*time.Second)
	assert.Equal(t, true, found)
}
