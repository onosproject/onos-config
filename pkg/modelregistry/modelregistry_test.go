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

package modelregistry

import (
	"fmt"
	td1 "github.com/onosproject/onos-config/modelplugin/TestDevice-1.0.0/testdevice_1_0_0"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"gotest.tools/assert"
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

// UnmarshalConfigValues uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTest) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	device := &td1.Device{}
	vgs := ygot.ValidatedGoStruct(device)

	if err := td1.Unmarshal([]byte(jsonTree), device); err != nil {
		return nil, err
	}

	return &vgs, nil
}

// Validate uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTest) Validate(ygotModel *ygot.ValidatedGoStruct, opts ...ygot.ValidationOption) error {
	deviceDeref := *ygotModel
	device, ok := deviceDeref.(*td1.Device)
	if !ok {
		return fmt.Errorf("unable to convert model in to testdevice_1_0_0")
	}
	return device.Validate()
}

// Schema uses the `generated.go` of the TestDevice1 plugin module
func (m modelPluginTest) Schema() (map[string]*yang.Entry, error) {
	return td1.UnzipSchema()
}

func Test_CastModelPlugin(t *testing.T) {
	var modelPluginTest modelPluginTest
	mpt := interface{}(modelPluginTest)

	modelPlugin, ok := mpt.(ModelPlugin)
	assert.Assert(t, ok, "Testing cast of model plugin")
	name, version, _, _ := modelPlugin.ModelData()
	assert.Equal(t, name, modelTypeTest)
	assert.Equal(t, version, modelVersionTest)

}

func Test_Schema(t *testing.T) {
	var modelPluginTest modelPluginTest

	td1Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(td1Schema), 6)

	readOnlyPaths := extractReadOnlyPaths(td1Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readOnlyPaths), 6)
	// Can be in any order
	for _, p := range readOnlyPaths {
		switch p {
		case "/test1:cont1a/cont2a/leaf2c",
			"/test1:cont1b-state",
			"/test1:cont1b-state/leaf2d",
			"/test1:cont1b-state/list2b[index=*]",
			"/test1:cont1b-state/list2b[index=*]/index",
			"/test1:cont1b-state/list2b[index=*]/leaf3c":
		default:
			t.Fatal("Unexpected readOnlyPath", p)
		}
	}
}

func Test_RemovePathIndices(t *testing.T) {
	assert.Equal(t,
		RemovePathIndices("/test1:cont1b-state"),
		"/test1:cont1b-state")

	assert.Equal(t,
		RemovePathIndices("/test1:cont1b-state/list2b[index=test1]/index"),
		"/test1:cont1b-state/list2b[index=*]/index")

	assert.Equal(t,
		RemovePathIndices("/test1:cont1b-state/list2b[index=test1,test2]/index"),
		"/test1:cont1b-state/list2b[index=*]/index")

	assert.Equal(t,
		RemovePathIndices("/test1:cont1b-state/list2b[index=test1]/index/t3[name=5]"),
		"/test1:cont1b-state/list2b[index=*]/index/t3[name=*]")

}

func Test_formatName1(t *testing.T) {
	dirEntry1 := yang.Entry{
		Name:   "testname",
		Prefix: &yang.Value{Name: "pfx1"},
	}
	assert.Equal(t, formatName(&dirEntry1, false, "pfx1", "/pfx1:testpath/testpath2"), "/pfx1:testpath/testpath2/testname")
}

func Test_formatName2(t *testing.T) {
	dirEntry1 := yang.Entry{
		Name:   "testname",
		Key:    "name",
		Prefix: &yang.Value{Name: "pfx1"},
	}
	assert.Equal(t, formatName(&dirEntry1, true, "pfx1", "/pfx1:testpath/testpath2"), "/pfx1:testpath/testpath2/testname[name=*]")
}
