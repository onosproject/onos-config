// Copyright 2020-present Open Networking Foundation.
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
	"io/ioutil"
	"testing"

	pathutils "github.com/onosproject/onos-config/pkg/utils/path"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	td2 "github.com/onosproject/config-models/modelplugin/testdevice-2.0.0/testdevice_2_0_0"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
)

// For testing TestDevice-2.0.0 - which has an augmented YANG with Choice and Case
type modelPluginTestDevice2 string

const modelTypeTest = "TestModel"
const modelVersionTest = "0.0.1"
const moduleNameTest = "testmodel.so.1.0.0"

var modelData = []*gnmi.ModelData{
	{Name: "testmodel", Organization: "Open Networking Lab", Version: "2019-07-10"},
}

func (m modelPluginTestDevice2) ModelData() (string, string, []*gnmi.ModelData, string) {
	return modelTypeTest, modelVersionTest, modelData, moduleNameTest
}

// UnmarshalConfigValues uses the `generated.go` of the TestDevice-2.0.0 plugin module
func (m modelPluginTestDevice2) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	device := &td2.Device{}
	vgs := ygot.ValidatedGoStruct(device)

	if err := td2.Unmarshal(jsonTree, device); err != nil {
		return nil, err
	}

	return &vgs, nil
}

// Validate uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTestDevice2) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*td2.Device)
	if !ok {
		return fmt.Errorf("unable to convert model in to testdevice_1_0_0")
	}
	return device.Validate()
}

// Schema uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTestDevice2) Schema() (map[string]*yang.Entry, error) {
	return td2.UnzipSchema()
}

// chocolate is a "case" within a "choice"
func Test_DecomposeJSONWithPathsTd2_config(t *testing.T) {

	var modelPluginTest modelPluginTestDevice2

	td2Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, 8, len(td2Schema))

	readOnlyPaths, readWritePaths := pathutils.ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, 15, len(readWritePaths))

	sampleTree, err := ioutil.ReadFile("./testdata/sample-testdevice2-config.json")
	assert.NilError(t, err)
	assert.Equal(t, 462, len(sampleTree))

	pathValues, err := DecomposeJSONWithPaths("", sampleTree, readOnlyPaths, readWritePaths)
	assert.NilError(t, err)
	assert.Equal(t, len(pathValues), 10)

	for _, pathValue := range pathValues {
		//t.Logf("%v", pathValue)
		switch pathValue.Path {
		case
			"/cont1a/leaf1a":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_STRING, pathValue.Path)
		case
			"/cont1a/cont2a/leaf2e":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_LEAFLIST_INT, pathValue.Path)
			leaf2eLl, width := (*configapi.TypedLeafListInt)(&pathValue.Value).List()
			assert.Equal(t, configapi.WidthThirtyTwo, width)
			assert.Equal(t, 5, len(leaf2eLl))
			assert.DeepEqual(t, []int64{5, 4, 3, 2, 1}, leaf2eLl)
		case
			"/cont1a/cont2a/leaf2a",
			"/cont1a/list2a[name=l2a1]/tx-power",
			"/cont1a/list2a[name=l2a1]/rx-power",
			"/cont1a/list2a[name=l2a2]/tx-power",
			"/cont1a/list2a[name=l2a2]/rx-power":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_UINT, pathValue.Path)
		case
			"/cont1a/cont2a/leaf2b",
			"/cont1a/cont2a/leaf2d":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_DECIMAL, pathValue.Path)
		case
			"/cont1a/cont2a/leaf2f":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_BYTES, pathValue.Path)
		case
			"/cont1a/cont2a/leaf2g":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_BOOL, pathValue.Path)

		default:
			t.Fatal("Unexpected path", pathValue.Path)
		}
	}
	ygotValues, err := modelPluginTest.UnmarshalConfigValues(sampleTree)
	assert.NilError(t, err, "Unexpected error unmarshalling values")
	err = modelPluginTest.Validate(ygotValues)
	assert.NilError(t, err, "Unexpected error Validation error")
}

