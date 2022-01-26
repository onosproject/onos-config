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

package config

import (
	"github.com/onosproject/onos-api/go/onos/topo"
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
	found := gnmi.WaitForTarget(t, func(rel *topo.Relation, eventType topo.EventType) bool {
		topoClient, err := gnmi.NewTopoClient()
		assert.NoError(t, err)
		d, err := topoClient.Get(gnmi.MakeContext(), rel.TgtEntityID)
		assert.NoError(t, err)
		protocols := &topo.Protocols{}
		err = d.GetAspect(protocols)
		assert.NoError(t, err)
		var protocolStates []*topo.ProtocolState
		if err != nil {
			protocolStates = nil
		} else {
			protocolStates = protocols.State
		}
		return len(protocolStates) > 0 &&
			protocolStates[0].Protocol == topo.Protocol_GNMI &&
			protocolStates[0].ConnectivityState == topo.ConnectivityState_REACHABLE &&
			protocolStates[0].ChannelState == topo.ChannelState_CONNECTED &&
			protocolStates[0].ServiceState == topo.ServiceState_AVAILABLE
	}, 5*time.Second)
	assert.Equal(t, true, found)
}
