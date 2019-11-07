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
	td1 "github.com/onosproject/onos-config/modelplugin/TestDevice-1.0.0/testdevice_1_0_0"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
	"io/ioutil"
	"testing"
)

type modelPluginTestDevice string

func (m modelPluginTestDevice) ModelData() (string, string, []*gnmi.ModelData, string) {
	return modelTypeTest, modelVersionTest, modelData, moduleNameTest
}

// UnmarshalConfigValues uses the `generated.go` of the TestDevice-1.0.0 plugin module
func (m modelPluginTestDevice) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	device := &td1.Device{}
	vgs := ygot.ValidatedGoStruct(device)

	if err := td1.Unmarshal(jsonTree, device); err != nil {
		return nil, err
	}

	return &vgs, nil
}

// Validate uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTestDevice) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*td1.Device)
	if !ok {
		return fmt.Errorf("unable to convert model in to testdevice_1_0_0")
	}
	return device.Validate()
}

// Schema uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTestDevice) Schema() (map[string]*yang.Entry, error) {
	return td1.UnzipSchema()
}

func Test_correctJsonPathValuesTd(t *testing.T) {

	var modelPluginTest modelPluginTestDevice

	td1Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(td1Schema), 6)

	readOnlyPaths, _ := modelregistry.ExtractPaths(td1Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readOnlyPaths), 2)

	// All values are taken from testdata/sample-testdevice-opstate.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-testdevice-opstate.json")
	assert.NilError(t, err)

	values, err := store.DecomposeTree(sampleTree)
	assert.NilError(t, err)
	assert.Equal(t, len(values), 6)

	correctedPathValues, err := CorrectJSONPaths("", values, readOnlyPaths, true)
	assert.NilError(t, err)

	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}
	assert.Equal(t, len(correctedPathValues), 4)

	for _, correctedPathValue := range correctedPathValues {
		switch correctedPathValue.Path {
		case
			"/cont1a/cont2a/leaf2c",
			"/cont1b-state/list2b[index=100]/leaf3c",
			"/cont1b-state/list2b[index=101]/leaf3c":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), devicechange.ValueType_STRING, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		case
			"/cont1b-state/leaf2d":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), devicechange.ValueType_UINT, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
}
