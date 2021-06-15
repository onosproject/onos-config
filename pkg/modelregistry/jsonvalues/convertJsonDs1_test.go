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
	"io/ioutil"
	"testing"

	ds1 "github.com/onosproject/config-models/modelplugin/devicesim-1.0.0/devicesim_1_0_0"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
)

func Test_correctJsonPathValues2(t *testing.T) {

	ds1Schema, err := ds1.UnzipSchema()
	assert.NilError(t, err)
	assert.Equal(t, len(ds1Schema), 137)

	readOnlyPaths, readWritePaths := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")

	sampleTree, err := ioutil.ReadFile("./testdata/sample-openconfig2.json")
	assert.NilError(t, err)

	pathValues, err := DecomposeJSONWithPaths("", sampleTree, readOnlyPaths, readWritePaths)
	assert.NilError(t, err)
	assert.Equal(t, len(pathValues), 24)

	for _, pathValue := range pathValues {
		t.Logf("%v", pathValue)
		switch pathValue.Path {
		case
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/source-interface",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/transport",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/address",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/source-interface",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/transport",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/address",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/source-interface",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/transport",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/address",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/source-interface",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/transport",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/address":
			assert.Equal(t, pathValue.GetValue().GetType(), devicechange.ValueType_STRING, pathValue.Path)
			assert.Equal(t, len(pathValue.GetValue().GetTypeOpts()), 0)
		case
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/priority",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/priority",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/priority",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/priority",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/aux-id",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/aux-id",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/aux-id",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/aux-id",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=10]/state/port",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=10]/state/port",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=11]/state/port",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=11]/state/port":
			assert.Equal(t, pathValue.GetValue().GetType(), devicechange.ValueType_UINT, pathValue.Path)
			assert.Equal(t, len(pathValue.GetValue().GetTypeOpts()), 1)
		default:
			t.Fatal("Unexpected path", pathValue.Path)
		}
	}
}

func Test_correctJsonPathRwValuesSubInterfaces(t *testing.T) {

	ds1Schema, err := ds1.UnzipSchema()
	assert.NilError(t, err)
	readOnlyPaths, readWritePaths := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")

	sampleTree, err := ioutil.ReadFile("./testdata/sample-openconfig-configuration.json")
	assert.NilError(t, err)

	pathValues, err := DecomposeJSONWithPaths("/interfaces/interface[name=eth1]", sampleTree, readOnlyPaths, readWritePaths)
	assert.NilError(t, err)
	assert.Equal(t, len(pathValues), 8)

	for _, pathValue := range pathValues {
		t.Logf("%v", pathValue)
		switch pathValue.Path {
		case
			"/interfaces/interface[name=eth1]/config/description",
			"/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=120]/config/description",
			"/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=121]/config/description":
			assert.Equal(t, pathValue.GetValue().GetType(), devicechange.ValueType_STRING, pathValue.Path)
			assert.Equal(t, len(pathValue.GetValue().GetTypeOpts()), 0)
		case
			"/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=120]/config/enabled",
			"/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=121]/config/enabled":
			assert.Equal(t, pathValue.GetValue().GetType(), devicechange.ValueType_BOOL, pathValue.Path)
			assert.Equal(t, len(pathValue.GetValue().GetTypeOpts()), 0)
		case
			"/interfaces/interface[name=eth1]/config/mtu",
			"/interfaces/interface[name=eth1]/hold-time/config/down",
			"/interfaces/interface[name=eth1]/hold-time/config/up":
			assert.Equal(t, pathValue.GetValue().GetType(), devicechange.ValueType_UINT, pathValue.Path)
			assert.Equal(t, len(pathValue.GetValue().GetTypeOpts()), 1)
		default:
			t.Fatal("Unexpected path", pathValue.Path)
		}
	}
}

func Test_correctJsonPathRwValuesSystemLogging(t *testing.T) {

	ds1Schema, err := ds1.UnzipSchema()
	assert.NilError(t, err)

	readOnlyPaths, readWritePaths := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")

	// All values are taken from testdata/sample-double-index.json and defined
	// here in the intermediate jsonToValues format
	sampleTree, err := ioutil.ReadFile("./testdata/sample-openconfig-double-index.json")
	assert.NilError(t, err)
	assert.Equal(t, 851, len(sampleTree))

	pathValues, err := DecomposeJSONWithPaths("", sampleTree, readOnlyPaths, readWritePaths)
	assert.NilError(t, err)
	assert.Equal(t, len(pathValues), 5)

	for _, pathValue := range pathValues {
		//t.Logf("%v", pathValue)
		switch pathValue.Path {
		case
			"/system/logging/remote-servers/remote-server[host=h1]/config/source-address",
			"/system/logging/remote-servers/remote-server[host=h1]/selectors/selector[facility=ALL][severity=2]/config/severity",
			"/system/logging/console/selectors/selector[facility=3][severity=4]/config/severity":
			assert.Equal(t, pathValue.GetValue().GetType(), devicechange.ValueType_STRING, pathValue.Path)
			assert.Equal(t, len(pathValue.GetValue().GetTypeOpts()), 0)
		case
			"/system/logging/remote-servers/remote-server[host=h1]/selectors/selector[facility=ALL][severity=2]/config/facility":
			assert.Equal(t, pathValue.GetValue().GetType(), devicechange.ValueType_STRING, pathValue.Path)
			strValue := (*devicechange.TypedString)(pathValue.Value)
			assert.Equal(t, "ALL", strValue.String(), "expected IdentityRef to be translated")
		case
			"/system/logging/console/selectors/selector[facility=3][severity=4]/config/facility":
			assert.Equal(t, pathValue.GetValue().GetType(), devicechange.ValueType_STRING, pathValue.Path)
			strValue := (*devicechange.TypedString)(pathValue.Value)
			assert.Equal(t, "KERNEL", strValue.String(), "expected IdentityRef to be translated from '3' to KERNEL")
		default:
			t.Fatal("Unexpected path", pathValue.Path)
		}
	}
}
