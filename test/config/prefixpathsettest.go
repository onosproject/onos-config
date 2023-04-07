// SPDX-FileCopyrightText: 2022-present Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

// TestPrefixPathSet tests GNMI updates using a prefix + path
func (s *TestSuite) TestPrefixPathSet() {
	const (
		systemPrefix = "/system"
		clockPrefix  = systemPrefix + "/clock"
		tzPath       = "config/timezone-name"
		fullTzPath   = clockPrefix + "/" + tzPath

		motdPath     = "config/motd-banner"
		fullMotdPath = systemPrefix + "/" + motdPath
		motdValue    = "test-motd-banner"
	)

	// Wait for config to connect to the targets
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator1.Name))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	gnmiTarget := s.simulator1.Name

	testCases := []struct {
		name          string
		prefixTarget  string
		prefixPath    string
		targets       []string
		paths         []string
		fullPaths     []string
		values        []string
		encoding      gnmiapi.Encoding
		expectedError string
	}{
		{
			name:         "Prefix using a target and a path - PROTO",
			prefixTarget: gnmiTarget,
			prefixPath:   clockPrefix,
			targets:      []string{""},
			paths:        []string{tzPath},
			fullPaths:    []string{fullTzPath},
			values:       []string{generateTimezoneName()},
			encoding:     gnmiapi.Encoding_PROTO,
		},
		{
			name:         "Prefix using a target and a path - JSON",
			prefixTarget: gnmiTarget,
			prefixPath:   clockPrefix,
			targets:      []string{""},
			paths:        []string{tzPath},
			fullPaths:    []string{fullTzPath},
			values:       []string{generateTimezoneName()},
			encoding:     gnmiapi.Encoding_JSON,
		},
		{
			name:         "Prefix using a path - PROTO",
			prefixTarget: "",
			prefixPath:   clockPrefix,
			targets:      []string{gnmiTarget},
			paths:        []string{tzPath},
			fullPaths:    []string{fullTzPath},
			values:       []string{generateTimezoneName()},
			encoding:     gnmiapi.Encoding_PROTO,
		},
		{
			name:         "Prefix using a path - JSON",
			prefixTarget: "",
			prefixPath:   clockPrefix,
			targets:      []string{gnmiTarget},
			paths:        []string{tzPath},
			fullPaths:    []string{fullTzPath},
			values:       []string{generateTimezoneName()},
			encoding:     gnmiapi.Encoding_JSON,
		},
		{
			name:          "Prefix specifies the entire path - PROTO",
			prefixTarget:  gnmiTarget,
			prefixPath:    fullTzPath,
			targets:       []string{""},
			paths:         []string{""},
			fullPaths:     []string{fullTzPath},
			values:        []string{generateTimezoneName()},
			encoding:      gnmiapi.Encoding_PROTO,
			expectedError: "unable to find exact match for RW model path /system/clock/config/timezone-name",
		},
		{
			name:          "Prefix specifies the entire path - JSON",
			prefixTarget:  gnmiTarget,
			prefixPath:    fullTzPath,
			targets:       []string{""},
			paths:         []string{""},
			fullPaths:     []string{fullTzPath},
			values:        []string{generateTimezoneName()},
			encoding:      gnmiapi.Encoding_JSON,
			expectedError: "unable to find exact match for RW model path /system/clock/config/timezone-name",
		},
		{
			name:         "Prefix for multiple paths - PROTO",
			prefixTarget: gnmiTarget,
			prefixPath:   "/system",
			targets:      []string{"", ""},
			paths:        []string{"clock/config/timezone-name", "config/motd-banner"},
			fullPaths:    []string{fullTzPath, fullMotdPath},
			values:       []string{generateTimezoneName(), motdValue},
			encoding:     gnmiapi.Encoding_PROTO,
		},
		{
			name:         "Prefix for multiple paths - JSON",
			prefixTarget: gnmiTarget,
			prefixPath:   "/system",
			targets:      []string{"", ""},
			paths:        []string{"clock/config/timezone-name", "config/motd-banner"},
			fullPaths:    []string{fullTzPath, fullMotdPath},
			values:       []string{generateTimezoneName(), motdValue},
			encoding:     gnmiapi.Encoding_JSON,
		},
	}

	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		s.Run(testCase.name, func() {
			// Set the GNMI paths
			targetPaths := gnmiutils.GetTargetPathsWithValues(testCase.targets, testCase.paths, testCase.values)
			prefixPath := gnmiutils.GetTargetPath(testCase.prefixTarget, testCase.prefixPath)[0]

			onosConfigSetReq := &gnmiutils.SetRequest{
				Ctx:         s.Context(),
				Client:      gnmiClient,
				Prefix:      prefixPath,
				UpdatePaths: targetPaths,
				Extensions:  s.SyncExtension(),
				Encoding:    testCase.encoding,
			}
			_, _, err := onosConfigSetReq.Set()
			if testCase.expectedError == "" {
				s.NoError(err)
			} else {
				s.Contains(err.Error(), testCase.expectedError)
				return
			}

			// Set the requested value with the requested path and prefix
			var onosConfigGetReq = &gnmiutils.GetRequest{
				Ctx:      s.Context(),
				Client:   gnmiClient,
				Paths:    targetPaths,
				Prefix:   prefixPath,
				Encoding: testCase.encoding,
			}

			// Check that the value was set correctly in onos-config
			onosConfigGetReq.CheckValues(s.T(), testCase.values...)

			// Check that the value was set correctly on the target
			simClient := s.NewSimulatorGNMIClientOrFail(s.simulator1.Name)
			targets := make([]string, 0)
			for range testCase.fullPaths {
				targets = append(targets, s.simulator1.Name)
			}
			simPaths := gnmiutils.GetTargetPaths(targets, testCase.fullPaths)

			simulatorGetReq := &gnmiutils.GetRequest{
				Ctx:      s.Context(),
				Client:   simClient,
				Paths:    simPaths,
				Encoding: gnmiapi.Encoding_JSON,
			}
			simulatorGetReq.CheckValues(s.T(), testCase.values...)
		})
	}
}
