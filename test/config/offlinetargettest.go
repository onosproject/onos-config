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
//

package config

import (
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

const (
	modPath           = "/system/clock/config/timezone-name"
	modValue          = "Europe/Rome"
	offlineTargetName = "test-offline-device-1"
)

// TestOfflineTarget tests set/query of a single GNMI path to a single target that is initially not in the config
func (s *TestSuite) TestOfflineTarget(t *testing.T) {
	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	createOfflineTarget(t, offlineTargetName, "devicesim-1.0.x", "1.0.0", "")

	devicePath := gnmi.GetTargetPathWithValue(offlineTargetName, modPath, modValue, proto.StringVal)
	gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, []*gnmi_ext.Extension{})

	// Bring the target online
	simulator := gnmi.CreateSimulatorWithName(t, offlineTargetName)
	defer gnmi.DeleteSimulator(t, simulator)

	// Wait for config to connect to the target
	gnmi.WaitForTargetAvailable(t, topo.ID(simulator.Name()), time.Minute)
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, modValue, 0, "Query after set returned the wrong value")

	// Check that the value was set properly on the target, wait for configuration gets completed
	targetGnmiClient := gnmi.GetTargetGNMIClientOrFail(t, simulator)
	gnmi.CheckTargetValue(t, targetGnmiClient, devicePath, modValue)
}
