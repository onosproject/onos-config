// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package config

import (
	"context"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	createRemoveTargetModPath   = "/system/clock/config/timezone-name"
	createRemoveTargetModValue1 = "Europe/Paris"
	createRemoveTargetModValue2 = "Europe/London"
)

// TestCreatedRemovedTarget tests set/query of a single GNMI path to a single target that is created, removed, then created again
func (s *TestSuite) TestCreatedRemovedTarget(ctx context.Context) {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(ctx, topo.ID(s.simulator1))
	s.True(ready)

	targetPath := gnmiutils.GetTargetPathWithValue(s.simulator1, createRemoveTargetModPath, createRemoveTargetModValue1, proto.StringVal)

	// Set a value using gNMI client - target is up
	c := s.NewOnosConfigGNMIClientOrFail(ctx, test.WithRetry)
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      c,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: targetPath,
	}
	setReq.SetOrFail(s.T())

	// Check that the value was set correctly
	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     c,
		Paths:      targetPath,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getReq.CheckValues(s.T(), createRemoveTargetModValue1)

	// interrogate the target to check that the value was set properly
	targetGnmiClient := s.NewSimulatorGNMIClientOrFail(ctx, s.simulator1)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetGnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValues(s.T(), createRemoveTargetModValue1)

	//  Shut down the simulator
	s.TearDownSimulator(ctx, s.simulator1)
	unavailable := s.WaitForTargetUnavailable(ctx, topo.ID(s.simulator1))
	s.True(unavailable)

	// Set a value using gNMI client - target is down
	setPath2 := gnmiutils.GetTargetPathWithValue(s.simulator1, createRemoveTargetModPath, createRemoveTargetModValue2, proto.StringVal)

	setReq.UpdatePaths = setPath2
	setReq.Extensions = nil
	setReq.SetOrFail(s.T())

	//  Restart simulated target
	s.SetupSimulator(ctx, s.simulator1, false)

	// Wait for config to connect to the target
	ready = s.WaitForTargetAvailable(ctx, topo.ID(s.simulator1))
	s.True(ready)
	// Check that the value was set correctly
	getReq.CheckValues(s.T(), createRemoveTargetModValue2)

	// interrogate the target to check that the value was set properly
	targetGnmiClient2 := s.NewSimulatorGNMIClientOrFail(ctx, s.simulator1)
	getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetGnmiClient2,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValues(s.T(), createRemoveTargetModValue2)
}
