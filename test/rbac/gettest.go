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
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-config/test/utils/rbac"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	starbucksInterface   = "starbucks"
	acmeInterface        = "acme"
	otherInterface       = "other"
	descriptionLeafName  = "description"
	nameLeafName         = "name"
	enabledLeafName      = "enabled"
	descriptionLeafValue = "ABC123"
	keycloakURL          = "https://keycloak-dev.onlab.us/auth/realms/master"
)

func getLeafPath(interfaceName string, leafName string) string {
	rootPath := "/interfaces/interface[name=" + interfaceName + "]"
	return rootPath + "/config/" + leafName
}

func setUpInterfaces(t *testing.T, target string, password string) {
	// get an access token
	token, err := rbac.FetchATokenViaKeyCloak(keycloakURL, "alicea", password)
	assert.NoError(t, err)
	assert.NotNil(t, token)

	// Make a GNMI client to use for requests
	ctx := rbac.GetBearerContext(context.Background(), token)
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)

	var interfaceNames = [...]string{starbucksInterface, acmeInterface, otherInterface}

	for _, interfaceName := range interfaceNames {
		namePath := getLeafPath(interfaceName, nameLeafName)
		enabledPath := getLeafPath(interfaceName, enabledLeafName)
		descriptionPath := getLeafPath(interfaceName, descriptionLeafName)

		// Create interface tree using gNMI client
		setNamePath := []proto.TargetPath{
			{TargetName: target, Path: namePath, PathDataValue: interfaceName, PathDataType: proto.StringVal},
		}
		gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, setNamePath, gnmiutils.NoPaths, gnmiutils.NoExtensions)

		// Set initial values for Enabled and Description using gNMI client
		setInitialValuesPath := []proto.TargetPath{
			{TargetName: target, Path: enabledPath, PathDataValue: "true", PathDataType: proto.BoolVal},
			{TargetName: target, Path: descriptionPath, PathDataValue: descriptionLeafValue, PathDataType: proto.StringVal},
		}
		gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, setInitialValuesPath, gnmiutils.NoPaths, gnmiutils.NoExtensions)
	}
}

// TestGetOperations tests get operations to a protected API with various users
func (s *TestSuite) TestGetOperations(t *testing.T) {
	testCases := map[string]struct {
		userName      string
		interfaceName string
		expectedValue string
	}{
		// alicea should be able to see everything - AetherROCAdmin
		"alicea - starbucks": {
			userName:      "alicea",
			interfaceName: starbucksInterface,
			expectedValue: descriptionLeafValue,
		},
		"alicea - acme": {
			userName:      "alicea",
			interfaceName: acmeInterface,
			expectedValue: descriptionLeafValue,
		},
		"alicea - other": {
			userName:      "alicea",
			interfaceName: otherInterface,
			expectedValue: descriptionLeafValue,
		},

		// bobc can't see anything - not in any enterprise
		"bobc - starbucks": {
			userName:      "bobc",
			interfaceName: starbucksInterface,
			expectedValue: "",
		},
		"bobc - acme": {
			userName:      "bobc",
			interfaceName: acmeInterface,
			expectedValue: "",
		},
		"bobc - other": {
			userName:      "bobc",
			interfaceName: otherInterface,
			expectedValue: "",
		},

		// charlieb can't see anything - not in any enterprise
		"charlieb - starbucks": {
			userName:      "charlieb",
			interfaceName: starbucksInterface,
			expectedValue: "",
		},
		"charlieb - acme": {
			userName:      "charlieb",
			interfaceName: acmeInterface,
			expectedValue: "",
		},
		"charlieb - other": {
			userName:      "charlieb",
			interfaceName: otherInterface,
			expectedValue: "",
		},

		// daisyd can see starbucks
		"daisyd - starbucks": {
			userName:      "daisyd",
			interfaceName: starbucksInterface,
			expectedValue: descriptionLeafValue,
		},
		"daisyd - acme": {
			userName:      "daisyd",
			interfaceName: acmeInterface,
			expectedValue: "",
		},
		"daisyd - other": {
			userName:      "daisyd",
			interfaceName: otherInterface,
			expectedValue: "",
		},

		// elmerf can see starbucks
		"elmerf - starbucks": {
			userName:      "elmerf",
			interfaceName: starbucksInterface,
			expectedValue: descriptionLeafValue,
		},
		"elmerf - acme": {
			userName:      "elmerf",
			interfaceName: acmeInterface,
			expectedValue: "",
		},
		"elmerf - other": {
			userName:      "elmerf",
			interfaceName: otherInterface,
			expectedValue: "",
		},

		// fredf can see acme
		"fredf - starbucks": {
			userName:      "fredf",
			interfaceName: starbucksInterface,
			expectedValue: "",
		},
		"fredf - acme": {
			userName:      "fredf",
			interfaceName: acmeInterface,
			expectedValue: descriptionLeafValue,
		},
		"fredf - other": {
			userName:      "fredf",
			interfaceName: otherInterface,
			expectedValue: "",
		},

		// gandalfg can see acme
		"gandalfg - starbucks": {
			userName:      "gandalfg",
			interfaceName: starbucksInterface,
			expectedValue: "",
		},
		"gandalfg - acme": {
			userName:      "gandalfg",
			interfaceName: acmeInterface,
			expectedValue: descriptionLeafValue,
		},
		"gandalfg - other": {
			userName:      "gandalfg",
			interfaceName: otherInterface,
			expectedValue: "",
		},
	}

	// Create a simulated device
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	setUpInterfaces(t, simulator.Name(), s.keycloakPassword)

	for name, testCase := range testCases {
		t.Run(name,
			func(t *testing.T) {
				token, err := rbac.FetchATokenViaKeyCloak(keycloakURL, testCase.userName, s.keycloakPassword)
				assert.NoError(t, err)
				assert.NotNil(t, token)

				// Make a GNMI client to use for requests
				ctx := rbac.GetBearerContext(context.Background(), token)
				gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)
				assert.NotNil(t, gnmiClient)

				descriptionPath := getLeafPath(testCase.interfaceName, descriptionLeafName)

				// Get path for the test value
				targetPath := []proto.TargetPath{
					{TargetName: simulator.Name(), Path: descriptionPath, PathDataValue: testCase.interfaceName, PathDataType: proto.StringVal},
				}

				// Check that the value can be read via get
				values, _, err := gnmiutils.GetGNMIValue(ctx, gnmiClient, targetPath, gnmiutils.NoExtensions, gpb.Encoding_PROTO)
				assert.NoError(t, err)
				value := ""
				if len(values) != 0 {
					value = values[0].PathDataValue
				}
				assert.Equal(t, testCase.expectedValue, value)
			},
		)
	}
}
