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
	"github.com/onosproject/onos-config/api/types/change/network"
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
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

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) TestTransaction(t *testing.T) {
	// Get the configured devices from the environment.
	device1 := env.NewSimulator().AddOrDie()
	device2 := env.NewSimulator().AddOrDie()
	devices := make([]string, 2)
	devices[0] = device1.Name()
	devices[1] = device2.Name()

	// Wait for config to connect to the devices
	testutils.WaitForDeviceAvailable(t, device.ID(device1.Name()), 10*time.Second)
	testutils.WaitForDeviceAvailable(t, device.ID(device2.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := getGNMIClientOrFail(t)

	// Set values
	var devicePathsForSet = getDevicePathsWithValues(devices, paths, values)
	changeID, extensions, errorSet := gNMISet(testutils.MakeContext(), gnmiClient, devicePathsForSet, noPaths, noExtensions)
	assert.NoError(t, errorSet)
	assert.True(t, changeID != "")
	assert.Equal(t, 1, len(extensions))
	extension := extensions[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(100))
	networkChangeID := network.ID(extension.Msg)

	// Check that the values were set correctly
	var devicePathsForGet = getDevicePaths(devices, paths)
	getValuesAfterSet, extensions, getValueAfterSetError := gNMIGet(testutils.MakeContext(), gnmiClient, devicePathsForGet)
	assert.NoError(t, getValueAfterSetError, "GNMI get operation returned an error")
	assert.NotEqual(t, "", getValuesAfterSet, "Query after set returned an error: %s\n", getValueAfterSetError)
	assert.Equal(t, value1, getValuesAfterSet[0].pathDataValue, "Query after set returned the wrong value: %s\n", getValuesAfterSet)
	assert.Equal(t, value2, getValuesAfterSet[1].pathDataValue, "Query after set 2 returned the wrong value: %s\n", getValuesAfterSet)
	assert.Equal(t, 0, len(extensions))

	// Wait for the network change to complete
	complete := testutils.WaitForNetworkChangeComplete(t, networkChangeID)
	assert.True(t, complete, "Set never completed")

	// Check that the values are set on the devices
	device1GnmiClient := getDeviceGNMIClientOrFail(t, device1)
	device2GnmiClient := getDeviceGNMIClientOrFail(t, device2)

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
	getValuesAfterRollback, extensions, errorGetAfterRollback := gNMIGet(testutils.MakeContext(), gnmiClient, devicePathsForGet)
	assert.NoError(t, errorGetAfterRollback, "Get after rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for get after rollback is nil")
	assert.Equal(t, "", getValuesAfterRollback[0].pathDataValue, "Query after rollback returned the wrong value: %s\n", getValuesAfterRollback)
	assert.Equal(t, "", getValuesAfterRollback[1].pathDataValue, "Query after rollback returned the wrong value: %s\n", getValuesAfterRollback)
	assert.Equal(t, 0, len(extensions))
}