// chocolate is a "case" within a "choice"
func Test_DecomposeJSONWithPathsTd2Choice(t *testing.T) {

	var modelPluginTest modelPluginTestDevice2

	td2Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(td2Schema), 8)

	readOnlyPaths, readWritePaths := pathutils.ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")

	sampleTree, err := ioutil.ReadFile("./testdata/sample-testdevice2-choice.json")
	assert.NilError(t, err)

	pathValues, err := DecomposeJSONWithPaths("", sampleTree, readOnlyPaths, readWritePaths)
	assert.NilError(t, err)
	assert.Equal(t, len(pathValues), 2)

	for _, pathValue := range pathValues {
		//t.Logf("%v", pathValue)
		switch pathValue.Path {
		case
			"/cont1a/cont2d/leaf2d3c",
			"/cont1a/cont2d/chocolate":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_STRING, pathValue.Path)
			assert.Equal(t, len(pathValue.Value.GetTypeOpts()), 0)
		default:
			t.Fatal("Unexpected path", pathValue.Path)
		}
	}
	ygotValues, err := modelPluginTest.UnmarshalConfigValues(sampleTree)
	assert.NilError(t, err, "Unexpected error unmarshalling values")
	err = modelPluginTest.Validate(ygotValues)
	assert.NilError(t, err, "Unexpected error Validation error")
}

// "chocolate" leaf is a "case" within a "choice". The other case has "beer" and "pretzel" together.
// See config-models/modelplugin/testdevice-2.0.0/yang/test1-augmented@2020-02-29.yang
func Test_ValidateTd2Wrong(t *testing.T) {
	var modelPluginTest modelPluginTestDevice2

	td2Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(td2Schema), 8)

	_, readWritePaths := pathutils.ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readWritePaths), 15)

	// All values are taken from testdata/sample-testdevice2-choice.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-testdevice2-choice-wrong.json")
	assert.NilError(t, err)

	ygotValues, err := modelPluginTest.UnmarshalConfigValues(sampleTree)
	assert.NilError(t, err, "Unexpected error unmarshalling values")
	err = modelPluginTest.Validate(ygotValues)
	t.Logf("Validate should be throwing error here since all elements of choice "+
		"are present and should not be. Raise issue on YGOT. %v", err)
	//assert.Error(t, err, "choice")
}

func Test_DecomposeJSONWithPathsTd2OpState(t *testing.T) {

	td2Schema, err := td2.UnzipSchema()
	assert.NilError(t, err)
	assert.Equal(t, len(td2Schema), 8)

	readOnlyPaths, _ := pathutils.ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readOnlyPaths), 3)

	sampleTree, err := ioutil.ReadFile("./testdata/sample-testdevice2-opstate.json")
	assert.NilError(t, err)

	pathValues, err := DecomposeJSONWithPaths("", sampleTree, readOnlyPaths, nil)
	assert.NilError(t, err)
	assert.Equal(t, 8, len(pathValues))

	for _, pathValue := range pathValues {
		//t.Logf("%v", pathValue)
		switch pathValue.Path {
		case
			"/cont1a/cont2a/leaf2c",
			"/cont1b-state/list2b[index1=101][index2=102]/leaf3c",
			"/cont1b-state/list2b[index1=101][index2=103]/leaf3c",
			"/cont1b-state/list2b[index1=101][index2=102]/leaf3d",
			"/cont1b-state/cont2c/leaf3b":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_STRING, pathValue.Path)
			assert.Equal(t, len(pathValue.Value.GetTypeOpts()), 0)
		case "/cont1b-state/list2b[index1=101][index2=103]/leaf3d":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_STRING, pathValue.Path)
			strVal := (*configapi.TypedString)(&pathValue.Value).String()
			assert.Equal(t, "IDTYPE2", strVal) // Should do the conversion fron "2" to "IDTYPE2"
		case
			"/cont1b-state/leaf2d":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_UINT, pathValue.Path)
			assert.Equal(t, len(pathValue.Value.GetTypeOpts()), 1)
		case
			"/cont1b-state/cont2c/leaf3a":
			assert.Equal(t, pathValue.Value.GetType(), configapi.ValueType_BOOL, pathValue.Path)
		default:
			t.Fatal("Unexpected path", pathValue.Path)
		}
	}
}
