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

package gnmi

import (
	"testing"
	"time"

	"github.com/Pallinder/go-randomdata"

	"github.com/onosproject/onos-api/go/onos/config/change/network"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/stretchr/testify/assert"
)

func generateTimezoneName() string {

	usCity := randomdata.ProvinceForCountry("US")
	timeZone := "US/" + usCity
	return timeZone
}

// TestMultipleSet tests multiple query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestMultipleSet(t *testing.T) {
	generateTimezoneName()

	// Create a simulated device
	simulator := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	var changeIDs []network.ID

	for i := 0; i < 10; i++ {

		msValue := generateTimezoneName()

		devicePath := gnmi.GetTargetPathWithValue(simulator.Name(), tzPath, msValue, proto.StringVal)
		// Set a value using gNMI client
		changeID := gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, gnmi.NoExtensions)

		// Append the changeID to list of changeIDs
		changeIDs = append(changeIDs, changeID)

		// Check that the value was set correctly
		gnmi.CheckGNMIValue(t, gnmiClient, devicePath, msValue, 0, "Query after set returned the wrong value")

		// Remove the path we added
		gnmi.SetGNMIValueOrFail(t, gnmiClient, gnmi.NoPaths, devicePath, gnmi.NoExtensions)

		//  Make sure it got removed
		gnmi.CheckGNMIValue(t, gnmiClient, devicePath, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")

	}

	// Make sure all of the changes have been completed
	for _, changeID := range changeIDs {
		complete := gnmi.WaitForTransactionComplete(t, changeID, 5*time.Second)
		assert.True(t, complete, "Set never completed")
	}
}
