// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/test"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
)

// TestVersionOverride tests using version override extension
func (s *TestSuite) TestVersionOverride(ctx context.Context) {
	const (
		value1 = "test-motd-banner"
		path1  = "/system/config/motd-banner"
	)

	var (
		paths  = []string{path1}
		values = []string{value1}
	)

	// Change topo entity for the simulator to non-existent type/version
	err := s.UpdateTargetTypeVersion(ctx, topo.ID(s.simulator1), "testdevice", "1.0.0")
	s.NoError(err)

	// Wait for config to connect to the first simulator
	s.WaitForTargetAvailable(ctx, topo.ID(s.simulator1), time.Minute)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.NoRetry)

	// Set up paths for the set
	targets := []string{s.simulator1}

	targetPaths := gnmiutils.GetTargetPathsWithValues(targets, paths, values)
	targetPathsForGet := gnmiutils.GetTargetPaths(targets, paths)

	// Formulate extensions with sync and with target version override
	extensions := s.SyncExtension()
	ttv, err := utils.TargetVersionOverrideExtension(configapi.TargetID(s.simulator1), "devicesim", "1.0.0")
	s.NoError(err)
	extensions = append(extensions, ttv)

	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		Extensions:  extensions,
		UpdatePaths: targetPaths,
	}
	setReq.SetOrFail(s.T())

	// Make sure the configuration has been applied to the target
	targetGnmiClient := s.NewSimulatorGNMIClientOrFail(ctx, s.simulator1)

	var targetGetReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     targetGnmiClient,
		Encoding:   gnmiapi.Encoding_JSON,
		Extensions: s.SyncExtension(),
	}
	targetGetReq.Paths = targetPathsForGet
	targetGetReq.CheckValues(s.T(), value1)

	// Make sure the configuration has been received by onos-config
	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Encoding:   gnmiapi.Encoding_PROTO,
		Extensions: extensions,
	}
	getReq.Paths = targetPathsForGet
	getReq.CheckValues(s.T(), value1)
}
