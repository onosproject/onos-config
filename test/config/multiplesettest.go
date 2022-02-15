// Copyright 2020-present Open Networking Foundation.
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
	"github.com/Pallinder/go-randomdata"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gbp "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func generateTimezoneName() string {

	usCity := randomdata.ProvinceForCountry("US")
	timeZone := "US/" + usCity
	return timeZone
}

// TestMultipleSet tests multiple query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestMultipleSet(t *testing.T) {
	generateTimezoneName()

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)
	targetClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)

	for i := 0; i < 10; i++ {

		msValue := generateTimezoneName()

		// Set a value using gNMI client
		targetPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), tzPath, msValue, proto.StringVal)
		var setReq = &gnmiutils.SetRequest{
			Ctx:         ctx,
			Client:      gnmiClient,
			Extensions:  gnmiutils.SyncExtension(t),
			Encoding:    gbp.Encoding_PROTO,
			UpdatePaths: targetPath,
		}
		transactionID, transactionIndex := setReq.SetOrFail(t)
		assert.NotNil(t, transactionID)
		assert.NotNil(t, transactionIndex)

		// Check that the value was set correctly, both in onos-config and the target
		var getConfigReq = &gnmiutils.GetRequest{
			Ctx:        ctx,
			Client:     gnmiClient,
			Paths:      targetPath,
			Extensions: gnmiutils.SyncExtension(t),
			Encoding:   gbp.Encoding_PROTO,
		}
		getConfigReq.CheckValue(t, msValue, 0, "Query after set returned the wrong value")
		var getTargetReq = &gnmiutils.GetRequest{
			Ctx:      ctx,
			Client:   targetClient,
			Encoding: gbp.Encoding_JSON,
			Paths:    targetPath,
		}
		getTargetReq.CheckValue(t, msValue, 0, "get from target incorrect")

		// Remove the path we added
		setReq.UpdatePaths = nil
		setReq.DeletePaths = targetPath
		setReq.SetOrFail(t)

		//  Make sure it got removed, both from onos-config and the target
		getConfigReq.CheckValue(t, "", 0, "incorrect value found for path /system/clock/config/timezone-name after delete")
		getTargetReq.CheckValueDeleted(t)
	}
}
