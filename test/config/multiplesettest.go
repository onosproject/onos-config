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
	"testing"

	"github.com/Pallinder/go-randomdata"

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

	for i := 0; i < 10; i++ {

		msValue := generateTimezoneName()

		// Set a value using gNMI client
		targetPath := gnmi.GetTargetPathWithValue(simulator.Name(), tzPath, msValue, proto.StringVal)
		transactionID, transactionIndex := gnmi.SetGNMIValueOrFail(t, gnmiClient, targetPath, gnmi.NoPaths, gnmi.NoExtensions)
		assert.NotNil(t, transactionID, transactionIndex)

		// Check that the value was set correctly
		gnmi.CheckGNMIValue(t, gnmiClient, targetPath, msValue, 0, "Query after set returned the wrong value")

		// Remove the path we added
		gnmi.SetGNMIValueOrFail(t, gnmiClient, gnmi.NoPaths, targetPath, gnmi.NoExtensions)

		//  Make sure it got removed
		gnmi.CheckGNMIValue(t, gnmiClient, targetPath, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")
	}
}
