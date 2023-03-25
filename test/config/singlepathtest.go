// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/net/context"
)

const (
	tzValue = "Europe/Dublin"
	tzPath  = "/system/clock/config/timezone-name"
)

func (s *TestSuite) testSinglePath(ctx context.Context, encoding gnmiapi.Encoding) {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(ctx, topoapi.ID(s.simulator1))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.NoRetry)
	targetClient := s.NewSimulatorGNMIClientOrFail(ctx, s.simulator1)

	// Get the GNMI path
	targetPaths := gnmiutils.GetTargetPathWithValue(s.simulator1, tzPath, tzValue, proto.StringVal)

	// Set up requests
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Paths:    targetPaths,
		Encoding: encoding,
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
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}

	// Set a new value for the time zone using onos-config
	onosConfigSetReq.SetOrFail(s.T())

	// Check that the value was set correctly, both in onos-config and on the target
	onosConfigGetReq.CheckValues(s.T(), tzValue)
	simulatorGetReq.CheckValues(s.T(), tzValue)

	// Remove the path we added
	onosConfigSetReq.DeletePaths = targetPaths
	onosConfigSetReq.UpdatePaths = nil
	onosConfigSetReq.SetOrFail(s.T())

	//  Make sure it got removed, both in onos-config and on the target
	onosConfigGetReq.CheckValuesDeleted(s.T())
	simulatorGetReq.CheckValuesDeleted(s.T())
}

// TestSinglePath tests query/set/delete of a single GNMI path to a single device
func (s *TestSuite) TestSinglePath(ctx context.Context) {
	s.Run("TestSinglePath PROTO", func() {
		s.testSinglePath(ctx, gnmiapi.Encoding_PROTO)
	})
	s.Run("TestSinglePath JSON", func() {
		s.testSinglePath(ctx, gnmiapi.Encoding_JSON)
	})
}
