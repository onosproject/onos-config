// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"github.com/Pallinder/go-randomdata"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"time"
)

func generateTimezoneName() string {
	usCity := randomdata.ProvinceForCountry("US")
	timeZone := "US/" + usCity
	return timeZone
}

func (s *TestSuite) testMultipleSet(ctx context.Context, encoding gnmiapi.Encoding) {
	generateTimezoneName()

	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(ctx, topoapi.ID(s.simulator1), 1*time.Minute)
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.NoRetry)
	targetClient := s.NewSimulatorGNMIClientOrFail(ctx, s.simulator1)

	for i := 0; i < 10; i++ {

		msValue := generateTimezoneName()

		// Set a value using gNMI client
		targetPath := gnmiutils.GetTargetPathWithValue(s.simulator1, tzPath, msValue, proto.StringVal)
		var setReq = &gnmiutils.SetRequest{
			Ctx:         ctx,
			Client:      gnmiClient,
			Extensions:  s.SyncExtension(),
			Encoding:    gnmiapi.Encoding_PROTO,
			UpdatePaths: targetPath,
		}
		transactionID, transactionIndex := setReq.SetOrFail(s.T())
		s.NotNil(transactionID)
		s.NotNil(transactionIndex)

		// Check that the value was set correctly, both in onos-config and the target
		var getConfigReq = &gnmiutils.GetRequest{
			Ctx:        ctx,
			Client:     gnmiClient,
			Paths:      targetPath,
			Extensions: s.SyncExtension(),
			Encoding:   encoding,
		}
		getConfigReq.CheckValues(s.T(), msValue)
		var getTargetReq = &gnmiutils.GetRequest{
			Ctx:      ctx,
			Client:   targetClient,
			Encoding: gnmiapi.Encoding_JSON,
			Paths:    targetPath,
		}
		getTargetReq.CheckValues(s.T(), msValue)

		// Remove the path we added
		setReq.UpdatePaths = nil
		setReq.DeletePaths = targetPath
		setReq.SetOrFail(s.T())

		//  Make sure it got removed, both from onos-config and the target
		getConfigReq.CheckValuesDeleted(s.T())
		getTargetReq.CheckValuesDeleted(s.T())
	}
}

// TestMultipleSet tests multiple query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestMultipleSet(ctx context.Context) {
	s.Run("TestMultipleSet PROTO", func() {
		s.testMultipleSet(ctx, gnmiapi.Encoding_PROTO)
	})
	s.Run("TestMultipleSet JSON", func() {
		s.testMultipleSet(ctx, gnmiapi.Encoding_JSON)
	})
}
