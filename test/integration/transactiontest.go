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
	"context"
	admin "github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/onosproject/onos-config/test/env"
	"github.com/onosproject/onos-config/test/runner"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	value1 = "v1"
	path1  = "/a/b/c1"
	value2 = "v2"
	path2  = "/a/b/c2"
)

// TestTransaction tests setting multiple paths in a single request and rolling it back
func TestTransaction(t *testing.T) {
	// Get the first configured device from the environment.
	device := env.GetDevices()[0]

	// Make a GNMI client to use for requests
	c, err := env.NewGnmiClient(MakeContext(), "")
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Set values
	var devicePathsForSet = make([]DevicePath, 2)
	devicePathsForSet[0].deviceName = device
	devicePathsForSet[0].path = path1
	devicePathsForSet[0].value = value1
	devicePathsForSet[1].deviceName = device
	devicePathsForSet[1].path = path2
	devicePathsForSet[1].value = value2
	changeID, errorSet := GNMISet(MakeContext(), c, devicePathsForSet)
	assert.NoError(t, errorSet)
	assert.True(t, changeID != "")

	var devicePathsForGet = make([]DevicePath, 2)
	devicePathsForGet[0].deviceName = device
	devicePathsForGet[0].path = path1
	devicePathsForGet[1].deviceName = device
	devicePathsForGet[1].path = path2

	// Check that the values were set correctly
	valueAfter, errorAfter := GNMIGet(MakeContext(), c, devicePathsForGet)
	assert.NoError(t, errorAfter)
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, value1, valueAfter[0].value, "Query after set returned the wrong value: %s\n", valueAfter)
	assert.Equal(t, value2, valueAfter[1].value, "Query after set 2 returned the wrong value: %s\n", valueAfter)

	// Now rollback the change
	_, adminClient := env.GetAdminClient()

	response, clientErr := adminClient.RollbackNetworkChange(
		context.Background(), &admin.RollbackRequest{Name: changeID})

	assert.NoError(t, clientErr)
	assert.NotNil(t, response)

	// Check that the values were really rolled back
	_, errorAfterRollback := GNMIGet(MakeContext(), c, devicePathsForGet)
	assert.Nil(t, errorAfterRollback)
}

func init() {
	Registry.RegisterTest("transaction", TestTransaction, []*runner.TestSuite{AllTests,SomeTests,IntegrationTests})
	//example of registering groups
}
