// Copyright 2022-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rbac

import (
	"context"
	"github.com/onosproject/onos-config/test/utils/rbac"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/onosproject/onos-config/test/utils/gnmi"
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
			name:          "Eather ROC Admin user",
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
	simulator := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator)

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
				gnmiClient := gnmi.GetGNMIClientWithContextOrFail(ctx, t)
				assert.NotNil(t, gnmiClient)

				// Get path for the test value
				targetPath := gnmi.GetTargetPathWithValue(simulator.Name(), tzPath, tzValue, proto.StringVal)
				assert.NotNil(t, targetPath)

				// Set a value using gNMI client
				_, _, err = gnmi.SetGNMIValueWithContext(ctx, t, gnmiClient, targetPath, gnmi.NoPaths, gnmi.NoExtensions)
				if testCase.expectedError != "" {
					assert.Contains(t, err.Error(), testCase.expectedError)
					return
				}

				// Check that the value was set correctly
				gnmi.CheckGNMIValueWithContext(ctx, t, gnmiClient, targetPath, tzValue, 0, "Query after set returned the wrong value")

				// Remove the path we added
				gnmi.SetGNMIValueWithContextOrFail(ctx, t, gnmiClient, gnmi.NoPaths, targetPath, gnmi.NoExtensions)
			},
		)
	}
}
