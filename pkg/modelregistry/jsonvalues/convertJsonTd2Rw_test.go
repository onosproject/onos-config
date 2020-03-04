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
	td2 "github.com/onosproject/config-models/modelplugin/testdevice-2.0.0/testdevice_2_0_0"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
	"io/ioutil"
	"testing"
)

// For testing TestDevice-2.0.0 - which has an augmented YANG with Choice and Case
type modelPluginTestDevice2 string

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
func Test_correctJsonPathValuesTd2(t *testing.T) {

	var modelPluginTest modelPluginTestDevice2

	td2Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(td2Schema), 8)

	_, readWritePaths := modelregistry.ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readWritePaths), 15)

	// All values are taken from testdata/sample-testdevice2-choice.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-testdevice2-choice.json")
	assert.NilError(t, err)

	values, err := store.DecomposeTree(sampleTree)
	assert.NilError(t, err)
	assert.Equal(t, len(values), 2)

	correctedPathValues, err := CorrectJSONPaths("", values, readWritePaths, true)
	assert.NilError(t, err)

	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}
	assert.Equal(t, len(correctedPathValues), 2)

	for _, correctedPathValue := range correctedPathValues {
		switch correctedPathValue.Path {
		case
			"/cont1a/cont2d/leaf2d3c",
			"/cont1a/cont2d/chocolate":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), devicechange.ValueType_STRING, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
	ygotValues, err := modelPluginTest.UnmarshalConfigValues(sampleTree)
	assert.NilError(t, err, "Unexpected error unmarshalling values")
	err = modelPluginTest.Validate(ygotValues)
	assert.NilError(t, err, "Unexpected error Validation error")
}

// "chocolate" leaf is a "case" within a "choice". The other case has "beer" and "pretzel" together.
// See config-models/modelplugin/testdevice-2.0.0/yang/test1-augmented@2020-02-29.yang
func Test_correctJsonPathValuesTd2Wrong(t *testing.T) {
	var modelPluginTest modelPluginTestDevice2

	td2Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(td2Schema), 8)

	_, readWritePaths := modelregistry.ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")
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
