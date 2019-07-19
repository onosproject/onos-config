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
	//value1 = "test-motd-banner"
	//path1  = "/openconfig-system:system/config/motd-banner"
	//value2 = "test-login-banner"
	//path2  = "/openconfig-system:system/config/login-banner"
)

var (
	//paths = []string {path1, path2}
	//values = []string {value1, value2}
)

func init() {
	Registry.RegisterTest("models", TestModels, []*runner.TestSuite{AllTests,SomeTests,IntegrationTests})
}

// TestTransaction tests setting multiple paths in a single request and rolling it back
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
}

