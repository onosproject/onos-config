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
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
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
	gnmiClient := getGNMIClientOrFail(t)

	getPath := getDevicePath(device.Name(), newRootEnabledPath)

	// Set name of new root using gNMI client
	setNamePath := []DevicePath{
		{deviceName: device.Name(), path: newRootConfigNamePath, pathDataValue: newRootName, pathDataType: StringVal},
	}
	_, _, errorSet := gNMISet(testutils.MakeContext(), gnmiClient, setNamePath, noPaths, noExtensions)
	assert.NoError(t, errorSet)

	// Set values using gNMI client
	setPath := []DevicePath{
		{deviceName: device.Name(), path: newRootDescriptionPath, pathDataValue: newDescription, pathDataType: StringVal},
		{deviceName: device.Name(), path: newRootEnabledPath, pathDataValue: "false", pathDataType: BoolVal},
	}
	_, _, errorSet = gNMISet(testutils.MakeContext(), gnmiClient, setPath, noPaths, noExtensions)
	assert.NoError(t, errorSet)

	// Check that the name value was set correctly
	checkGnmiValue(t, gnmiClient, setNamePath, newRootName, 0, "Query name after set returned the wrong value")

	// Check that the enabled value was set correctly
	checkGnmiValue(t, gnmiClient, getPath, "false", 0, "Query enabled after set returned the wrong value")

	// Remove the root path we added
	_, extensions, errorDelete := gNMISet(testutils.MakeContext(), gnmiClient, noPaths, getPath, noExtensions)
	assert.NoError(t, errorDelete)
	assert.Equal(t, 1, len(extensions))
	extension := extensions[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(100))

	//  Make sure child got removed
	checkGnmiValue(t, gnmiClient, setNamePath, newRootName, 0, "New child was not removed")

	//  Make sure new root got removed
	checkGnmiValue(t, gnmiClient, getPath, "", 0, "New root was not removed")
}
