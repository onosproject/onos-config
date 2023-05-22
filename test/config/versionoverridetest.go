// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/test"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
)

// TestVersionOverride tests using version override extension
func (s *TestSuite) TestVersionOverride() {
	const (
		value1 = "test-motd-banner"
		path1  = "/system/config/motd-banner"
	)

	var (
		paths  = []string{path1}
		values = []string{value1}
	)

	// Change topo entity for the simulator to non-existent type/version
	err := s.UpdateTargetTypeVersion(topo.ID(s.simulator1.Name), "testdevice", "1.0.0")
	s.NoError(err)

	// Wait for config to connect to the first simulator
	s.WaitForTargetAvailable(topo.ID(s.simulator1.Name))

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Set up paths for the set
	targets := []string{s.simulator1.Name}

	targetPaths := gnmiutils.GetTargetPathsWithValues(targets, paths, values)
	targetPathsForGet := gnmiutils.GetTargetPaths(targets, paths)

	// Formulate extensions with sync and with target version override
	extensions := s.SyncExtension()
	ttv, err := utils.TargetVersionOverrideExtension(configapi.TargetID(s.simulator1.Name), "devicesim", "1.0.x")
	s.NoError(err)
	extensions = append(extensions, ttv)

	var setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		Extensions:  extensions,
		UpdatePaths: targetPaths,
	}
	setReq.SetOrFail(s.T())

	// Make sure the configuration has been applied to the target
	targetGnmiClient := s.NewSimulatorGNMIClientOrFail(s.simulator1.Name)

	var targetGetReq = &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     targetGnmiClient,
		Encoding:   gnmiapi.Encoding_JSON,
		Extensions: s.SyncExtension(),
	}
	targetGetReq.Paths = targetPathsForGet
	targetGetReq.CheckValues(s.T(), value1)

	// Make sure the configuration has been received by onos-config
	var getReq = &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Encoding:   gnmiapi.Encoding_PROTO,
		Extensions: extensions,
	}
	getReq.Paths = targetPathsForGet
	getReq.CheckValues(s.T(), value1)
}
