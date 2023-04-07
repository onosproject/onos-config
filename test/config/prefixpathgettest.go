// SPDX-FileCopyrightText: 2022-present Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

// TestPrefixPathGet tests GNMI queries using a prefix + path
func (s *TestSuite) TestPrefixPathGet() {
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

	// Wait for config to connect to the targets
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator1.Name))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	gnmiTarget := s.simulator1.Name

	testCases := []struct {
		name         string
		prefixTarget string
		prefixPath   string
		targets      []string
		paths        []string
		values       []string
		encoding     gnmiapi.Encoding
	}{
		{
			name:         "Prefix using a target and a path - PROTO",
			prefixTarget: gnmiTarget,
			prefixPath:   clockPrefix,
			targets:      []string{""},
			paths:        []string{tzPath},
			values:       []string{tzValue},
			encoding:     gnmiapi.Encoding_PROTO,
		},
		{
			name:         "Prefix using a target and a path - JSON",
			prefixTarget: gnmiTarget,
			prefixPath:   clockPrefix,
			targets:      []string{""},
			paths:        []string{tzPath},
			values:       []string{tzValue},
			encoding:     gnmiapi.Encoding_JSON,
		},
		{
			name:         "Prefix using a path - PROTO",
			prefixTarget: "",
			prefixPath:   clockPrefix,
			targets:      []string{gnmiTarget},
			paths:        []string{tzPath},
			values:       []string{tzValue},
			encoding:     gnmiapi.Encoding_PROTO,
		},
		{
			name:         "Prefix using a path - JSON",
			prefixTarget: "",
			prefixPath:   clockPrefix,
			targets:      []string{gnmiTarget},
			paths:        []string{tzPath},
			values:       []string{tzValue},
			encoding:     gnmiapi.Encoding_JSON,
		},
		{
			name:         "Prefix specifies the entire path - PROTO",
			prefixTarget: gnmiTarget,
			prefixPath:   fullTzPath,
			targets:      []string{""},
			paths:        []string{""},
			values:       []string{tzValue},
			encoding:     gnmiapi.Encoding_PROTO,
		},
		{
			name:         "Prefix specifies the entire path - JSON",
			prefixTarget: gnmiTarget,
			prefixPath:   fullTzPath,
			targets:      []string{""},
			paths:        []string{""},
			values:       []string{tzValue},
			encoding:     gnmiapi.Encoding_JSON,
		},
		{
			name:         "Prefix for multiple paths - PROTO",
			prefixTarget: gnmiTarget,
			prefixPath:   "/system",
			targets:      []string{"", ""},
			paths:        []string{"clock/config/timezone-name", "config/motd-banner"},
			values:       []string{tzValue, motdValue},
			encoding:     gnmiapi.Encoding_PROTO,
		},
		{
			name:         "Prefix for multiple paths - JSON",
			prefixTarget: gnmiTarget,
			prefixPath:   "/system",
			targets:      []string{"", ""},
			paths:        []string{"clock/config/timezone-name", "config/motd-banner"},
			values:       []string{tzValue, motdValue},
			encoding:     gnmiapi.Encoding_JSON,
		},
	}

	// Set a new value for the time zone using onos-config
	fullTargetPaths := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, fullTzPath, tzValue, proto.StringVal)
	var onosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: fullTargetPaths,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}
	onosConfigSetReq.SetOrFail(s.T())

	// set a value for the MOTD using onos-config
	motdFullTargetPaths := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, fullMotdPath, motdValue, proto.StringVal)
	var motdOnosConfigSetReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		UpdatePaths: motdFullTargetPaths,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
	}
	motdOnosConfigSetReq.SetOrFail(s.T())

	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		s.Run(testCase.name, func() {
			// Get the GNMI paths
			targetPaths := gnmiutils.GetTargetPaths(testCase.targets, testCase.paths)
			prefixPaths := gnmiutils.GetTargetPath(testCase.prefixTarget, testCase.prefixPath)

			// Set up requests
			var onosConfigGetReq = &gnmiutils.GetRequest{
				Ctx:      s.Context(),
				Client:   gnmiClient,
				Paths:    targetPaths,
				Prefix:   prefixPaths[0],
				Encoding: gnmiapi.Encoding_PROTO,
			}

			// Check that the value was fetched correctly
			onosConfigGetReq.CheckValues(s.T(), testCase.values...)
		})
	}
}
