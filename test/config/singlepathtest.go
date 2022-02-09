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

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
)

const (
	tzValue = "Europe/Dublin"
	tzPath  = "/system/clock/config/timezone-name"
)

// TestSinglePath tests query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestSinglePath(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.GetGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)
	targetClient := gnmiutils.GetTargetGNMIClientOrFail(ctx, t, simulator)

	devicePath := gnmiutils.GetTargetPathWithValue(simulator.Name(), tzPath, tzValue, proto.StringVal)

	// Set a value using gNMI client
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, devicePath, gnmiutils.NoPaths, gnmiutils.SyncExtension(t))

	// Check that the value was set correctly, both in onos-config and on the target
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, devicePath, tzValue, 0, "Query after set returned the wrong value")
	gnmiutils.CheckTargetValue(ctx, t, targetClient, devicePath, tzValue)

	// Remove the path we added
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, gnmiutils.NoPaths, devicePath, gnmiutils.SyncExtension(t))

	//  Make sure it got removed, both in onos-config and on the target
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, devicePath, "", 0,
		"incorrect value found for path /system/clock/config/timezone-name after delete")
	gnmiutils.CheckTargetValueDeleted(ctx, t, targetClient, devicePath)
}
