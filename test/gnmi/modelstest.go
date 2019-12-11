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

package gnmi

import (
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/status"
	"testing"
)

// TestModels tests GNMI operation involving unknown or illegal paths
func (s *TestSuite) TestModels(t *testing.T) {
	const (
		unknownPath       = "/system/config/no-such-path"
		ntpPath           = "/system/ntp/state/enable-ntp-auth"
		hostNamePath      = "/system/config/hostname"
		clockTimeZonePath = "/system/clock/config/timezone-name"
	)

	simulator := env.NewSimulator().AddOrDie()

	// Data to run the test cases
	testCases := []struct {
		description   string
		path          string
		value         string
		valueType     string
		expectedError string
	}{
		{description: "Unknown path", path: unknownPath, valueType: StringVal, value: "123456", expectedError: "no-such-path"},
		{description: "Read only path", path: ntpPath, valueType: BoolVal, value: "false", expectedError: "read only"},
		{description: "Wrong type", path: clockTimeZonePath, valueType: IntVal, value: "11111", expectedError: "expect string"},
		{description: "Constraint violation", path: hostNamePath, valueType: StringVal, value: "not a host name", expectedError: "does not match regular expression pattern"},
	}

	// Make a GNMI client to use for requests
	gnmiClient, gnmiClientError := env.Config().NewGNMIClient()
	assert.NoError(t, gnmiClientError)
	assert.True(t, gnmiClient != nil, "Fetching client returned nil")

	// Run the test cases
	for _, testCase := range testCases {
		thisTestCase := testCase
		t.Run(thisTestCase.description,
			func(t *testing.T) {
				description := thisTestCase.description
				path := thisTestCase.path
				value := thisTestCase.value
				valueType := thisTestCase.valueType
				expectedError := thisTestCase.expectedError
				t.Parallel()

				t.Logf("testing %q", description)

				setResult := makeDevicePath(simulator.Name(), path)
				setResult[0].pathDataValue = value
				setResult[0].pathDataType = valueType
				msg, _, errorSet := gNMISet(MakeContext(), gnmiClient, setResult, noPaths, noExtensions)
				assert.NotNil(t, errorSet, "Set operation for %s does not generate an error", description)
				assert.Contains(t, status.Convert(errorSet).Message(), expectedError,
					"set operation for %s generates wrong error %s", description, msg)
			})
	}
}
