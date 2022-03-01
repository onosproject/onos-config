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
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
)

const (
	tzValue = "Europe/Dublin"
	tzPath  = "/system/clock/config/timezone-name"
)

// TestSinglePath tests query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestSinglePath(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(simulator.Name()), 1*time.Minute)
	assert.True(t, ready)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)
	targetClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)

	// Get the GNMI path
	targetPaths := gnmiutils.GetTargetPathWithValue(simulator.Name(), tzPath, tzValue, proto.StringVal)

	// Set up requests
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPaths,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	var simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    targetPaths,
		Encoding: gnmiapi.Encoding_JSON,
	}
	var onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: targetPaths,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(t)

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(t, tzValue)
	simulatorGetReq.CheckValues(t, tzValue)

	// Remove the path we added
	onosConfigSetReq.DeletePaths = targetPaths
	onosConfigSetReq.UpdatePaths = nil
	onosConfigSetReq.SetOrFail(t)

	//  Make sure it got removed, both in onos-config and on the target
	onosConfigGetReq.CheckValuesDeleted(t)
	simulatorGetReq.CheckValuesDeleted(t)
}
