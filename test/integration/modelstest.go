// Copyright 2019-present Open Networking Foundation.
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

package integration

import (
	"github.com/onosproject/onos-config/test/env"
	"github.com/onosproject/onos-config/test/runner"
	"github.com/stretchr/testify/assert"
	"testing"
)

func init() {
	Registry.RegisterTest("models", TestModels, []*runner.TestSuite{AllTests, SomeTests, IntegrationTests})
}

// TestModels tests GNMI operation involving unknown or illegal paths
func TestModels(t *testing.T) {
	const (
		unknownPath       = "/system/config/no-such-path"
		ntpPath           = "/system/ntp/state/enable-ntp-auth"
		hostNamePath      = "/system/config/hostname"
		clockTimeZonePath = "/system/clock/config/timezone-name"
	)

	// Get the configured device from the environment.
	device := env.GetDevices()[0]

	// Data to run the test cases
	testCases := []struct {
		description   string
		path          string
		value         string
		valueType     string
		expectedError string
	}{
		{description: "Unknown path", path: unknownPath, valueType: StringVal, value: "123456", expectedError: "JSON contains unexpected field no-such-path"},
		{description: "Read only path", path: ntpPath, valueType: BoolVal, value: "false", expectedError: "read only"},
		{description: "Wrong type", path: clockTimeZonePath, valueType: IntVal, value: "11111", expectedError: "expect string"},
		{description: "Constraint violation", path: hostNamePath, valueType: StringVal, value: "not a host name", expectedError: "does not match regular expression pattern"},
	}

	// Make a GNMI client to use for requests
	gnmiClient, gnmiClientError := env.NewGnmiClient(MakeContext(), "gnmi")
	assert.NoError(t, gnmiClientError)
	assert.True(t, gnmiClient != nil, "Fetching client returned nil")

	// Run the test cases
	for _, testCase := range testCases {
		t.Run(testCase.description,
			func(t *testing.T) {
				description := testCase.description
				path := testCase.path
				value := testCase.value
				valueType := testCase.valueType
				expectedError := testCase.expectedError
				t.Parallel()

				t.Logf("testing %q", description)

				setResult := makeDevicePath(device, path)
				setResult[0].pathDataValue = value
				setResult[0].pathDataType = valueType
				_, errorSet := GNMISet(MakeContext(), gnmiClient, setResult)
				assert.NotNil(t, errorSet, "Set operation for %s does not generate an error", description)
				assert.Contains(t, errorSet.Error(),
					expectedError,
					"set operation for %s generates wrong error", description)
			})
	}
}
