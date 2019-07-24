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

const (
	badPath           = "/openconfig-system:system/config/no-such-path"
	ntpPath           = "/openconfig-system:system/ntp/state/enable-ntp-auth"
	hostnamePath      = "/openconfig-system:system/config/hostname"
	clockTimeZonePath = "/openconfig-system:system/clock/config/timezone-name"
)

func init() {
	Registry.RegisterTest("models", TestModels, []*runner.TestSuite{AllTests,SomeTests,IntegrationTests})
}

// TestModels tests GNMI operation involving unknown or illegal paths
func TestModels(t *testing.T) {
	// Get the configured devices from the environment.
	device1 := env.GetDevices()[0]
	device2 := env.GetDevices()[1]
	devices := make([]string, 2)
	devices[0] = device1
	devices[1] = device2

	// Make a GNMI client to use for requests
	gnmiClient, gnmiClientError := env.NewGnmiClient(MakeContext(), "gnmi")
	assert.NoError(t, gnmiClientError)
	assert.True(t, gnmiClient != nil, "Fetching client returned nil")

	// Try to set a bad path
	setBadPath := makeDevicePath(device1, badPath)
	setBadPath[0].value = "123456"
	_, errorBadSet := GNMISet(MakeContext(), gnmiClient, setBadPath, LeaveNamespaces)
	assert.NotNil(t, errorBadSet, "Set operation on unknown path does not generate an error")
	assert.Contains(t, errorBadSet.Error(),
		"JSON contains unexpected field no-such-path",
		"set operation on unknown path generates wrong error")

	// Try to set a read-only path
	setReadOnlyPath := makeDevicePath(device1, ntpPath)
	setReadOnlyPath[0].value = "bool_val:false"
	_, errorReadOnlySet := GNMISet(MakeContext(), gnmiClient, setReadOnlyPath, LeaveNamespaces)
	assert.NotNil(t, errorReadOnlySet, "Set operation on read-only path does not generate an error")
	assert.Contains(t, errorReadOnlySet.Error(),
		"read only",
		"set operation on unknown path generates wrong error")

	// Try to set the wrong type
	setWrongType := makeDevicePath(device1, clockTimeZonePath)
	setWrongType[0].value = "int_val:11111"
	_, errorWrongTypeSet := GNMISet(MakeContext(), gnmiClient, setWrongType, LeaveNamespaces)
	assert.NotNil(t, errorWrongTypeSet, "Set operation with bad type does not generate an error")
	assert.Contains(t, errorWrongTypeSet.Error(),
		"expect string",
		"set operation on unknown path generates wrong error")

	// Try to set a value that does not match constraints
	setWrongValue := makeDevicePath(device1, hostnamePath)
	setWrongValue[0].value = "not a host name"
	_, errorWrongValueSet := GNMISet(MakeContext(), gnmiClient, setWrongValue, LeaveNamespaces)
	assert.NotNil(t, errorWrongValueSet, "Set operation with bad type does not generate an error")
	assert.Contains(t, errorWrongValueSet.Error(),
		"does not match regular expression pattern",
		"set operation on unknown path generates wrong error")
}

