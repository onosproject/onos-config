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

package ha

import (
	"github.com/onosproject/onos-api/go/onos/topo"
	"testing"
	"time"

	"github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	hautils "github.com/onosproject/onos-config/test/utils/ha"
	"github.com/stretchr/testify/assert"
)

// TestSessionFailOver tests gnmi session failover is happening when the master node
// is crashed
func (s *TestSuite) TestSessionFailOver(t *testing.T) {
	// See https://jira.opennetworking.org/browse/SDRAN-542 for info about why this test is skipped
	t.Skip()
	simulator := gnmi.CreateSimulator(t)
	assert.NotNil(t, simulator)
	var currentTerm uint64
	var masterNode string
	found := gnmi.WaitForTarget(t, func(d *device.Device, eventType topo.EventType) bool {
		currentTerm = d.MastershipTerm
		masterNode = d.MasterKey
		return currentTerm == 1 && len(d.Protocols) > 0 &&
			d.Protocols[0].Protocol == topo.Protocol_GNMI &&
			d.Protocols[0].ConnectivityState == topo.ConnectivityState_REACHABLE &&
			d.Protocols[0].ChannelState == topo.ChannelState_CONNECTED &&
			d.Protocols[0].ServiceState == topo.ServiceState_AVAILABLE
	}, 5*time.Second)
	assert.Equal(t, true, found)
	assert.Equal(t, currentTerm, 1)
	masterPod := hautils.FindPodWithPrefix(t, masterNode)
	// Crash master pod
	hautils.CrashPodOrFail(t, masterPod)

	// Waits for a new master to be elected (i.e. the term will be increased), it establishes a connection to the device
	// and updates the device state
	found = gnmi.WaitForTarget(t, func(d *device.Device, t topo.EventType) bool {
		currentTerm = d.MastershipTerm
		return currentTerm == 2 && len(d.Protocols) > 0 &&
			d.Protocols[0].Protocol == topo.Protocol_GNMI &&
			d.Protocols[0].ConnectivityState == topo.ConnectivityState_REACHABLE &&
			d.Protocols[0].ChannelState == topo.ChannelState_CONNECTED &&
			d.Protocols[0].ServiceState == topo.ServiceState_AVAILABLE
	}, 70*time.Second)
	assert.Equal(t, true, found)

}
