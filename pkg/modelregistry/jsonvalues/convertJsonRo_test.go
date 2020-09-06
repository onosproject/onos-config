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
	ds1 "github.com/onosproject/config-models/modelplugin/devicesim-1.0.0/devicesim_1_0_0"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
	"io/ioutil"
	"testing"
)

type modelPluginTest string

const modelTypeTest = "TestModel"
const modelVersionTest = "0.0.1"
const moduleNameTest = "testmodel.so.1.0.0"

var modelData = []*gnmi.ModelData{
	{Name: "testmodel", Organization: "Open Networking Lab", Version: "2019-07-10"},
}

func (m modelPluginTest) ModelData() (string, string, []*gnmi.ModelData, string) {
	return modelTypeTest, modelVersionTest, modelData, moduleNameTest
}

// UnmarshalConfigValues uses the `generated.go` of the DeviceSim-1.0.0 plugin module
func (m modelPluginTest) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	device := &ds1.Device{}
	vgs := ygot.ValidatedGoStruct(device)

	if err := ds1.Unmarshal(jsonTree, device); err != nil {
		return nil, err
	}

	return &vgs, nil
}

// Validate uses the `generated.go` of the DeviceSim-1.0.0 plugin module
func (m modelPluginTest) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*ds1.Device)
	if !ok {
		return fmt.Errorf("unable to convert model in to testdevice_1_0_0")
	}
	return device.Validate()
}

// Schema uses the `generated.go` of the DeviceSim-1.0.0 plugin module
func (m modelPluginTest) Schema() (map[string]*yang.Entry, error) {
	return ds1.UnzipSchema()
}

func Test_correctJsonPathValues2(t *testing.T) {

	var modelPluginTest modelPluginTest

	ds1Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(ds1Schema), 137)

	readOnlyPaths, readWritePaths := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readOnlyPaths), 37)

	// All values are taken from testdata/sample-openconfig.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-openconfig2.json")
	assert.NilError(t, err)

	correctedPathValues, err := DecomposeJSONWithPaths(sampleTree, readOnlyPaths, readWritePaths)
	assert.NilError(t, err)
	assert.Equal(t, len(correctedPathValues), 24)
	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}

	for _, correctedPathValue := range correctedPathValues {
		switch correctedPathValue.Path {
		case
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/source-interface",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/transport",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/address",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/aux-id",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/port",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/source-interface",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/transport",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/address",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/aux-id",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/port",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/source-interface",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/transport",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/address",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/aux-id",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/port",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/source-interface",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/transport",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/address",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/aux-id",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/port":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), devicechange.ValueType_STRING, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		case
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/priority",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/priority",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/priority",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/priority":
			assert.Equal(t, correctedPathValue.GetValue().GetType(), devicechange.ValueType_UINT, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.GetValue().GetTypeOpts()), 0)
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
}
