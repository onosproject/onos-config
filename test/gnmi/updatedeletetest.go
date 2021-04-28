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
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"testing"
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
	device := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, device)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Create interface tree using gNMI client
	setNamePath := []proto.DevicePath{
		{DeviceName: device.Name(), Path: udtestNamePath, PathDataValue: udtestNameValue, PathDataType: proto.StringVal},
	}
	gnmi.SetGNMIValueOrFail(t, gnmiClient, setNamePath, gnmi.NoPaths, gnmi.NoExtensions)

	// Set initial values for Enabled and Description using gNMI client
	setInitialValuesPath := []proto.DevicePath{
		{DeviceName: device.Name(), Path: udtestEnabledPath, PathDataValue: "true", PathDataType: proto.BoolVal},
		{DeviceName: device.Name(), Path: udtestDescriptionPath, PathDataValue: udtestDescriptionValue, PathDataType: proto.StringVal},
	}
	gnmi.SetGNMIValueOrFail(t, gnmiClient, setInitialValuesPath, gnmi.NoPaths, gnmi.NoExtensions)

	// Update Enabled, delete Description using gNMI client
	updateEnabledPath := []proto.DevicePath{
		{DeviceName: device.Name(), Path: udtestEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	deleteDescriptionPath := []proto.DevicePath{
		{DeviceName: device.Name(), Path: udtestDescriptionPath},
	}
	gnmi.SetGNMIValueOrFail(t, gnmiClient, updateEnabledPath, deleteDescriptionPath, gnmi.NoExtensions)

	// Check that the Enabled value is set correctly
	gnmi.CheckGNMIValue(t, gnmiClient, updateEnabledPath, "false", 0, "Query name after set returned the wrong value")

	//  Make sure Description got removed
	gnmi.CheckGNMIValue(t, gnmiClient, gnmi.GetDevicePath(device.Name(), udtestDescriptionPath), "", 0, "New child was not removed")
}
