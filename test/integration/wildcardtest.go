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
	"encoding/json"
	"github.com/onosproject/onos-config/test/env"
	"github.com/onosproject/onos-config/test/runner"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)


// TestWildcard tests multiple types of wildcard requests
func TestWildcard(t *testing.T) {
	const (
		allState          = "/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/state"
		allStateAndConfig = "/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]"
		address           = "/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/*/address"
		addressState      = "/system/openflow/controllers/controller[name=*]/connections/connection[aux-id=*]/state/address"
	)

	//_, basePath, _, _     := runtime.Caller(0)
	//allStatePath         := filepath.Join(filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(basePath)), "integration"), "_resources"), "allState.json")
	//allStateAndConfigPath := filepath.Join(filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(basePath)), "integration"), "_resources"), "allStateAndConfig.json")
	//addressPath  := filepath.Join(filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(basePath)), "integration"), "_resources"), "address.json")
	//addressStatePath  := filepath.Join(filepath.Join(filepath.Join(filepath.Dir(filepath.Dir(basePath)), "integration"), "_resources"), "addressState.json")
	allStatePath         := filepath.Join("/etc/onos-config/test/integration/_resources", "allState.json")
	allStateAndConfigPath := filepath.Join("/etc/onos-config/test/integration/_resources", "allStateAndConfig.json")
	addressPath  := filepath.Join("/etc/onos-config/test/integration/_resources",  "address.json")
	addressStatePath  := filepath.Join("/etc/onos-config/test/integration/_resources",  "addressState.json")
	//"onos-config/test/integration/_resources"
	// Get the configured device from the environment.
	device := env.GetDevices()[0]

	// Data to run the test cases
	testCases := []struct {
		description  string
		path         string
		expectedJSON string
	}{
		{description: "All state", path: allState, expectedJSON: allStatePath},
		{description: "All state and config", path: allStateAndConfig, expectedJSON: allStateAndConfigPath},
		{description: "Address ", path: address, expectedJSON: addressPath},
		{description: "Address state", path: addressState, expectedJSON: addressStatePath},
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
				jsonFilePath := testCase.expectedJSON
				t.Parallel()

				t.Logf("testing %q", description)
				// Open our jsonFile
				directory,_ := os.Getwd()
				t.Logf(directory)
				files, err := ioutil.ReadDir(directory)
				if err != nil {
					t.Logf(err.Error())
				}

				for _, file := range files {
					t.Logf(file.Name())
				}
				files, err = ioutil.ReadDir("/etc/onos-config/test/integration/_resources")
				if err != nil {
					t.Logf(err.Error())
				}

				for _, file := range files {
					t.Logf(file.Name())
				}
				jsonFile, err := os.Open(jsonFilePath)
				// if we os.Open returns an error then handle it
				assert.NoError(t, err, "unexpected error while opening ", jsonFilePath)
				// defer the closing of our jsonFile so that we can parse it later on
				defer jsonFile.Close()
				expectedJSONBytes, _ := ioutil.ReadAll(jsonFile)

				reply, errorGet := GNMIGetResponse(MakeContext(), gnmiClient, makeDevicePath(device, path))
				assert.NoError(t, errorGet)
				jsonReply := reply.Notification[0].Update[0].Val.GetJsonVal()
				same, err := JSONBytesEqual(expectedJSONBytes, jsonReply)
				t.Logf("expectd json %s", string(expectedJSONBytes))
				t.Logf("got json %s", string(jsonReply))
				assert.True(t, same, "Json should be equal", string(expectedJSONBytes), string(jsonReply))
			})
	}
}

// JSONBytesEqual compares the JSON in two byte slices.
func JSONBytesEqual(a, b []byte) (bool, error) {
	var j, j2 interface{}
	if err := json.Unmarshal(a, &j); err != nil {
		return false, err
	}
	if err := json.Unmarshal(b, &j2); err != nil {
		return false, err
	}
	return reflect.DeepEqual(j2, j), nil
}

func init() {
	Registry.RegisterTest("wildcard", TestWildcard, []*runner.TestSuite{AllTests,IntegrationTests})
}
