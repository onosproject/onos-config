// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package rbac

import (
	"context"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-config/test/utils/rbac"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
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
		var setReq = &gnmiutils.SetRequest{
			Ctx:         ctx,
			Client:      gnmiClient,
			Encoding:    gnmiapi.Encoding_PROTO,
			UpdatePaths: setNamePath,
		}
		setReq.SetOrFail(t)

		// Set initial values for Enabled and Description using gNMI client
		setInitialValuesPath := []proto.TargetPath{
			{TargetName: target, Path: enabledPath, PathDataValue: "true", PathDataType: proto.BoolVal},
			{TargetName: target, Path: descriptionPath, PathDataValue: descriptionLeafValue, PathDataType: proto.StringVal},
		}
		setReq.UpdatePaths = setInitialValuesPath
		setReq.SetOrFail(t)
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
				var onosConfigGetReq = &gnmiutils.GetRequest{
					Ctx:      ctx,
					Client:   gnmiClient,
					Paths:    targetPath,
					Encoding: gnmiapi.Encoding_PROTO,
					DataType: gnmiapi.GetRequest_CONFIG,
				}
				values, err := onosConfigGetReq.Get()
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
