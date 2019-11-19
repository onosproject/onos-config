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

	"github.com/stretchr/testify/assert"
)

const (
	udtestRootPath         = "/interfaces/interface[name=test]"
	udtestNamePath         = udtestRootPath + "/config/name"
	udtestEnabledPath      = udtestRootPath + "/config/enabled"
	udtestDescriptionPath  = udtestRootPath + "/config/description"
	udtestNameValue        = "test"
	udtestDescriptionValue = "description"
)

// TestUpdateDelete tests update and delete paths in a single GNMI request
func (s *TestSuite) TestUpdateDelete(t *testing.T) {
	// Get the first configured device from the environment.
	device := env.NewSimulator().AddOrDie()

	// Make a GNMI client to use for requests
	c, err := env.Config().NewGNMIClient()
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Create interface tree using gNMI client
	setNamePath := []DevicePath{
		{deviceName: device.Name(), path: udtestNamePath, pathDataValue: udtestNameValue, pathDataType: StringVal},
	}
	_, _, errorSet := gNMISet(MakeContext(), c, setNamePath, noPaths)
	assert.NoError(t, errorSet)

	// Set initial values for Enabled and Description using gNMI client
	setInitialValuesPath := []DevicePath{
		{deviceName: device.Name(), path: udtestEnabledPath, pathDataValue: "true", pathDataType: BoolVal},
		{deviceName: device.Name(), path: udtestDescriptionPath, pathDataValue: udtestDescriptionValue, pathDataType: StringVal},
	}
	_, _, errorSet = gNMISet(MakeContext(), c, setInitialValuesPath, noPaths)
	assert.NoError(t, errorSet)

	// Update Enabled, delete Description using gNMI client
	updateEnabledPath := []DevicePath{
		{deviceName: device.Name(), path: udtestEnabledPath, pathDataValue: "false", pathDataType: BoolVal},
	}
	deleteDescriptionPath := []DevicePath{
		{deviceName: device.Name(), path: udtestDescriptionPath},
	}
	_, _, errorSet = gNMISet(MakeContext(), c, updateEnabledPath, deleteDescriptionPath)
	assert.NoError(t, errorSet)

	// Check that the Enabled value is set correctly
	valueAfter, extensions, errorAfter := gNMIGet(MakeContext(), c, updateEnabledPath)
	assert.NoError(t, errorAfter)
	assert.Equal(t, 0, len(extensions))
	assert.NotEqual(t, "", valueAfter, "Query name after set returned an error: %s\n", errorAfter)
	assert.Equal(t, "false", valueAfter[0].pathDataValue, "Query name after set returned the wrong value: %s\n", valueAfter)

	//  Make sure Description got removed
	valueAfterDelete, extensions, errorAfterDelete := gNMIGet(MakeContext(), c, makeDevicePath(device.Name(), udtestDescriptionPath))
	assert.NoError(t, errorAfterDelete)
	assert.Equal(t, valueAfterDelete[0].pathDataValue, "", "New child was not removed")
	assert.Equal(t, 0, len(extensions))
}
