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

package jsonvalues

import (
	"fmt"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
	"io/ioutil"
	"testing"
)

func Test_correctJsonPathValuesTd(t *testing.T) {

	var modelPluginTest modelPluginTestDevice2

	td2Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(td2Schema), 8)

	readOnlyPaths, _ := modelregistry.ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readOnlyPaths), 2)

	// All values are taken from testdata/sample-testdevice-opstate.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-testdevice2-opstate.json")
	assert.NilError(t, err)

	correctedPathValues, err := DecomposeJSONWithPaths(sampleTree, readOnlyPaths, nil)
	assert.NilError(t, err)
	assert.Equal(t, 8, len(correctedPathValues))

	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}

	for _, correctedPathValue := range correctedPathValues {
		switch correctedPathValue.Path {
		case
			"/cont1a/cont2a/leaf2c",
			"/cont1b-state/list2b[index1=101][index2=102]/leaf3c",
			"/cont1b-state/list2b[index1=101][index2=103]/leaf3c",
			"/cont1b-state/list2b[index1=101][index2=102]/leaf3d",
			"/cont1b-state/list2b[index1=101][index2=103]/leaf3d",
			"/cont1b-state/cont2c/leaf3b":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), devicechange.ValueType_STRING, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		case
			"/cont1b-state/leaf2d":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), devicechange.ValueType_UINT, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		case
			"/cont1b-state/cont2c/leaf3a":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), devicechange.ValueType_BOOL, correctedPathValue.Path)
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
}
