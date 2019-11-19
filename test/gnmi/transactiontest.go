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
	"context"
	"github.com/onosproject/onos-config/api/admin"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"strconv"
	"testing"

	"github.com/openconfig/gnmi/client"
	"github.com/stretchr/testify/assert"
)

const (
	value1 = "test-motd-banner"
	path1  = "/system/config/motd-banner"
	value2 = "test-login-banner"
	path2  = "/system/config/login-banner"
)

var (
	paths  = []string{path1, path2}
	values = []string{value1, value2}
)

func getDevicePaths(devices []string, paths []string) []DevicePath {
	var devicePaths = make([]DevicePath, len(paths)*len(devices))
	pathIndex := 0
	for _, device := range devices {
		for _, path := range paths {
			devicePaths[pathIndex].deviceName = device
			devicePaths[pathIndex].path = path
			pathIndex++
		}
	}
	return devicePaths
}

func getDevicePathsWithValues(devices []string, paths []string, values []string) []DevicePath {
	var devicePaths = getDevicePaths(devices, paths)
	valueIndex := 0
	for range devices {
		for _, value := range values {
			devicePaths[valueIndex].pathDataValue = value
			devicePaths[valueIndex].pathDataType = StringVal
			valueIndex++
		}
	}
	return devicePaths
}

func checkDeviceValue(t *testing.T, deviceGnmiClient client.Impl, devicePaths []DevicePath, expectedValue string) {
	deviceValues, extensions, deviceValuesError := gNMIGet(MakeContext(), deviceGnmiClient, devicePaths)
	assert.NoError(t, deviceValuesError, "GNMI get operation to device returned an error")
	assert.Equal(t, expectedValue, deviceValues[0].pathDataValue, "Query after set returned the wrong value: %s\n", expectedValue)
	assert.Equal(t, 0, len(extensions))
}

func getDeviceGNMIClient(t *testing.T, simulator env.SimulatorEnv) client.Impl {
	deviceGnmiClient, deviceGnmiClientError := simulator.NewGNMIClient()
	assert.NoError(t, deviceGnmiClientError)
	assert.True(t, deviceGnmiClient != nil, "Fetching device client returned nil")
	return deviceGnmiClient
}

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) TestTransaction(t *testing.T) {
	// Get the configured devices from the environment.
	device1 := env.NewSimulator().AddOrDie()
	device2 := env.NewSimulator().AddOrDie()
	devices := make([]string, 2)
	devices[0] = device1.Name()
	devices[1] = device2.Name()

	// Make a GNMI client to use for requests
	gnmiClient, gnmiClientError := env.Config().NewGNMIClient()
	assert.NoError(t, gnmiClientError)
	assert.True(t, gnmiClient != nil, "Fetching client returned nil")

	// Set values
	var devicePathsForSet = getDevicePathsWithValues(devices, paths, values)
	changeID, extensions, errorSet := gNMISet(MakeContext(), gnmiClient, devicePathsForSet, noPaths)
	assert.NoError(t, errorSet)
	assert.True(t, changeID != "")
	assert.Equal(t, 1, len(extensions))
	extension := extensions[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(100))

	// Check that the values were set correctly
	var devicePathsForGet = getDevicePaths(devices, paths)
	getValuesAfterSet, extensions, getValueAfterSetError := gNMIGet(MakeContext(), gnmiClient, devicePathsForGet)
	assert.NoError(t, getValueAfterSetError, "GNMI get operation returned an error")
	assert.NotEqual(t, "", getValuesAfterSet, "Query after set returned an error: %s\n", getValueAfterSetError)
	assert.Equal(t, value1, getValuesAfterSet[0].pathDataValue, "Query after set returned the wrong value: %s\n", getValuesAfterSet)
	assert.Equal(t, value2, getValuesAfterSet[1].pathDataValue, "Query after set 2 returned the wrong value: %s\n", getValuesAfterSet)
	assert.Equal(t, 0, len(extensions))

	// Check that the values are set on the devices
	device1GnmiClient := getDeviceGNMIClient(t, device1)
	device2GnmiClient := getDeviceGNMIClient(t, device2)

	checkDeviceValue(t, device1GnmiClient, devicePathsForGet[0:1], value1)
	checkDeviceValue(t, device1GnmiClient, devicePathsForGet[1:2], value2)
	checkDeviceValue(t, device2GnmiClient, devicePathsForGet[2:3], value1)
	checkDeviceValue(t, device2GnmiClient, devicePathsForGet[3:4], value2)

	// Now rollback the change
	adminClient, err := env.Config().NewAdminServiceClient()
	assert.NoError(t, err)
	rollbackResponse, rollbackError := adminClient.RollbackNetworkChange(
		context.Background(), &admin.RollbackRequest{Name: changeID})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")
	assert.Contains(t, rollbackResponse.Message, changeID, "rollbackResponse message does not contain change ID")

	// Check that the values were really rolled back
	getValuesAfterRollback, extensions, errorGetAfterRollback := gNMIGet(MakeContext(), gnmiClient, devicePathsForGet)
	assert.NoError(t, errorGetAfterRollback, "Get after rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for get after rollback is nil")
	assert.Equal(t, "", getValuesAfterRollback[0].pathDataValue, "Query after rollback returned the wrong value: %s\n", getValuesAfterRollback)
	assert.Equal(t, "", getValuesAfterRollback[1].pathDataValue, "Query after rollback returned the wrong value: %s\n", getValuesAfterRollback)
	assert.Equal(t, 0, len(extensions))
}
