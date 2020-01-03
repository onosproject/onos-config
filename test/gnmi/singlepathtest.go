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
	"github.com/onosproject/onos-test/pkg/onit/env"
	"testing"
)

const (
	tzValue = "Europe/Dublin"
	tzPath  = "/system/clock/config/timezone-name"
)

// TestSinglePath tests query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestSinglePath(t *testing.T) {
	simulator := env.NewSimulator().AddOrDie()

	// Make a GNMI client to use for requests
	gnmiClient := getGNMIClientOrFail(t)

	devicePath := getDevicePathWithValue(simulator.Name(), tzPath, tzValue, StringVal)

	// Set a value using gNMI client
	setGNMIValueOrFail(t, gnmiClient, devicePath, noPaths, noExtensions)

	// Check that the value was set correctly
	checkGNMIValue(t, gnmiClient, devicePath, tzValue, 0, "Query after set returned the wrong value")

	// Remove the path we added
	setGNMIValueOrFail(t, gnmiClient, noPaths, devicePath, noExtensions)

	//  Make sure it got removed
	checkGNMIValue(t, gnmiClient, devicePath, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")
}
