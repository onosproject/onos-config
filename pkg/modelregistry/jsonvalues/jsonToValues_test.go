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
	ds1 "github.com/onosproject/config-models/modelplugin/devicesim-1.0.0/devicesim_1_0_0"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
	"io/ioutil"
	"testing"
)

func setUpJSONToValues(filename string) ([]byte, error) {
	sampleTree, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return sampleTree, nil
}

func setUpRwPaths() (modelregistry.ReadOnlyPathMap, modelregistry.ReadWritePathMap) {
	ds1Schema, _ := ds1.UnzipSchema()
	readOnlyPathsDeviceSim1, readWritePathsDeviceSim1 :=
		modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")

	return readOnlyPathsDeviceSim1, readWritePathsDeviceSim1
}

func Test_DecomposeTree(t *testing.T) {
	sampleTree, err := setUpJSONToValues("./testdata/sample-tree-from-devicesim.json")
	assert.NilError(t, err)
	assert.Assert(t, len(sampleTree) > 0, "Empty sample tree", len(sampleTree))

	ds1RoPaths, ds1RwPaths := setUpRwPaths()
	values, err := DecomposeJSONWithPaths(sampleTree, ds1RoPaths, ds1RwPaths)
	assert.NilError(t, err)
	assert.Equal(t, len(values), 25)

	for _, v := range values {
		t.Logf("%s %s\n", (*v).Path, (*v).GetValue().ValueToString())
		switch v.Path {
		case
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/address",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/aux-id",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/port",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/source-interface",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/transport",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/address",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/aux-id",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/port",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/source-interface",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/transport",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/address",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/aux-id",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/port",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/source-interface",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/transport",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/address",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/aux-id",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/port",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/source-interface",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/transport":
			assert.Equal(t, devicechange.ValueType_STRING, v.GetValue().GetType(), v.Path)
		case
			"/interfaces/interface[name=admin]/config/enabled":
			assert.Equal(t, devicechange.ValueType_BOOL, v.GetValue().GetType(), v.Path)
		case
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=0]/state/priority",
			"/system/openflow/controllers/controller[name=main]/connections/connection[aux-id=1]/state/priority",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=0]/state/priority",
			"/system/openflow/controllers/controller[name=second]/connections/connection[aux-id=1]/state/priority":
			assert.Equal(t, devicechange.ValueType_UINT, v.GetValue().GetType(), v.Path)
		default:
			t.Fatal("Unexpected jsonPath", v.Path)
		}
	}
}

// Deal with double (and more) key lists and config only
func Test_DecomposeTreeConfigOnly(t *testing.T) {
	sampleTree, err := setUpJSONToValues("./testdata/sample-tree-double-key.json")
	assert.NilError(t, err)
	assert.Assert(t, len(sampleTree) > 0, "Empty sample tree", len(sampleTree))

	_, ds1RwPaths := setUpRwPaths()
	values, err := DecomposeJSONWithPaths(sampleTree, nil, ds1RwPaths)
	assert.NilError(t, err)
	assert.Equal(t, len(values), 6)

	for _, v := range values {
		t.Logf("%s %s\n", (*v).Path, (*v).GetValue().ValueToString())
		switch v.Path {
		case
			"/system/logging/remote-servers/remote-server[host=h2]/config/host",
			"/system/logging/remote-servers/remote-server[host=h2]/config/source-address",
			"/system/logging/remote-servers/remote-server[host=h1]/selectors/selector[facility=f1][severity=s1]/config/facility",
			"/system/logging/remote-servers/remote-server[host=h1]/selectors/selector[facility=f1][severity=s1]/config/severity",
			"/system/logging/remote-servers/remote-server[host=h1]/config/host",
			"/system/logging/remote-servers/remote-server[host=h1]/config/source-address":
			assert.Equal(t, devicechange.ValueType_STRING, v.GetValue().GetType(), v.Path)
		default:
			t.Fatal("Unexpected jsonPath", v.Path)
		}
	}
}

func Test_findModelRwPathNoIndices(t *testing.T) {
	_, ds1RwPaths := setUpRwPaths()
	const jsonPath = "/system/logging/remote-servers/remote-server[0]/selectors/selector[0]/config/facility"
	const modelPath = "/system/logging/remote-servers/remote-server[host=0]/selectors/selector[facility=0][severity=0]/config/facility"

	rwElem, fullpath, ok := findModelRwPathNoIndices(ds1RwPaths, jsonPath)
	assert.Equal(t, true, ok)
	assert.Equal(t, modelPath, fullpath)
	assert.Assert(t, rwElem != nil, "rwElem map not expected to be nil")
	assert.Equal(t, devicechange.ValueType_STRING, rwElem.ValueType)
}

func Test_findModelRoPathNoIndices(t *testing.T) {
	ds1RoPaths, _ := setUpRwPaths()
	const jsonPath = "/system/logging/remote-servers/remote-server[0]/state/host"
	const modelPath = "/system/logging/remote-servers/remote-server[host=*]/state/host"

	roAttr, fullpath, ok := findModelRoPathNoIndices(ds1RoPaths, jsonPath)
	assert.Equal(t, true, ok)
	assert.Equal(t, modelPath, fullpath)
	assert.Assert(t, roAttr != nil, "roAttr map not expected to be nil")
	assert.Equal(t, devicechange.ValueType_STRING, roAttr.Datatype)
}

func Test_stripNamespace(t *testing.T) {
	const jsonPathNs = "/openconfig-system:system/openconfig-openflow:openflow/controllers/controller[0]/connections/connection[0]/aux-id"
	const jsonPath = "/system/openflow/controllers/controller[0]/connections/connection[0]/aux-id"

	stripped := stripNamespace(jsonPathNs)
	assert.Equal(t, jsonPath, stripped, "expected namespaces to have been removed")
}

