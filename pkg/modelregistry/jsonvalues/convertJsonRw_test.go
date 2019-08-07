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
	td1 "github.com/onosproject/onos-config/modelplugin/TestDevice-1.0.0/testdevice_1_0_0"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
	"io/ioutil"
	"testing"
)

func Test_correctJsonPathRwValuesSubInterfaces(t *testing.T) {

	var modelPluginTest modelPluginTest

	ds1Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)
	assert.Equal(t, len(ds1Schema), 137)

	_, readWritePaths := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readWritePaths), 113)

	// All values are taken from testdata/sample-openconfig.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-openconfig-configuration.json")
	assert.NilError(t, err)

	values, err := store.DecomposeTree(sampleTree)
	assert.NilError(t, err)
	assert.Equal(t, len(values), 11)

	correctedPathValues, err := CorrectJSONPaths("/interfaces", values, readWritePaths, true)
	assert.NilError(t, err)

	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}
	assert.Equal(t, len(correctedPathValues), 8)

	for _, correctedPathValue := range correctedPathValues {
		switch correctedPathValue.Path {
		case
			"/interfaces/interface[name=eth1]/config/description",
			"/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=120]/config/description",
			"/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=121]/config/description":
			assert.Equal(t, correctedPathValue.Type, change.ValueTypeSTRING, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.TypeOpts), 0)
		case
			"/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=120]/config/enabled",
			"/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=121]/config/enabled":
			assert.Equal(t, correctedPathValue.Type, change.ValueTypeBOOL, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.TypeOpts), 0)
		case
			"/interfaces/interface[name=eth1]/config/mtu",
			"/interfaces/interface[name=eth1]/hold-time/config/down",
			"/interfaces/interface[name=eth1]/hold-time/config/up":
			assert.Equal(t, correctedPathValue.Type, change.ValueTypeUINT, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.TypeOpts), 0)
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
}

func Test_correctJsonPathRwValuesSystemLogging(t *testing.T) {

	var modelPluginTest modelPluginTest

	ds1Schema, err := modelPluginTest.Schema()
	assert.NilError(t, err)

	_, readWritePaths := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")

	// All values are taken from testdata/sample-double-index.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-openconfig-double-index.json")
	assert.NilError(t, err)

	values, err := store.DecomposeTree(sampleTree)
	assert.NilError(t, err)
	assert.Equal(t, len(values), 10)

	correctedPathValues, err := CorrectJSONPaths("/system/logging", values, readWritePaths, true)
	assert.NilError(t, err)

	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}
	assert.Equal(t, len(correctedPathValues), 4)

	for _, correctedPathValue := range correctedPathValues {
		switch correctedPathValue.Path {
		case
			"/system/logging/remote-servers/remote-server[host=h1]/selectors/selector[facility=1,severity=2]/config/severity",
			"/system/logging/remote-servers/remote-server[host=h1]/selectors/selector[facility=1,severity=2]/config/facility",
			"/system/logging/console/selectors/selector[facility=3,severity=4]/config/facility",
			"/system/logging/console/selectors/selector[facility=3,severity=4]/config/severity":
			assert.Equal(t, correctedPathValue.Type, change.ValueTypeSTRING, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.TypeOpts), 0)
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
}

func Test_correctJsonPathRwValues2(t *testing.T) {

	td1Schema, _ := td1.UnzipSchema()

	_, readWritePaths := modelregistry.ExtractPaths(td1Schema["Device"], yang.TSUnset, "", "")
	assert.Equal(t, len(readWritePaths), 10)

	// All values are taken from testdata/sample-device1.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-device1.json")
	assert.NilError(t, err)

	values, err := store.DecomposeTree(sampleTree)
	assert.NilError(t, err)
	assert.Equal(t, len(values), 2)

	correctedPathValues, err := CorrectJSONPaths("/test1:cont1a/cont2a", values, readWritePaths, true)
	assert.NilError(t, err)

	for _, v := range correctedPathValues {
		fmt.Printf("%s %v\n", (*v).Path, v.String())
	}
	assert.Equal(t, len(correctedPathValues), 2)

	for _, correctedPathValue := range correctedPathValues {
		switch correctedPathValue.Path {
		case
			"/cont1a/cont2a/leaf2a":
			assert.Equal(t, correctedPathValue.Type, change.ValueTypeUINT, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.TypeOpts), 0)
			assert.Equal(t, correctedPathValue.String(), "12")
		case
			"/cont1a/cont2a/leaf2b":
			assert.Equal(t, correctedPathValue.Type, change.ValueTypeFLOAT, correctedPathValue.Path)
			assert.Equal(t, len(correctedPathValue.TypeOpts), 0)
			assert.Equal(t, correctedPathValue.String(), "1.210000")
		default:
			t.Fatal("Unexpected path", correctedPathValue.Path)
		}
	}
}
