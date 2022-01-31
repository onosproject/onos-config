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
	"testing"

	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
)

const (
	tzValue = "Europe/Dublin"
	tzPath  = "/system/clock/config/timezone-name"
)

// TestSinglePath tests query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestSinglePath(t *testing.T) {
	// Create a simulated device
	simulator := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	devicePath := gnmi.GetTargetPathWithValue(simulator.Name(), tzPath, tzValue, proto.StringVal)

	// Set a value using gNMI client
	gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, gnmi.SyncExtension(t))

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, tzValue, 0, "Query after set returned the wrong value")

	// Remove the path we added
	gnmi.SetGNMIValueOrFail(t, gnmiClient, gnmi.NoPaths, devicePath, gnmi.SyncExtension(t))

	//  Make sure it got removed
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")
}
