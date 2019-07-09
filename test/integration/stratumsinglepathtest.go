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
	"fmt"
	"testing"

	"github.com/onosproject/onos-config/test/env"
	"github.com/onosproject/onos-config/test/runner"
	"github.com/stretchr/testify/assert"
)

const (
	stratumPath = "/interfaces/"
)

// TestStratumSinglePath tests query/set/delete of a single GNMI path to a stratum device
func TestStratumSinglePath(t *testing.T) {
	// Get the first stratum configured device from the environment.
	device := env.GetDevices()[0]

	// Make a GNMI client to use for requests
	c, err := env.NewGnmiClient(MakeContext(), "")
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	devicePath, err := GNMIGet(MakeContext(), c, makeDevicePath(device, stratumPath))
	fmt.Println(devicePath, err)

}

func init() {
	Registry.RegisterTest("stratum-single-path", TestStratumSinglePath, []*runner.TestSuite{AllTests})

}
