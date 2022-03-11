// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	"github.com/onosproject/onos-config/test/utils/rbac"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
)

// TestSetOperations tests set operations to a protected API with various users
func (s *TestSuite) TestSetOperations(t *testing.T) {
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

	// Create a simulated target
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		t.Run(testCase.name,
			func(t *testing.T) {
				// get an access token
				token, err := rbac.FetchATokenViaKeyCloak("https://keycloak-dev.onlab.us/auth/realms/master", testCase.username, s.keycloakPassword)
				assert.NoError(t, err)
				assert.NotNil(t, token)

				// Make a GNMI client to use for requests
				ctx := rbac.GetBearerContext(context.Background(), token)
				gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)
				assert.NotNil(t, gnmiClient)

				// Get path for the test value
				targetPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), tzPath, tzValue, proto.StringVal)
				assert.NotNil(t, targetPath)

				// Set a value using gNMI client
				var setReq = &gnmiutils.SetRequest{
					Ctx:         ctx,
					Client:      gnmiClient,
					Encoding:    gnmiapi.Encoding_PROTO,
					UpdatePaths: targetPath,
				}
				_, _, err = setReq.Set()
				if testCase.expectedError != "" {
					assert.Contains(t, err.Error(), testCase.expectedError)
					return
				}

				// Check that the value was set correctly
				var getConfigReq = &gnmiutils.GetRequest{
					Ctx:      ctx,
					Client:   gnmiClient,
					Paths:    targetPath,
					Encoding: gnmiapi.Encoding_PROTO,
				}
				getConfigReq.CheckValues(t, tzValue)

				// Remove the path we added
				setReq.UpdatePaths = nil
				setReq.DeletePaths = targetPath
				setReq.SetOrFail(t)
			},
		)
	}
}
