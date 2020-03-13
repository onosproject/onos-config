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
	"github.com/onosproject/onos-test/pkg/helm"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-test/pkg/util/random"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestDeviceState tests that a device is connected and available.
func (s *TestSuite) TestDeviceState(t *testing.T) {
	simulator := helm.Namespace().
		Chart("/etc/charts/device-simulator").
		Release(random.NewPetName(2))
	err := simulator.Install(true)
	assert.NoError(t, err)
	err = simulator.Await(func(d *device.Device) bool {
		return len(d.Protocols) > 0 &&
			d.Protocols[0].Protocol == device.Protocol_GNMI &&
			d.Protocols[0].ConnectivityState == device.ConnectivityState_REACHABLE &&
			d.Protocols[0].ChannelState == device.ChannelState_CONNECTED &&
			d.Protocols[0].ServiceState == device.ServiceState_AVAILABLE
	}, 5*time.Second)
	assert.NoError(t, err)
}
