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
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	newInterfacesPath      = "/interfaces"
	newRootName            = "new-root"
	newRootPath            = newInterfacesPath + "/interface[name=" + newRootName + "]"
	newConfigPath          = newRootPath + "/config"
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
	//defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	getPath := gnmiutils.GetTargetPath(simulator.Name(), newRootEnabledPath)

	// Set name of new root using gNMI client
	setNamePath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: newRootConfigNamePath, PathDataValue: newRootName, PathDataType: proto.StringVal},
	}
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, setNamePath, gnmiutils.NoPaths, gnmiutils.NoExtensions)

	// Set values using gNMI client
	setPath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: newRootDescriptionPath, PathDataValue: newDescription, PathDataType: proto.StringVal},
		{TargetName: simulator.Name(), Path: newRootEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, setPath, gnmiutils.NoPaths, gnmiutils.SyncExtension(t))

	// Check that the name value was set correctly
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, setNamePath, gnmiutils.NoExtensions, newRootName, 0, "Query name after set returned the wrong value")

	// Check that the enabled value was set correctly
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, getPath, gnmiutils.NoExtensions, "false", 0, "Query enabled after set returned the wrong value")

	// Remove the root path we added
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, gnmiutils.NoPaths, getPath, gnmiutils.SyncExtension(t))

	//  Make sure child got removed
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, setNamePath, gnmiutils.NoExtensions, newRootName, 0, "New child was not removed")

	//  Make sure new root got removed
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, getPath, gnmiutils.NoExtensions, "", 0, "New root was not removed")

	// Make sure path got removed
	interfacesPath := gnmiutils.GetTargetPath(simulator.Name(), newConfigPath)
	intfs, _, err := gnmiutils.GetGNMIValue(ctx, gnmiClient, interfacesPath, gnmiutils.NoExtensions, gpb.Encoding_PROTO)
	assert.NoError(t, err)

	for _, intf := range intfs {
		assert.NotEqual(t, newRootEnabledPath, intf.Path)
	}
}
