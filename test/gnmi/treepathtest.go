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
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	newRootName            = "new-root"
	newRootPath            = "/interfaces/interface[name=" + newRootName + "]"
	newRootConfigNamePath  = newRootPath + "/config/name"
	newRootEnabledPath     = newRootPath + "/config/enabled"
	newRootDescriptionPath = newRootPath + "/config/description"
	newDescription         = "description"
)

// TestTreePath tests create/set/delete of a tree of GNMI paths to a single device
func (s *TestSuite) TestTreePath(t *testing.T) {
	// Get the first configured device from the environment.
	device := env.NewSimulator().AddOrDie()

	// Make a GNMI client to use for requests
	c, err := env.Config().NewGNMIClient()
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Set name of new root using gNMI client
	setNamePath := []DevicePath{
		{deviceName: device.Name(), path: newRootConfigNamePath, pathDataValue: newRootName, pathDataType: StringVal},
	}
	_, _, errorSet := gNMISet(MakeContext(), c, setNamePath, noPaths, noExtensions)
	assert.NoError(t, errorSet)

	// Set values using gNMI client
	setPath := []DevicePath{
		{deviceName: device.Name(), path: newRootDescriptionPath, pathDataValue: newDescription, pathDataType: StringVal},
		{deviceName: device.Name(), path: newRootEnabledPath, pathDataValue: "false", pathDataType: BoolVal},
	}
	_, _, errorSet = gNMISet(MakeContext(), c, setPath, noPaths, noExtensions)
	assert.NoError(t, errorSet)

	// Check that the name value was set correctly
	valueAfter, extensions, errorAfter := gNMIGet(MakeContext(), c, setNamePath)
	assert.NoError(t, errorAfter)
	assert.Equal(t, 0, len(extensions))
	assert.NotEqual(t, "", valueAfter, "Query name after set returned an error: %s\n", errorAfter)
	assert.Equal(t, newRootName, valueAfter[0].pathDataValue, "Query name after set returned the wrong value: %s\n", valueAfter)

	// Check that the enabled value was set correctly
	valueAfter, extensions, errorAfter = gNMIGet(MakeContext(), c, makeDevicePath(device.Name(), newRootEnabledPath))
	assert.NoError(t, errorAfter)
	assert.NotEqual(t, "", valueAfter, "Query enabled after set returned an error: %s\n", errorAfter)
	assert.Equal(t, "false", valueAfter[0].pathDataValue, "Query enabled after set returned the wrong value: %s\n", valueAfter)
	assert.Equal(t, 0, len(extensions))

	// Remove the root path we added
	_, extensions, errorDelete := gNMISet(MakeContext(), c, noPaths, makeDevicePath(device.Name(), newRootPath), noExtensions)
	assert.NoError(t, errorDelete)
	assert.Equal(t, 0, len(extensions))
	extension := extensions[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(100))

	//  Make sure child got removed
	valueAfterDelete, extensions, errorAfterDelete := gNMIGet(MakeContext(), c, makeDevicePath(device.Name(), newRootConfigNamePath))
	assert.NoError(t, errorAfterDelete)
	assert.Equal(t, valueAfterDelete[0].pathDataValue, "", "New child was not removed")
	assert.Equal(t, 0, len(extensions))

	//  Make sure new root got removed
	valueAfterRootDelete, extensions, errorAfterRootDelete := gNMIGet(MakeContext(), c, makeDevicePath(device.Name(), newRootPath))
	assert.NoError(t, errorAfterRootDelete)
	assert.Equal(t, valueAfterRootDelete[0].pathDataValue, "",
		"New root was not removed")
	assert.Equal(t, 0, len(extensions))
}
