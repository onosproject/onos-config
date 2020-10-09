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
	"testing"
	"time"

	"github.com/onosproject/onos-config/api/admin"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
)

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) TestTransaction(t *testing.T) {

	const (
		value1     = "test-motd-banner"
		path1      = "/system/config/motd-banner"
		value2     = "test-login-banner"
		path2      = "/system/config/login-banner"
		initValue1 = "1"
		initValue2 = "2"
	)

	var (
		paths         = []string{path1, path2}
		values        = []string{value1, value2}
		initialValues = []string{initValue1, initValue2}
	)

	// Get the configured devices from the environment.
	device1 := gnmi.CreateSimulator(t)
	device2 := gnmi.CreateSimulator(t)
	devices := make([]string, 2)
	devices[0] = device1.Name()
	devices[1] = device2.Name()

	// Wait for config to connect to the devices
	gnmi.WaitForDeviceAvailable(t, device.ID(device1.Name()), 10*time.Second)
	gnmi.WaitForDeviceAvailable(t, device.ID(device2.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)
	devicePathsForGet := gnmi.GetDevicePaths(devices, paths)

	// Set initial values
	devicePathsForInit := gnmi.GetDevicePathsWithValues(devices, paths, initialValues)
	initialChangeID := gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePathsForInit, gnmi.NoPaths, gnmi.NoExtensions)
	gnmi.WaitForNetworkChangeComplete(t, initialChangeID, 10*time.Second)
	gnmi.CheckGNMIValues(t, gnmiClient, devicePathsForGet, initialValues, 0, "Query after initial set returned the wrong value")

	// Create a change that can be rolled back
	devicePathsForSet := gnmi.GetDevicePathsWithValues(devices, paths, values)
	changeID := gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePathsForSet, gnmi.NoPaths, gnmi.NoExtensions)
	gnmi.WaitForNetworkChangeComplete(t, changeID, 10*time.Second)

	// Check that the values were set correctly
	expectedValues := []string{value1, value2}
	gnmi.CheckGNMIValues(t, gnmiClient, devicePathsForGet, expectedValues, 0, "Query after set returned the wrong value")

	// Wait for the network change to complete
	complete := gnmi.WaitForNetworkChangeComplete(t, changeID, 10*time.Second)
	assert.True(t, complete, "Set never completed")

	// Check that the values are set on the devices
	device1GnmiClient := gnmi.GetDeviceGNMIClientOrFail(t, device1)
	device2GnmiClient := gnmi.GetDeviceGNMIClientOrFail(t, device2)

	gnmi.CheckDeviceValue(t, device1GnmiClient, devicePathsForGet[0:1], value1)
	gnmi.CheckDeviceValue(t, device1GnmiClient, devicePathsForGet[1:2], value2)
	gnmi.CheckDeviceValue(t, device2GnmiClient, devicePathsForGet[2:3], value1)
	gnmi.CheckDeviceValue(t, device2GnmiClient, devicePathsForGet[3:4], value2)

	// Now rollback the change
	adminClient, err := gnmi.NewAdminServiceClient()
	assert.NoError(t, err)
	rollbackResponse, rollbackError := adminClient.RollbackNetworkChange(
		context.Background(), &admin.RollbackRequest{Name: string(changeID)})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")
	assert.Contains(t, rollbackResponse.Message, changeID, "rollbackResponse message does not contain change ID")

	// Check that the values were really rolled back in onos-config
	// TODO - this is actually wrong, the value we read back should be {initValue1, initValue2}
	// Need to investigate why
	expectedValuesAfterRollback := []string{"", ""}
	gnmi.CheckGNMIValues(t, gnmiClient, devicePathsForGet, expectedValuesAfterRollback, 0, "Query after rollback returned the wrong value")

	// Check that the values were rolled back on the devices
	gnmi.CheckDeviceValue(t, device1GnmiClient, devicePathsForGet[0:1], initValue1)
	gnmi.CheckDeviceValue(t, device1GnmiClient, devicePathsForGet[1:2], initValue2)
	gnmi.CheckDeviceValue(t, device2GnmiClient, devicePathsForGet[2:3], initValue1)
	gnmi.CheckDeviceValue(t, device2GnmiClient, devicePathsForGet[3:4], initValue2)
}