func Test_indicesOfPath(t *testing.T) {
	ds1RoPaths, ds1RwPaths := setUpRwPaths()
	const jsonPath = "/system/logging/remote-servers/remote-server[0]/selectors/selector[0]"

	indices := indicesOfPath(ds1RoPaths, ds1RwPaths, jsonPath)
	assert.Equal(t, 3, len(indices))
	assert.Equal(t, "host", indices[0])
	assert.Equal(t, "facility", indices[1])
	assert.Equal(t, "severity", indices[2])
}

func Test_removePathIndices(t *testing.T) {
	const jsonPath = "/p/q/r[10]/s/t[20]/u/v[30]/w"
	const jsonPathRemovedIdx = "/p/q/r/s/t/u/v/w"
	noIndices := removePathIndices(jsonPath)
	assert.Equal(t, jsonPathRemovedIdx, noIndices)
}

func Test_extractIndexNames(t *testing.T) {
	const modelPath = "/p/q/r[a=*]/s/t[b=*][c=*]/u/v[d=*][e=*][f=*]/w"
	indexNames := extractIndexNames(modelPath)
	assert.Equal(t, 6, len(indexNames))
	assert.Equal(t, "a", indexNames[0])
	assert.Equal(t, "b", indexNames[1])
	assert.Equal(t, "c", indexNames[2])
	assert.Equal(t, "d", indexNames[3])
	assert.Equal(t, "e", indexNames[4])
	assert.Equal(t, "f", indexNames[5])
}

func Test_insertNumericalIndices(t *testing.T) {
	const modelPath = "/p/q/r[a=*]/s/t[b=*][c=*]/u/v[d=*][e=*][f=*]/w"
	const jsonPath = "/p/q/r[10]/s/t[20]/u/v[30]/w"
	const expected = "/p/q/r[a=10]/s/t[b=20][c=20]/u/v[d=30][e=30][f=30]/w"
	replaced, err := insertNumericalIndices(modelPath, jsonPath)
	assert.NilError(t, err, "unexpected error replacing wildcards")
	assert.Equal(t, expected, replaced)
}

func Test_insertNumericalIndicesNoIdx(t *testing.T) {
	const modelPath = "/l/m/n/o"
	const jsonPath = "/l/m/n/o"
	const expected = "/l/m/n/o"
	replaced, err := insertNumericalIndices(modelPath, jsonPath)
	assert.NilError(t, err, "unexpected error replacing wildcards")
	assert.Equal(t, expected, replaced)
}

func Test_prefixLength(t *testing.T) {
	const objPath = "/system/logging/remote-servers/remote-server[host=0]/selectors/selector[facility=0][severity=0]/config/facility"
	const parentPath = "/system/logging/remote-servers/remote-server[0]/selectors/selector"
	suffixLen := prefixLength(objPath, parentPath)
	assert.Equal(t, 95, suffixLen, "unexpected suffix len")
}

func Test_prefixLength2(t *testing.T) {
	const objPath = "/system/logging/remote-servers/remote-server[host=0]/config/host"
	const parentPath = "/system/logging/remote-servers/remote-server"
	suffixLen := prefixLength(objPath, parentPath)
	assert.Equal(t, 52, suffixLen, "unexpected suffix len")
}

func Test_replaceIndices(t *testing.T) {
	const modelPathNumericalIdx = "/p/q/r[a=10]/s/t[b=20][c=20]/u/v[d=30][e=30][f=30]/w"
	const modelPathExpected = "/p/q/r[a=12]/s/t[b=34][c=56]/u/v[d=78][e=9][f=10]/w"

	indices := make([]indexValue, 0)
	indices = append(indices, indexValue{"a", devicechange.NewTypedValueString("12")})
	indices = append(indices, indexValue{"b", devicechange.NewTypedValueUint64(34)})
	indices = append(indices, indexValue{"c", devicechange.NewTypedValueInt64(56)})
	indices = append(indices, indexValue{"d", devicechange.NewTypedValueString("78")})
	indices = append(indices, indexValue{"e", devicechange.NewTypedValueString("9")})
	indices = append(indices, indexValue{"f", devicechange.NewTypedValueString("10")})
	replaced, err := replaceIndices(modelPathNumericalIdx, len(modelPathNumericalIdx), indices)
	assert.NilError(t, err, "unexpected error replacing numbers")
	assert.Equal(t, modelPathExpected, replaced, "unexpected value after replacing numbers")
}

// Test with a missing index and a repeated index name
func Test_replaceIndices2(t *testing.T) {
	const modelPathNumericalIdx = "/p/q/r[name=10]/s/t[b=20][name=20]/u/v[d=30][e=30][f=30]/w/x[name=40]/y"
	const modelPathExpected = "/p/q/r[name=10]/s/t[b=20][name=20]/u/v[d=78][e=9][f=10]/w/x[name=40]/y"

	indices := make([]indexValue, 0)
	indices = append(indices, indexValue{"d", devicechange.NewTypedValueString("78")})
	indices = append(indices, indexValue{"e", devicechange.NewTypedValueString("9")})
	indices = append(indices, indexValue{"f", devicechange.NewTypedValueString("10")})

	replaced, err := replaceIndices(modelPathNumericalIdx, 57, indices)
	assert.NilError(t, err, "unexpected error replacing numbers")
	assert.Equal(t, modelPathExpected, replaced, "unexpected value after replacing numbers")
}
