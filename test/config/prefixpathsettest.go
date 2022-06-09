// SPDX-FileCopyrightText: 2022-present Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2022-present Intel Corporation
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
)

// TestPrefixPathSet tests GNMI updates using a prefix + path
func (s *TestSuite) TestPrefixPathSet(t *testing.T) {
	const (
		systemPrefix = "/system"
		clockPrefix  = systemPrefix + "/clock"
		tzPath       = "config/timezone-name"
		fullTzPath   = clockPrefix + "/" + tzPath

		motdPath     = "config/motd-banner"
		fullMotdPath = systemPrefix + "/" + motdPath
		motdValue    = "test-motd-banner"
	)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	//defer gnmiutils.DeleteSimulator(t, simulator)

	// Wait for config to connect to the targets
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(simulator.Name()), 1*time.Minute)
	assert.True(t, ready)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	gnmiTarget := simulator.Name()

	testCases := []struct {
		name          string
		prefixTarget  string
		prefixPath    string
		targets       []string
		paths         []string
		fullPaths     []string
		values        []string
		expectedError string
	}{
		{
			name:         "Prefix using a target and a path",
			prefixTarget: gnmiTarget,
			prefixPath:   clockPrefix,
			targets:      []string{""},
			paths:        []string{tzPath},
			fullPaths:    []string{fullTzPath},
			values:       []string{generateTimezoneName()},
		},
		{
			name:         "Prefix using a path",
			prefixTarget: "",
			prefixPath:   clockPrefix,
			targets:      []string{gnmiTarget},
			paths:        []string{tzPath},
			fullPaths:    []string{fullTzPath},
			values:       []string{generateTimezoneName()},
		},
		{
			name:          "Prefix specifies the entire path",
			prefixTarget:  gnmiTarget,
			prefixPath:    fullTzPath,
			targets:       []string{""},
			paths:         []string{""},
			fullPaths:     []string{fullTzPath},
			values:        []string{generateTimezoneName()},
			expectedError: "unable to find exact match for RW model path /system/clock/config/timezone-name",
		},
		{
			name:         "Prefix for multiple paths",
			prefixTarget: gnmiTarget,
			prefixPath:   "/system",
			targets:      []string{""},
			paths:        []string{"clock/config/timezone-name", "config/motd-banner"},
			fullPaths:    []string{fullTzPath, fullMotdPath},
			values:       []string{generateTimezoneName(), motdValue},
		},
	}

	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		t.Run(testCase.name,
			func(t *testing.T) {
				// Set the GNMI paths
				targetPaths := gnmiutils.GetTargetPathsWithValues(testCase.targets, testCase.paths, testCase.values)
				prefixPath := gnmiutils.GetTargetPath(testCase.prefixTarget, testCase.prefixPath)[0]

				onosConfigSetReq := &gnmiutils.SetRequest{
					Ctx:         ctx,
					Client:      gnmiClient,
					Prefix:      prefixPath,
					UpdatePaths: targetPaths,
					Extensions:  gnmiutils.SyncExtension(t),
					Encoding:    gnmiapi.Encoding_PROTO,
				}
				_, _, err := onosConfigSetReq.Set()
				if testCase.expectedError == "" {
					assert.NoError(t, err)
				} else {
					assert.Contains(t, err.Error(), testCase.expectedError)
					return
				}

				// Set the requested value with the requested path and prefix
				var onosConfigGetReq = &gnmiutils.GetRequest{
					Ctx:      ctx,
					Client:   gnmiClient,
					Paths:    targetPaths,
					Prefix:   prefixPath,
					Encoding: gnmiapi.Encoding_PROTO,
				}

				// Check that the value was set correctly in onos-config
				onosConfigGetReq.CheckValues(t, testCase.values...)

				// Check that the value was set correctly on the target
				simClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)
				targets := make([]string, 0)
				for range testCase.fullPaths {
					targets = append(targets, simulator.Name())
				}
				simPaths := gnmiutils.GetTargetPaths(targets, testCase.fullPaths)

				simulatorGetReq := &gnmiutils.GetRequest{
					Ctx:      ctx,
					Client:   simClient,
					Paths:    simPaths,
					Encoding: gnmiapi.Encoding_JSON,
				}
				simulatorGetReq.CheckValues(t, testCase.values...)
			})
	}
}
