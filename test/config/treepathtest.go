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
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"testing"
)

const (
	newRootName            = "new-root"
	newRootPath            = "/interfaces/interface[name=" + newRootName + "]"
	newRootConfigNamePath  = newRootPath + "/config/name"
	newRootEnabledPath     = newRootPath + "/config/enabled"
	newRootDescriptionPath = newRootPath + "/config/description"
	newDescription         = "description"
)

// TestTreePath tests create/set/delete of a tree of GNMI paths to a single device
func (s *TestSuite) TestTreePath(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Make a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	getPath := gnmiutils.GetTargetPath(simulator.Name(), newRootEnabledPath)

	// Set name of new root using gNMI client
	setNamePath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: newRootConfigNamePath, PathDataValue: newRootName, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(t)

	// Set values using gNMI client
	setPath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: newRootDescriptionPath, PathDataValue: newDescription, PathDataType: proto.StringVal},
		{TargetName: simulator.Name(), Path: newRootEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	setReq.UpdatePaths = setPath
	setReq.SetOrFail(t)

	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}

	// Check that the name value was set correctly
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValue(t, newRootName)

	// Check that the enabled value was set correctly
	getConfigReq.Paths = getPath
	getConfigReq.CheckValue(t, "false")

	// Remove the root path we added
	setReq.UpdatePaths = nil
	setReq.DeletePaths = getPath
	setReq.SetOrFail(t)

	//  Make sure child got removed
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValue(t, newRootName)

	//  Make sure new root got removed
	getConfigReq.Paths = getPath
	getConfigReq.CheckValue(t, "")
}
