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
	tzValue = "Europe/Dublin"
	tzPath  = "/system/clock/config/timezone-name"
)

// TestSinglePath tests query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestSinglePath(t *testing.T) {
	simulator := env.NewSimulator().AddOrDie()

	// Make a GNMI client to use for requests
	gnmiClient := getGNMIClientOrFail(t)

	// Set a value using gNMI client
	devicePath := getDevicePathWithValue(simulator.Name(), tzPath, tzValue, StringVal)

	_, extensions, errorSet := gNMISet(testutils.MakeContext(), gnmiClient, devicePath, noPaths, noExtensions)
	assert.NoError(t, errorSet)
	assert.Equal(t, 1, len(extensions))
	extension := extensions[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(100))

	// Check that the value was set correctly
	valueAfter, extensions, errorAfter := gNMIGet(testutils.MakeContext(), gnmiClient, devicePath)
	assert.NoError(t, errorAfter)
	assert.Equal(t, 0, len(extensions))
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, tzValue, valueAfter[0].pathDataValue, "Query after set returned the wrong value: %s\n", valueAfter)

	// Remove the path we added
	_, extensions, errorDelete := gNMISet(testutils.MakeContext(), gnmiClient, noPaths, devicePath, noExtensions)
	assert.NoError(t, errorDelete)
	assert.Equal(t, 1, len(extensions))
	extension = extensions[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(100))

	//  Make sure it got removed
	valueAfterDelete, extensions, errorAfterDelete := gNMIGet(testutils.MakeContext(), gnmiClient, devicePath)
	assert.NoError(t, errorAfterDelete)
	assert.Equal(t, 0, len(extensions))
	assert.Equal(t, valueAfterDelete[0].pathDataValue, "",
		"incorrect value found for path /system/clock/config/timezone-name after delete")
}
