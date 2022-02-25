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
	dutestRootPath  = "/interfaces/interface[name=foo]"
	dutestNamePath  = dutestRootPath + "/config/name"
	dutestNameValue = "foo"
)

// TestDeleteUpdate tests update of a path after a previous deletion of a parent path
func (s *TestSuite) TestDeleteUpdate(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Get the first configured simulator from the environment.
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Create interface tree using gNMI client
	setNamePath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: dutestNamePath, PathDataValue: dutestNameValue, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(t)

	// Check the name is there...
	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(t, dutestNameValue)

	deleteAllPath := []proto.TargetPath{
		{TargetName: simulator.Name(), Path: dutestRootPath},
	}
	setReq.UpdatePaths = nil
	setReq.DeletePaths = deleteAllPath
	setReq.SetOrFail(t)

	//  Make sure everything got removed
	getConfigReq.Paths = gnmiutils.GetTargetPath(simulator.Name(), dutestRootPath)
	getConfigReq.CheckValues(t, "")

	// Now recreate the same interface tree....
	setReq.UpdatePaths = setNamePath
	setReq.DeletePaths = nil
	setReq.SetOrFail(t)

	// And check it's there...
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(t, dutestNameValue)
}
