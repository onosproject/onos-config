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

package config

import (
	"github.com/Pallinder/go-randomdata"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func generateTimezoneName() string {

	usCity := randomdata.ProvinceForCountry("US")
	timeZone := "US/" + usCity
	return timeZone
}

// TestMultipleSet tests multiple query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestMultipleSet(t *testing.T) {
	generateTimezoneName()

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.GetGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)
	targetClient := gnmiutils.GetTargetGNMIClientOrFail(ctx, t, simulator)

	for i := 0; i < 10; i++ {

		msValue := generateTimezoneName()

		// Set a value using gNMI client
		targetPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), tzPath, msValue, proto.StringVal)
		transactionID, transactionIndex := gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, targetPath, gnmiutils.NoPaths, gnmiutils.SyncExtension(t))
		assert.NotNil(t, transactionID, transactionIndex)

		// Check that the value was set correctly, both in onos-config and the target
		gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, targetPath, msValue, 0, "Query after set returned the wrong value")
		gnmiutils.CheckTargetValue(ctx, t, targetClient, targetPath, msValue)

		// Remove the path we added
		gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, gnmiutils.NoPaths, targetPath, gnmiutils.SyncExtension(t))

		//  Make sure it got removed, both from onos-config and the target
		gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, targetPath, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")
		gnmiutils.CheckTargetValueDeleted(ctx, t, targetClient, targetPath)
	}
}
