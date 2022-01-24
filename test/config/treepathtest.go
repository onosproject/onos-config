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
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
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
	// Make a simulated device
	simulator := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	getPath := gnmi.GetTargetPath(simulator.Name(), newRootEnabledPath)

	// Set name of new root using gNMI client
	setNamePath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: newRootConfigNamePath, PathDataValue: newRootName, PathDataType: proto.StringVal},
	}
	gnmi.SetGNMIValueOrFail(t, gnmiClient, setNamePath, gnmi.NoPaths, gnmi.NoExtensions)

	// Set values using gNMI client
	setPath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: newRootDescriptionPath, PathDataValue: newDescription, PathDataType: proto.StringVal},
		{TargetName: simulator.Name(), Path: newRootEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	gnmi.SetGNMIValueOrFail(t, gnmiClient, setPath, gnmi.NoPaths, gnmi.NoExtensions)

	// Check that the name value was set correctly
	gnmi.CheckGNMIValue(t, gnmiClient, setNamePath, newRootName, 0, "Query name after set returned the wrong value")

	// Check that the enabled value was set correctly
	gnmi.CheckGNMIValue(t, gnmiClient, getPath, "false", 0, "Query enabled after set returned the wrong value")

	// Remove the root path we added
	gnmi.SetGNMIValueOrFail(t, gnmiClient, gnmi.NoPaths, getPath, gnmi.NoExtensions)

	//  Make sure child got removed
	gnmi.CheckGNMIValue(t, gnmiClient, setNamePath, newRootName, 0, "New child was not removed")

	//  Make sure new root got removed
	gnmi.CheckGNMIValue(t, gnmiClient, getPath, "", 0, "New root was not removed")
}
