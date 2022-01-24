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
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-config/test/utils/rbac"
)

// TestBadTokens tests access to a protected API with no access token supplied
func (s *TestSuite) TestBadTokens(t *testing.T) {
	const (
		tzValue      = "Europe/Dublin"
		tzPath       = "/system/clock/config/timezone-name"
		openIDIssuer = "https://keycloak-dev.onlab.us/auth/realms/master"
		expiredToken = "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJ5bWkyYUZqS2RGRTdkY1VobnV4QzhHU2Y2NXM5SVpjUm1zVWtMVDBmaHZBIn0.eyJleHAiOjE2NDE5MzMxMDIsImlhdCI6MTY0MTkzMzA0MiwianRpIjoiMzkxMzkzZTctM2FhZC00ZmQzLThlNTUtOGI5OTQ1ZmVhMTMyIiwiaXNzIjoiaHR0cHM6Ly9rZXljbG9hay1kZXYub25sYWIudXMvYXV0aC9yZWFsbXMvbWFzdGVyIiwic3ViIjoiNDAzOTdmZWQtNzc2Ny00ZmM4LTg3YzEtOWI4MTY2MTU1NDdhIiwidHlwIjoiQmVhcmVyIiwiYXpwIjoib25vcy1jb25maWctdGVzdCIsInNlc3Npb25fc3RhdGUiOiJjODViNWE4MC0zY2IwLTQ1MTgtOWI2YS00NDQ4YWU4NGU0MzciLCJhY3IiOiIxIiwic2NvcGUiOiJvcGVuaWQgZW1haWwgcHJvZmlsZSBncm91cHMiLCJzaWQiOiJjODViNWE4MC0zY2IwLTQ1MTgtOWI2YS00NDQ4YWU4NGU0MzciLCJlbWFpbF92ZXJpZmllZCI6dHJ1ZSwibmFtZSI6IkFsaWNlIEFkbWluIiwiZ3JvdXBzIjpbIkFldGhlclJPQ0FkbWluIiwibWl4ZWRHcm91cCJdLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJhbGljZWEiLCJnaXZlbl9uYW1lIjoiQWxpY2UiLCJmYW1pbHlfbmFtZSI6IkFkbWluIiwiZW1haWwiOiJhbGljZWFAb3Blbm5ldHdvcmtpbmcub3JnIn0.eaVrlbX9XXOQg006_N4Y_OJ6MXBMUPepPRd6PZGWbhieEafRv4CvlCL6Jy2W4tdTmkY5bjXLLXwQdum3iesJKJ8QcXlgoTc1N1MwyfZAChMLtuN1y75VAfWKLL5fctpP8ZlFC3xDj84IM46KpYUiwocv1TGFrTE0LNd2xC5bB3ajmiZ69KyetJjBZ04xvLfKcY88XEXNVb-FqoRaoCiMavqio9Dk_2CBVzbdVvv-mtvGfYJzPE3dEadkKG8w8P2o46Iv8FL_NjaNPNm0_LZAtOPx3KMRYZOBnR0-Gd2p1eyN-h4VZr9T2k0u5dm-7bmxTWNTwyF_PP7bUIUJhS6gYw"
	)

	type testCase struct {
		name                string
		username            string
		password            string
		token               string
		expectedClientError string
		expectedGetError    string
	}

	testCases := []testCase{
		{
			name:                "Bad Password",
			username:            "alicec",
			password:            "not-the-password",
			token:               "",
			expectedClientError: "401",
		},
		{
			name:                "Bad User",
			username:            "go-ask-alice",
			password:            "I-think-she'll-know",
			token:               "",
			expectedClientError: "401",
		},
		{
			name:             "Expired Token",
			username:         "alicea",
			password:         s.keycloakPassword,
			token:            expiredToken,
			expectedGetError: "Token is expired",
		},
		{
			name:             "Bad Token",
			username:         "alicea",
			password:         s.keycloakPassword,
			token:            expiredToken + "X",
			expectedGetError: "verification error",
		},
		{
			name:             "Token from another server",
			username:         "alicea",
			password:         s.keycloakPassword,
			token:            "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
			expectedGetError: "signature is invalid",
		},
		{
			name:             "Empty token",
			username:         "alicea",
			password:         s.keycloakPassword,
			token:            "",
			expectedGetError: "token contains an invalid number of segments",
		},
	}

	// Create a simulated target
	simulator := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator)

	devicePath := gnmi.GetTargetPathWithValue(simulator.Name(), tzPath, tzValue, proto.StringVal)

	// Run the test cases
	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		t.Run(testCase.name,
			func(t *testing.T) {

				token, err := rbac.FetchATokenViaKeyCloak(openIDIssuer, testCase.username, testCase.password)
				if testCase.expectedClientError != "" {
					assert.Contains(t, err.Error(), testCase.expectedClientError)
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, token)

				// Make a GNMI client to use for requests
				ctx := rbac.GetBearerContext(context.Background(), testCase.token)
				gnmiClient := gnmi.GetGNMIClientWithContextOrFail(ctx, t)

				// Try to fetch a value from the GNMI client
				_, _, err = gnmi.GetGNMIValue(ctx, gnmiClient, devicePath, gpb.Encoding_PROTO)

				if testCase.expectedGetError != "" {
					// An error is expected
					assert.Error(t, err)
					if err != nil {
						assert.Contains(t, err.Error(), testCase.expectedGetError)
					}
				}
			},
		)
	}
}
