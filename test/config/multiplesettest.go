// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/Pallinder/go-randomdata"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(simulator.Name()), 1*time.Minute)
	assert.True(t, ready)

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
			Encoding:    gnmiapi.Encoding_PROTO,
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
			Encoding:   gnmiapi.Encoding_PROTO,
		}
		getConfigReq.CheckValues(t, msValue)
		var getTargetReq = &gnmiutils.GetRequest{
			Ctx:      ctx,
			Client:   targetClient,
			Encoding: gnmiapi.Encoding_JSON,
			Paths:    targetPath,
		}
		getTargetReq.CheckValues(t, msValue)

		// Remove the path we added
		setReq.UpdatePaths = nil
		setReq.DeletePaths = targetPath
		setReq.SetOrFail(t)

		//  Make sure it got removed, both from onos-config and the target
		getConfigReq.CheckValuesDeleted(t)
		getTargetReq.CheckValuesDeleted(t)
	}
}
