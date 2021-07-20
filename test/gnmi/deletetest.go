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
	gbp "github.com/openconfig/gnmi/proto/gnmi"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/stretchr/testify/assert"
)

// TestDelete :
func (s *TestSuite) TestDelete(t *testing.T) {
	t.Skip()
	const (
		oldValue = "old-value"
		newValue = "new-value"
		newPath  = "/system/config/login-banner"
	)

	var (
		newPaths  = []string{newPath}
		newValues = []string{newValue}
		oldValues = []string{oldValue}
	)

	// Get the configured devices from the environment.
	device1 := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, device1)
	devices := make([]string, 1)
	devices[0] = device1.Name()

	// Wait for config to connect to the device
	gnmi.WaitForDeviceAvailable(t, device.ID(device1.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Set values
	var devicePathsForSet = gnmi.GetDevicePathsWithValues(devices, newPaths, newValues)
	changeID := gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePathsForSet, gnmi.NoPaths, gnmi.NoExtensions)

	devicePathsForGet := gnmi.GetDevicePaths(devices, newPaths)

	// Check that the values were set correctly
	expectedValues := []string{newValue}
	gnmi.CheckGNMIValues(t, gnmiClient, devicePathsForGet, expectedValues, 0, "Query after set returned the wrong value")

	// Wait for the network change to complete
	complete := gnmi.WaitForNetworkChangeComplete(t, changeID, 10*time.Second)
	assert.True(t, complete, "Set never completed")

	// Check that the values are set on the devices
	device1GnmiClient := gnmi.GetDeviceGNMIClientOrFail(t, device1)
	gnmi.CheckDeviceValue(t, device1GnmiClient, devicePathsForGet[0:1], newValue)

	// Now rollback the change
	adminClient, err := gnmi.NewAdminServiceClient()
	assert.NoError(t, err)
	rollbackResponse, rollbackError := adminClient.RollbackNetworkChange(
		context.Background(), &admin.RollbackRequest{Name: string(changeID)})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")
	assert.Contains(t, rollbackResponse.Message, changeID, "rollbackResponse message does not contain change ID")

	// Check that the value was really rolled back- should be an error here since the node was deleted
	_, _, err = gnmi.GetGNMIValue(gnmi.MakeContext(), device1GnmiClient, devicePathsForGet, gbp.Encoding_PROTO)
	assert.Error(t, err)
}
