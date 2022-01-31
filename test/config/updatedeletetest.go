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

package config

import (
	"testing"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/stretchr/testify/assert"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
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
	// Get the first configured target from the environment.
	target := gnmiutils.CreateSimulator(t)
	defer gnmiutils.DeleteSimulator(t, target)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.GetGNMIClientOrFail(t)

	// Create interface tree using gNMI client
	setNamePath := []proto.TargetPath{
		{TargetName: target.Name(), Path: udtestNamePath, PathDataValue: udtestNameValue, PathDataType: proto.StringVal},
	}
	gnmiutils.SetGNMIValueOrFail(t, gnmiClient, setNamePath, gnmiutils.NoPaths, gnmiutils.NoExtensions)

	err := gnmiutils.WaitForConfigurationCompleteOrFail(t, configapi.ConfigurationID(target.Name()), time.Minute)
	assert.NoError(t, err)

	gnmiutils.CheckGNMIValue(t, gnmiClient, setNamePath, udtestNameValue, 0, "Query name after set returned the wrong value")

	// Set initial values for Enabled and Description using gNMI client
	setInitialValuesPath := []proto.TargetPath{
		{TargetName: target.Name(), Path: udtestEnabledPath, PathDataValue: "true", PathDataType: proto.BoolVal},
		{TargetName: target.Name(), Path: udtestDescriptionPath, PathDataValue: udtestDescriptionValue, PathDataType: proto.StringVal},
	}
	gnmiutils.SetGNMIValueOrFail(t, gnmiClient, setInitialValuesPath, gnmiutils.NoPaths, gnmiutils.NoExtensions)

	err = gnmiutils.WaitForConfigurationCompleteOrFail(t, configapi.ConfigurationID(target.Name()), time.Minute)
	assert.NoError(t, err)

	// Update Enabled, delete Description using gNMI client
	updateEnabledPath := []proto.TargetPath{
		{TargetName: target.Name(), Path: udtestEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	deleteDescriptionPath := []proto.TargetPath{
		{TargetName: target.Name(), Path: udtestDescriptionPath},
	}
	gnmiutils.SetGNMIValueOrFail(t, gnmiClient, updateEnabledPath, deleteDescriptionPath, gnmiutils.NoExtensions)

	err = gnmiutils.WaitForConfigurationCompleteOrFail(t, configapi.ConfigurationID(target.Name()), time.Minute)
	assert.NoError(t, err)
	// Check that the Enabled value is set correctly
	gnmiutils.CheckGNMIValue(t, gnmiClient, updateEnabledPath, "false", 0, "Query name after set returned the wrong value")

	//  Make sure Description got removed
	gnmiutils.CheckGNMIValue(t, gnmiClient, gnmiutils.GetTargetPath(target.Name(), udtestDescriptionPath), "", 0, "New child was not removed")
}
