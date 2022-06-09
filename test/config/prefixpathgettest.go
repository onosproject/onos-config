// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

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

// TestPrefixPathGet tests GNMI queries using a prefix + path
func (s *TestSuite) TestPrefixPathGet(t *testing.T) {
	const (
		systemPrefix = "/system"
		clockPrefix  = systemPrefix + "/clock"
		tzPath       = "config/timezone-name"
		fullTzPath   = clockPrefix + "/" + tzPath
		tzValue      = "Europe/Dublin"

		motdPath     = "config/motd-banner"
		fullMotdPath = systemPrefix + "/" + motdPath
		motdValue    = "test-motd-banner"
	)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Wait for config to connect to the targets
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(simulator.Name()), 1*time.Minute)
	assert.True(t, ready)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	gnmiTarget := simulator.Name()

	testCases := []struct {
		name         string
		prefixTarget string
		prefixPath   string
		targets      []string
		paths        []string
		values       []string
	}{
		{
			name:         "Prefix using a target and a path",
			prefixTarget: gnmiTarget,
			prefixPath:   clockPrefix,
			targets:      []string{""},
			paths:        []string{tzPath},
			values:       []string{tzValue},
		},
		{
			name:         "Prefix using a path",
			prefixTarget: "",
			prefixPath:   clockPrefix,
			targets:      []string{gnmiTarget},
			paths:        []string{tzPath},
			values:       []string{tzValue},
		},
		{
			name:         "Prefix specifies the entire path",
			prefixTarget: gnmiTarget,
			prefixPath:   fullTzPath,
			targets:      []string{""},
			paths:        []string{""},
			values:       []string{tzValue},
		},
		{
			name:         "Prefix for multiple paths",
			prefixTarget: gnmiTarget,
			prefixPath:   "/system",
			targets:      []string{""},
			paths:        []string{"clock/config/timezone-name", "config/motd-banner"},
			values:       []string{tzValue, motdValue},
		},
	}

	// Set a new value for the time zone using onos-config
	fullTargetPaths := gnmiutils.GetTargetPathWithValue(simulator.Name(), fullTzPath, tzValue, proto.StringVal)
	var onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: fullTargetPaths,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}
	onosConfigSetReq.SetOrFail(t)

	// set a value for the MOTD using onos-config
	motdFullTargetPaths := gnmiutils.GetTargetPathWithValue(simulator.Name(), fullMotdPath, motdValue, proto.StringVal)
	var motdOnosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		UpdatePaths: motdFullTargetPaths,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
	}
	motdOnosConfigSetReq.SetOrFail(t)

	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		t.Run(testCase.name,
			func(t *testing.T) {
				// Get the GNMI paths
				targetPaths := gnmiutils.GetTargetPaths(testCase.targets, testCase.paths)
				prefixPaths := gnmiutils.GetTargetPath(testCase.prefixTarget, testCase.prefixPath)

				// Set up requests
				var onosConfigGetReq = &gnmiutils.GetRequest{
					Ctx:      ctx,
					Client:   gnmiClient,
					Paths:    targetPaths,
					Prefix:   prefixPaths[0],
					Encoding: gnmiapi.Encoding_PROTO,
				}

				// Check that the value was fetched correctly
				onosConfigGetReq.CheckValues(t, testCase.values...)
			})
	}
}
