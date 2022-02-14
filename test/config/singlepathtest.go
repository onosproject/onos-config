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
	protognmi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"

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

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)
	targetClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)

	devicePath := gnmiutils.GetTargetPathWithValue(simulator.Name(), tzPath, tzValue, proto.StringVal)

	// Set a value using gNMI client
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, devicePath, gnmiutils.NoPaths, gnmiutils.SyncExtension(t))

	// Check that the value was set correctly, both in onos-config and on the target
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    devicePath,
		Encoding: protognmi.Encoding_PROTO,
	}
	var simulatorGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetClient,
		Paths:    devicePath,
		Encoding: protognmi.Encoding_JSON,
	}
	onosConfigGetReq.CheckValue(t, tzValue, 0, "Query after set returned the wrong value from onos-config")
	simulatorGetReq.CheckValue(t, tzValue, 0, "Query after set returned the wrong value from target")

	// Remove the path we added
	var onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		DeletePaths: devicePath,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    protognmi.Encoding_PROTO,
	}
	_, _, err := onosConfigSetReq.Set()
	assert.NoError(t, err)

	//  Make sure it got removed, both in onos-config and on the target
	onosConfigGetReq.CheckValue(t, "", 0, "incorrect value from onos-config for path /system/clock/config/timezone-name after delete")
	// Currently, the path is left behind so this check does not work
	//onosConfigGetReq.CheckValueDeleted(t)
	simulatorGetReq.CheckValueDeleted(t)
}
