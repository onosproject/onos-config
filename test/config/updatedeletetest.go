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
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gbp "github.com/openconfig/gnmi/proto/gnmi"
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
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Get the first configured simulator from the environment.
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Create interface tree using gNMI client
	setNamePath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: udtestNamePath, PathDataValue: udtestNameValue, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gbp.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(t)

	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gbp.Encoding_PROTO,
	}
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValue(t, udtestNameValue, 0, "Query name after set returned the wrong value")

	// Set initial values for Enabled and Description using gNMI client
	setInitialValuesPath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: udtestEnabledPath, PathDataValue: "true", PathDataType: proto.BoolVal},
		{TargetName: simulator.Name(), Path: udtestDescriptionPath, PathDataValue: udtestDescriptionValue, PathDataType: proto.StringVal},
	}
	setReq.UpdatePaths = setInitialValuesPath
	setReq.SetOrFail(t)

	// Update Enabled, delete Description using gNMI client
	updateEnabledPath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: udtestEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	deleteDescriptionPath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: udtestDescriptionPath},
	}
	setReq.UpdatePaths = updateEnabledPath
	setReq.DeletePaths = deleteDescriptionPath
	setReq.SetOrFail(t)

	// Check that the Enabled value is set correctly
	getConfigReq.Paths = updateEnabledPath
	getConfigReq.CheckValue(t, "false", 0, "Query name after set returned the wrong value")

	//  Make sure Description got removed
	getConfigReq.Paths = gnmiutils.GetTargetPath(simulator.Name(), udtestDescriptionPath)
	getConfigReq.CheckValue(t, "", 0, "New child was not removed")
}
