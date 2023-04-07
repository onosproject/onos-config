// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0
//

package config

import (
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
func (s *TestSuite) TestCreatedRemovedTarget() {
	// Wait for config to connect to the target
	s.True(s.WaitForTargetAvailable(topo.ID(s.simulator1.Name)))

	targetPath := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, createRemoveTargetModPath, createRemoveTargetModValue1, proto.StringVal)

	// Set a value using gNMI client - target is up
	c := s.NewOnosConfigGNMIClientOrFail(test.WithRetry)
	var setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      c,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: targetPath,
	}
	setReq.SetOrFail(s.T())

	// Check that the value was set correctly
	var getReq = &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     c,
		Paths:      targetPath,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getReq.CheckValues(s.T(), createRemoveTargetModValue1)

	// interrogate the target to check that the value was set properly
	targetGnmiClient := s.NewSimulatorGNMIClientOrFail(s.simulator1.Name)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetGnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValues(s.T(), createRemoveTargetModValue1)

	//  Shut down the simulator
	s.TearDownSimulator(s.simulator1.Name)
	s.True(s.WaitForTargetUnavailable(topo.ID(s.simulator1.Name)))

	// Set a value using gNMI client - target is down
	setPath2 := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, createRemoveTargetModPath, createRemoveTargetModValue2, proto.StringVal)

	setReq.UpdatePaths = setPath2
	setReq.Extensions = nil
	setReq.SetOrFail(s.T())

	//  Restart simulated target
	s.SetupSimulator(s.simulator1.Name, false)

	// Wait for config to connect to the target
	s.True(s.WaitForTargetAvailable(topo.ID(s.simulator1.Name)))
	// Check that the value was set correctly
	getReq.CheckValues(s.T(), createRemoveTargetModValue2)

	// interrogate the target to check that the value was set properly
	targetGnmiClient2 := s.NewSimulatorGNMIClientOrFail(s.simulator1.Name)
	getTargetReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   targetGnmiClient2,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValues(s.T(), createRemoveTargetModValue2)
}
