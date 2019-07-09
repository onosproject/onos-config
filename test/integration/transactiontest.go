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
	"context"
	admin "github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/onosproject/onos-config/test/env"
	"github.com/onosproject/onos-config/test/runner"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	value1 = "Europe/Dublin"
	path1  = "/openconfig-system:system/clock/config/timezone-name"
	value2 = "test-banner"
	path2  = "/openconfig-system:system/config/login-banner"
)

var (
	paths = []string {path1, path2}
	values = []string {value1, value2}
)

func getDevicePaths(device string, paths []string) []DevicePath {
	var devicePaths = make([]DevicePath, len(paths))
	for pathIndex, path := range paths {
		devicePaths[pathIndex].deviceName = device
		devicePaths[pathIndex].path = path
	}
	return devicePaths
}

func getDevicePathsWithValues(device string, paths []string, values[]string) []DevicePath {
	var devicePaths = getDevicePaths(device, paths)
	for valueIndex, value := range values {
		devicePaths[valueIndex].value = value
	}
	return devicePaths
}

// TestTransaction tests setting multiple paths in a single request and rolling it back
func TestTransaction(t *testing.T) {
	// Get the first configured device from the environment.
	device := env.GetDevices()[0]

	// Make a GNMI client to use for requests
	gnmiClient, gnmiClientError := env.NewGnmiClient(MakeContext(), "")
	assert.NoError(t, gnmiClientError)
	assert.True(t, gnmiClient != nil, "Fetching client returned nil")

	// Set values
	var devicePathsForSet = getDevicePathsWithValues(device, paths, values)

	changeID, errorSet := GNMISet(MakeContext(), gnmiClient, devicePathsForSet)
	assert.NoError(t, errorSet)
	assert.True(t, changeID != "")

	var devicePathsForGet = getDevicePaths(device, paths)

	// Check that the values were set correctly
	getValuesAfterSet, getValueAfterSetError := GNMIGet(MakeContext(), gnmiClient, devicePathsForGet)
	assert.NoError(t, getValueAfterSetError, "GNMI set operation returned an error")
	assert.NotEqual(t, "", getValuesAfterSet, "Query after set returned an error: %s\n", getValueAfterSetError)
	assert.Equal(t, value1, getValuesAfterSet[0].value, "Query after set returned the wrong value: %s\n", getValuesAfterSet)
	assert.Equal(t, value2, getValuesAfterSet[1].value, "Query after set 2 returned the wrong value: %s\n", getValuesAfterSet)

	// Now rollback the change
	_, adminClient := env.GetAdminClient()

	rollbackResponse, rollbackError := adminClient.RollbackNetworkChange(
		context.Background(), &admin.RollbackRequest{Name: changeID})

	assert.NoError(t, rollbackError, "Getting admin client returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")
	assert.Contains(t, rollbackResponse.Message, changeID, "rollbackResponse message does not contain change ID")

	// Check that the values were really rolled back
	getValuesAfterRollback, errorGetAfterRollback := GNMIGet(MakeContext(), gnmiClient, devicePathsForGet)
	assert.NoError(t, errorGetAfterRollback, "Get after rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for get after rollback is nil")
	assert.Equal(t, "", getValuesAfterRollback[0].value, "Query after rollback returned the wrong value: %s\n", getValuesAfterRollback)
	assert.Equal(t, "", getValuesAfterRollback[1].value, "Query after rollback returned the wrong value: %s\n", getValuesAfterRollback)
}

func init() {
	Registry.RegisterTest("transaction", TestTransaction, []*runner.TestSuite{AllTests,SomeTests,IntegrationTests})
	//example of registering groups
}
