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
	"github.com/openconfig/gnmi/client"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	value1 = "test-motd-banner"
	path1  = "/openconfig-system:system/config/motd-banner"
	value2 = "test-login-banner"
	path2  = "/openconfig-system:system/config/login-banner"
)

var (
	paths = []string {path1, path2}
	values = []string {value1, value2}
)

func init() {
	Registry.RegisterTest("transaction", TestTransaction, []*runner.TestSuite{AllTests,SomeTests,IntegrationTests})
}

func getDevicePaths(devices []string, paths []string) []DevicePath {
	var devicePaths = make([]DevicePath, len(paths) * len(devices))
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

func getDevicePathsWithValues(devices []string, paths []string, values[]string) []DevicePath {
	var devicePaths = getDevicePaths(devices, paths)
	valueIndex := 0
	for range devices {
		for _, value := range values {
			devicePaths[valueIndex].value = value
			valueIndex++
		}
	}
	return devicePaths
}

func checkDeviceValue(t *testing.T, deviceGnmiClient client.Impl, devicePaths []DevicePath, expectedValue string) {
	deviceValues, deviceValuesError := GNMIGet(MakeContext(), deviceGnmiClient, devicePaths, StripNamespaces)
	assert.NoError(t, deviceValuesError, "GNMI get operation to device returned an error")
	assert.Equal(t, expectedValue, deviceValues[0].value, "Query after set returned the wrong value: %s\n", expectedValue)
}

func getDeviceGNMIClient(t *testing.T, device string) client.Impl {
	deviceGnmiClient, deviceGnmiClientError := env.NewGnmiClientForDevice(MakeContext(), device + ":10161", "gnmi")
	assert.NoError(t, deviceGnmiClientError)
	assert.True(t, deviceGnmiClient != nil, "Fetching device client returned nil")
	return deviceGnmiClient
}

// TestTransaction tests setting multiple paths in a single request and rolling it back
func TestTransaction(t *testing.T) {
	// Get the configured devices from the environment.
	device1 := env.GetDevices()[0]
	device2 := env.GetDevices()[1]
	devices := make([]string, 2)
	devices[0] = device1
	devices[1] = device2

	// Make a GNMI client to use for requests
	gnmiClient, gnmiClientError := env.NewGnmiClient(MakeContext(), "gnmi")
	assert.NoError(t, gnmiClientError)
	assert.True(t, gnmiClient != nil, "Fetching client returned nil")

	// Set values
	var devicePathsForSet = getDevicePathsWithValues(devices, paths, values)
	changeID, errorSet := GNMISet(MakeContext(), gnmiClient, devicePathsForSet, StripNamespaces)
	assert.NoError(t, errorSet)
	assert.True(t, changeID != "")

	// Check that the values were set correctly
	var devicePathsForGet = getDevicePaths(devices, paths)
	getValuesAfterSet, getValueAfterSetError := GNMIGet(MakeContext(), gnmiClient, devicePathsForGet, StripNamespaces)
	assert.NoError(t, getValueAfterSetError, "GNMI get operation returned an error")
	assert.NotEqual(t, "", getValuesAfterSet, "Query after set returned an error: %s\n", getValueAfterSetError)
	assert.Equal(t, value1, getValuesAfterSet[0].value, "Query after set returned the wrong value: %s\n", getValuesAfterSet)
	assert.Equal(t, value2, getValuesAfterSet[1].value, "Query after set 2 returned the wrong value: %s\n", getValuesAfterSet)

	// Check that the values are set on the devices
	device1GnmiClient := getDeviceGNMIClient(t, device1)
	device2GnmiClient := getDeviceGNMIClient(t, device2)

	checkDeviceValue(t, device1GnmiClient, devicePathsForGet[0:1], value1)
	checkDeviceValue(t, device1GnmiClient, devicePathsForGet[1:2], value2)
	checkDeviceValue(t, device2GnmiClient, devicePathsForGet[2:3], value1)
	checkDeviceValue(t, device2GnmiClient, devicePathsForGet[3:4], value2)

	// Now rollback the change
	_, adminClient := env.GetAdminClient()
	rollbackResponse, rollbackError := adminClient.RollbackNetworkChange(
		context.Background(), &admin.RollbackRequest{Name: changeID})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")
	assert.Contains(t, rollbackResponse.Message, changeID, "rollbackResponse message does not contain change ID")

	// Check that the values were really rolled back
	getValuesAfterRollback, errorGetAfterRollback := GNMIGet(MakeContext(), gnmiClient, devicePathsForGet, StripNamespaces)
	assert.NoError(t, errorGetAfterRollback, "Get after rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for get after rollback is nil")
	assert.Equal(t, "", getValuesAfterRollback[0].value, "Query after rollback returned the wrong value: %s\n", getValuesAfterRollback)
	assert.Equal(t, "", getValuesAfterRollback[1].value, "Query after rollback returned the wrong value: %s\n", getValuesAfterRollback)
}

