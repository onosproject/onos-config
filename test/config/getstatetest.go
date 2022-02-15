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

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	stateValue = "opennetworking.org"
	statePath  = "/system/state/domain-name"
)

// TestGetState tests query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestGetState(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)

	// Get the GNMI path
	targetPaths := gnmiutils.GetTargetPathWithValue(simulator.Name(), statePath, stateValue, proto.StringVal)

	// Set up requests
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPaths,
		Encoding: gnmiapi.Encoding_JSON,
		DataType: gnmiapi.GetRequest_STATE,
	}

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValue(t, stateValue)

}
