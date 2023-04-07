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
	"google.golang.org/grpc/status"
)

// TestModels tests GNMI operation involving unknown or illegal paths
func (s *TestSuite) TestModels() {
	const (
		unknownPath       = "/system/config/no-such-path"
		ntpPath           = "/system/ntp/state/enable-ntp-auth"
		hostNamePath      = "/system/config/hostname"
		clockTimeZonePath = "/system/clock/config/timezone-name"
	)

	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator1.Name))
	s.True(ready)

	// Data to run the test cases
	testCases := []struct {
		description   string
		path          string
		value         string
		valueType     string
		expectedError string
	}{
		{description: "Unknown path", path: unknownPath, valueType: proto.StringVal, value: "123456", expectedError: "no-such-path"},
		{description: "Read only path", path: ntpPath, valueType: proto.BoolVal, value: "false",
			expectedError: "unable to find exact match for RW model path /system/ntp/state/enable-ntp-auth. 113 paths inspected"},
		{description: "Wrong type", path: clockTimeZonePath, valueType: proto.IntVal, value: "11111", expectedError: "expect string"},
		{description: "Constraint violation", path: hostNamePath, valueType: proto.StringVal, value: "not a host name", expectedError: "does not match regular expression pattern"},
	}

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Run the test cases
	for _, testCase := range testCases {
		thisTestCase := testCase
		s.Run(thisTestCase.description, func() {
			description := thisTestCase.description
			path := thisTestCase.path
			value := thisTestCase.value
			valueType := thisTestCase.valueType
			expectedError := thisTestCase.expectedError
			s.T().Logf("testing %q", description)

			setResult := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, path, value, valueType)
			var setReq = &gnmiutils.SetRequest{
				Ctx:         s.Context(),
				Client:      gnmiClient,
				Extensions:  s.SyncExtension(),
				Encoding:    gnmiapi.Encoding_PROTO,
				UpdatePaths: setResult,
			}
			msg, _, err := setReq.Set()
			s.NotNil(err, "Set operation for %s does not generate an error", description)
			s.Contains(status.Convert(err).Message(), expectedError,
				"set operation for %s generates wrong error %s", description, msg)
		})
	}
}
