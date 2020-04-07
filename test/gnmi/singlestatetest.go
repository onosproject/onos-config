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
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	stateValueRegexp     = `192\.[0-9]+\.[0-9]+\.[0-9]+`
	stateControllersPath = "/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/address"
)

// TestSingleState tests query of a single GNMI path of a read/only value to a single device
func (s *TestSuite) TestSingleState(t *testing.T) {
	// Create a simulated device
	simulator := gnmi.CreateSimulator(t)

	// Wait for config to connect to the device
	gnmi.WaitForDeviceAvailable(t, device.ID(simulator.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Check that the value was correctly retrieved from the device and stored in the state cache
	success := false
	re := regexp.MustCompile(stateValueRegexp)
	for attempt := 1; attempt <= 10; attempt++ {
		// If the device cache has not been completely initialized, we can hit a race here where the value
		// will be returned as null. Needs further investigation.
		valueAfter, extensions, errorAfter := gnmi.GetGNMIValue(gnmi.MakeContext(), gnmiClient, gnmi.GetDevicePath(simulator.Name(), stateControllersPath))
		assert.NoError(t, errorAfter)
		assert.NotEqual(t, nil, valueAfter, "Query after state returned nil")
		address := valueAfter[0].PathDataValue
		if !strings.Contains(address, ".") {
			time.Sleep(time.Second)
			continue
		}
		match := re.MatchString(address)
		assert.True(t, match, "Query for state returned the wrong value: %s\n", valueAfter)
		assert.Equal(t, 0, len(extensions))
		success = true
		break
	}
	assert.Equal(t, success, true, "state value was not found")
}
