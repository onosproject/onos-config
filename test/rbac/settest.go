// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-config/test/utils/rbac"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

// TestSetOperations tests set operations to a protected API with various users
func (s *TestSuite) TestSetOperations() {
	const (
		tzValue = "Europe/Dublin"
		tzPath  = "/system/clock/config/timezone-name"
	)

	type testCase struct {
		name          string
		username      string
		expectedError string
	}

	testCases := []testCase{
		{
			name:          "Aether ROC Admin user",
			username:      "alicea",
			expectedError: "",
		},
		{
			name:          "Enterprise Admin user",
			username:      "daisyd",
			expectedError: "",
		},
		{
			name:          "No access user",
			username:      "bobc",
			expectedError: "Set allowed only for AetherROCAdmin,EnterpriseAdmin",
		},
	}

	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		s.Run(testCase.name, func() {
			// get an access token
			token, err := rbac.FetchATokenViaKeyCloak("https://keycloak-dev.onlab.us/auth/realms/master", testCase.username, s.keycloakPassword)
			s.NoError(err)
			s.NotNil(token)

			// Make a GNMI client to use for requests
			ctx := rbac.GetBearerContext(s.Context(), token)
			gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.WithRetry)
			s.NotNil(gnmiClient)

			// Get path for the test value
			targetPath := gnmiutils.GetTargetPathWithValue(s.simulator.Name, tzPath, tzValue, proto.StringVal)
			s.NotNil(targetPath)

			// Set a value using gNMI client
			var setReq = &gnmiutils.SetRequest{
				Ctx:         ctx,
				Client:      gnmiClient,
				Encoding:    gnmiapi.Encoding_PROTO,
				UpdatePaths: targetPath,
			}
			_, _, err = setReq.Set()
			if testCase.expectedError != "" {
				s.Contains(err.Error(), testCase.expectedError)
				return
			}

			// Check that the value was set correctly
			var getConfigReq = &gnmiutils.GetRequest{
				Ctx:      ctx,
				Client:   gnmiClient,
				Paths:    targetPath,
				Encoding: gnmiapi.Encoding_PROTO,
			}
			getConfigReq.CheckValues(s.T(), tzValue)

			// Remove the path we added
			setReq.UpdatePaths = nil
			setReq.DeletePaths = targetPath
			setReq.SetOrFail(s.T())
		})
	}
}
