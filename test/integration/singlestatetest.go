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

package integration

import (
	"github.com/onosproject/onos-config/test/env"
	"github.com/onosproject/onos-config/test/runner"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	stateValue           = "192.0.2.10"
	stateControllersPath = "/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/address"
)

// TestSingleState tests query of a single GNMI path of a read/only value to a single device
func TestSingleState(t *testing.T) {
	// Get the first configured device from the environment.
	device := env.GetDevices()[0]

	// Make a GNMI client to use for requests
	c, err := env.NewGnmiClient(MakeContext(), "")
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Check that the value was correctly retrieved from the device and store in the state cache
	valueAfter, errorAfter := GNMIGet(MakeContext(), c, makeDevicePath(device, stateControllersPath))
	assert.NoError(t, errorAfter)
	assert.NotEqual(t, "", valueAfter, "Query after state returned an error: %s\n", errorAfter)
	assert.Equal(t, stateValue, valueAfter[0].pathDataValue, "Query for state returned the wrong value: %s\n", valueAfter)
}

func init() {
	Registry.RegisterTest("single-state", TestSingleState, []*runner.TestSuite{AllTests, IntegrationTests})
}
