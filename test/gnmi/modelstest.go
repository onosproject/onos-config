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
	"testing"

	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/status"
)

// TestModels tests GNMI operation involving unknown or illegal paths
func (s *TestSuite) TestModels(t *testing.T) {
	const (
		unknownPath       = "/system/config/no-such-path"
		ntpPath           = "/system/ntp/state/enable-ntp-auth"
		hostNamePath      = "/system/config/hostname"
		clockTimeZonePath = "/system/clock/config/timezone-name"
	)

	simulator := gnmi.CreateSimulator(t)

	// Data to run the test cases
	testCases := []struct {
		description   string
		path          string
		value         string
		valueType     string
		expectedError string
	}{
		{description: "Unknown path", path: unknownPath, valueType: proto.StringVal, value: "123456", expectedError: "no-such-path"},
		{description: "Read only path", path: ntpPath, valueType: proto.BoolVal, value: "false", expectedError: "read only"},
		{description: "Wrong type", path: clockTimeZonePath, valueType: proto.IntVal, value: "11111", expectedError: "expect string"},
		{description: "Constraint violation", path: hostNamePath, valueType: proto.StringVal, value: "not a host name", expectedError: "does not match regular expression pattern"},
	}

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

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

				setResult := gnmi.GetDevicePathWithValue(simulator.Name(), path, value, valueType)
				msg, _, errorSet := gnmi.SetGNMIValue(gnmi.MakeContext(), gnmiClient, setResult, gnmi.NoPaths, gnmi.NoExtensions)
				assert.NotNil(t, errorSet, "Set operation for %s does not generate an error", description)
				assert.Contains(t, status.Convert(errorSet).Message(), expectedError,
					"set operation for %s generates wrong error %s", description, msg)
			})
	}
}
