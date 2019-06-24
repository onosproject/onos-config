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
	"errors"
	"fmt"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/test/env"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	tzValue = "Europe/Dublin"
	tzPath  = "/system/clock/config/timezone-name"
)

func findPathValue(response *gpb.GetResponse, path string) (string, error) {
	if len(response.Notification) != 1 {
		return "", errors.New("response notifications must have one entry")
	}

	pathElements := utils.SplitPath(path)
	responsePathElements := response.Notification[0].Update[0].Path.Elem
	for pathIndex, pathElement := range pathElements {
		responsePathElement := responsePathElements[pathIndex]
		if pathElement != responsePathElement.Name {
			return "", fmt.Errorf("element at %d dos not match - want %s got %s", pathIndex, pathElement, responsePathElement.Name)
		}
	}
	value := response.Notification[0].Update[0].Val
	if value == nil {
		return "", fmt.Errorf("no value found for path %s", path)
	}
	return utils.StrVal(value), nil
}

func TestSinglePath(t *testing.T) {
	// Get the first configured device from the environment.
	device := env.GetDevices()[0]

	// Make a GNMI client to use for requests
	c, err := env.NewGnmiClient(MakeContext(), "")
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// First lookup should return an error - the path has not been given a value yet
	valueBefore, findErrorBefore := GNMIGet(MakeContext(), c, device, tzPath)
	assert.Error(t, findErrorBefore, "no value found for path")
	assert.True(t, valueBefore == "", "Initial query did not return an error\n")

	// Set a value using gNMI client
	errorSet := GNMISet(MakeContext(), c, device, tzPath, tzValue)
	assert.NoError(t, errorSet)

	// Check that the value was set correctly
	valueAfter, errorAfter := GNMIGet(MakeContext(), c, device, tzPath)
	assert.NoError(t, errorAfter)
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, tzValue, valueAfter, "Query after set returned the wrong value: %s\n", valueAfter)

	// Remove the path we added
	errorDelete := GNMIDelete(MakeContext(), c, device, tzPath)
	assert.NoError(t, errorDelete)

	//  Make sure it got removed
	_, errorAfterDelete := GNMIGet(MakeContext(), c, device, tzPath)
	assert.NotNil(t, errorAfterDelete)
	assert.EqualError(t, errorAfterDelete, "no value found for path /system/clock/config/timezone-name")
}

func init() {
	Registry.Register("test-single-path-test", TestSinglePath)
}
