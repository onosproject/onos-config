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
	"testing"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	td1 "github.com/onosproject/config-models/modelplugin/testdevice-1.0.0/testdevice_1_0_0"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
)

func Test_SchemaTestDevice1(t *testing.T) {
	td1Schema, _ := td1.UnzipSchema()
	readOnlyPathsTestDevice1, readWritePathsTestDevice1 :=
		ExtractPaths(td1Schema["Device"], yang.TSUnset, "", "")

	////////////////////////////////////////////////////
	/// Read only paths
	////////////////////////////////////////////////////
	readOnlyPathsKeys := Paths(readOnlyPathsTestDevice1)
	assert.Equal(t, len(readOnlyPathsKeys), 6)
	// Can be in any order
	for _, p := range readOnlyPathsKeys {
		switch p {
		case
			"/cont1b-state",
			"/cont1a/cont2a/leaf2c",
			"/cont1a/list2a[name=*]",
			"/cont1a/list4[id=*]",
			"/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]",
			"/cont1a/list5[key1=*][key2=*]":
		default:
			t.Fatal("Unexpected readOnlyPath", p)
		}
	}

	leaf2c, leaf2cOk := readOnlyPathsTestDevice1["/cont1a/cont2a/leaf2c"]
	assert.Assert(t, leaf2cOk, "expected to get /cont1a/cont2a/leaf2c")
	assert.Equal(t, len(leaf2c), 1, "expected /cont1a/cont2a/leaf2c to have only 1 subpath")
	leaf2cVt, leaf2cVtOk := leaf2c["/"]
	assert.Assert(t, leaf2cVtOk, "expected /cont1a/cont2a/leaf2c to have subpath /")
	assert.Equal(t, leaf2cVt.ValueType, configapi.ValueType_STRING)
	assert.Equal(t, leaf2cVt.Description, "Read only leaf inside Container 2a")
	assert.Equal(t, leaf2cVt.Units, "")

	cont1b, cont1bOk := readOnlyPathsTestDevice1["/cont1b-state"]
	assert.Assert(t, cont1bOk, "expected to get /cont1b-state")
	assert.Equal(t, len(cont1b), 3, "expected /cont1b-state to have 3 subpaths")

	cont1bVt, cont1bVtOk := cont1b["/leaf2d"]
	assert.Assert(t, cont1bVtOk, "expected /cont1b-state to have subpath /leaf2d")
	assert.Equal(t, cont1bVt.ValueType, configapi.ValueType_UINT)

	l2bIdxVt, l2bIdxVtOk := cont1b["/list2b[index=*]/index"]
	assert.Assert(t, l2bIdxVtOk, "expected /cont1b-state to have subpath /list2b[index[*]/index")
	assert.Assert(t, l2bIdxVt.IsAKey)
	assert.Equal(t, l2bIdxVt.AttrName, "index")
	assert.Equal(t, l2bIdxVt.ValueType, configapi.ValueType_UINT)

	l2bLeaf3cVt, l2bLeaf3cVtOk := cont1b["/list2b[index=*]/leaf3c"]
	assert.Assert(t, l2bLeaf3cVtOk, "expected /cont1b-state to have subpath /list2b[index[*]/leaf3c")
	assert.Equal(t, l2bLeaf3cVt.AttrName, "leaf3c")
	assert.Assert(t, !l2bLeaf3cVt.IsAKey)
	assert.Equal(t, l2bLeaf3cVt.ValueType, configapi.ValueType_STRING)

	////////////////////////////////////////////////////
	/// Read write paths
	////////////////////////////////////////////////////
	readWritePathsKeys := PathsRW(readWritePathsTestDevice1)
	assert.Equal(t, len(readWritePathsKeys), 21)

	// Can be in any order
	for _, p := range readWritePathsKeys {
		switch p {
		case
			"/cont1a/leaf1a",
			"/cont1a/list2a[name=*]/name",
			"/cont1a/list2a[name=*]/tx-power",
			"/cont1a/list2a[name=*]/ref2d",
			"/cont1a/list2a[name=*]/range-min",
			"/cont1a/list2a[name=*]/range-max",
			"/cont1a/cont2a/leaf2a",
			"/cont1a/cont2a/leaf2b",
			"/cont1a/cont2a/leaf2d",
			"/cont1a/cont2a/leaf2e",
			"/cont1a/cont2a/leaf2f",
			"/cont1a/cont2a/leaf2g",
			"/cont1a/list4[id=*]/id",
			"/cont1a/list4[id=*]/leaf4b",
			"/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey1",
			"/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey2",
			"/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/displayname",
			"/cont1a/list5[key1=*][key2=*]/key1",
			"/cont1a/list5[key1=*][key2=*]/key2",
			"/cont1a/list5[key1=*][key2=*]/leaf5a",
			"/leafAtTopLevel":

		default:
			t.Fatal("Unexpected readWritePath", p)
		}
	}

	list2aName, list2aNameOk := readWritePathsTestDevice1["/cont1a/list2a[name=*]/name"]
	assert.Assert(t, list2aNameOk, "expected to get /cont1a/list2a[name=*]/name")
	assert.Equal(t, list2aName.ValueType, configapi.ValueType_STRING, "expected /cont1a/list2a[name=*]/name to be STRING")
	assert.Equal(t, len(list2aName.Length), 1, "expected 1 length terms")
	assert.Equal(t, list2aName.Length[0], "4..8", "expected length 0 to be 4..8")

	list2aTxp, list2aTxpOk := readWritePathsTestDevice1["/cont1a/list2a[name=*]/tx-power"]
	assert.Assert(t, list2aTxpOk, "expected to get /cont1a/list2a[name=*]/tx-power")
	assert.Equal(t, list2aTxp.ValueType, configapi.ValueType_UINT, "expected /cont1a/list2a[name=*]/tx-power to be UINT")
	assert.Equal(t, len(list2aTxp.Range), 1, "expected 1 range terms")
	assert.Equal(t, list2aTxp.Range[0], "1..20", "expected range 0 to be 1..20")

	leaf1a, leaf1aOk := readWritePathsTestDevice1["/cont1a/leaf1a"]
	assert.Assert(t, leaf1aOk, "expected to get /cont1a/leaf1a")
	assert.Equal(t, leaf1a.ValueType, configapi.ValueType_STRING, "expected /cont1a/leaf1a to be STRING")

	leaf2a, leaf2aOk := readWritePathsTestDevice1["/cont1a/cont2a/leaf2a"]
	assert.Assert(t, leaf2aOk, "expected to get /cont1a/cont2a/leaf2a")
	assert.Equal(t, leaf2a.ValueType, configapi.ValueType_UINT, "expected /cont1a/cont2a/leaf2a to be UINT")
	assert.Equal(t, leaf2a.Default, "2", "expected default 2")
	assert.Equal(t, leaf2a.Description, "Numeric leaf inside Container 2a")
	assert.Equal(t, leaf2a.Units, "", "expected units to be blank - boo for YGOT - hopefully should get units sometime")
	assert.Equal(t, len(leaf2a.Range), 2, "expected 2 range terms")
	assert.Equal(t, leaf2a.Range[0], "1..3", "expected range 0 to be 1..3")
	assert.Equal(t, leaf2a.Range[1], "11..13", "expected range 1 to be 11..13")

	leaf2b, leaf2bOk := readWritePathsTestDevice1["/cont1a/cont2a/leaf2b"]
	assert.Assert(t, leaf2bOk, "expected to get /cont1a/cont2a/leaf2b")
	assert.Equal(t, leaf2b.ValueType, configapi.ValueType_DECIMAL, "expected /cont1a/cont2a/leaf2b to be DECIMAL")
	assert.Equal(t, len(leaf2b.Range), 1, "expected 1 range term")
	assert.Equal(t, leaf2b.Range[0], "0.001..2.000", "expected range 0 to be 0.001..2.000")

	leaf2d, leaf2dOk := readWritePathsTestDevice1["/cont1a/cont2a/leaf2d"]
	assert.Assert(t, leaf2dOk, "expected to get /cont1a/cont2a/leaf2d")
	assert.Equal(t, leaf2d.ValueType, configapi.ValueType_DECIMAL, "expected /cont1a/cont2a/leaf2d to be DECIMAL")

	leaf2e, leaf2eOk := readWritePathsTestDevice1["/cont1a/cont2a/leaf2e"]
	assert.Assert(t, leaf2eOk, "expected to get /cont1a/cont2a/leaf2e")
	assert.Equal(t, leaf2e.ValueType, configapi.ValueType_LEAFLIST_INT, "expected /cont1a/cont2a/leaf2d to be Leaf List INT")
	assert.Equal(t, len(leaf2e.Range), 1, "expected 1 range term")
	assert.Equal(t, leaf2e.Range[0], "-100..200", "expected range 0 to be -100..200")

	leaf2f, leaf2fOk := readWritePathsTestDevice1["/cont1a/cont2a/leaf2f"]
	assert.Assert(t, leaf2fOk, "expected to get /cont1a/cont2a/leaf2f")
	assert.Equal(t, leaf2f.ValueType, configapi.ValueType_BYTES, "expected /cont1a/cont2a/leaf2f to be BYTES")

	leaf2g, leaf2gOk := readWritePathsTestDevice1["/cont1a/cont2a/leaf2g"]
	assert.Assert(t, leaf2gOk, "expected to get /cont1a/cont2a/leaf2g")
	assert.Equal(t, leaf2g.ValueType, configapi.ValueType_BOOL, "expected /cont1a/cont2a/leaf2g to be BOOL")

	leafTopLevel, leafTopLevelOk := readWritePathsTestDevice1["/leafAtTopLevel"]
	assert.Assert(t, leafTopLevelOk, "expected to get /leafAtTopLevel")
	assert.Equal(t, leafTopLevel.ValueType, configapi.ValueType_STRING, "expected /leafAtTopLevel to be STRING")

	list4ID, list4IDOk := readWritePathsTestDevice1["/cont1a/list4[id=*]/id"]
	assert.Assert(t, list4IDOk, "expected to get /cont1a/list4[id=*]/id")
	assert.Assert(t, list4ID.IsAKey)
	assert.Equal(t, list4ID.ValueType, configapi.ValueType_STRING, "expected //cont1a/list4[id=*]/id to be STRING")

	list4Leaf4b, list4Leaf4bOk := readWritePathsTestDevice1["/cont1a/list4[id=*]/leaf4b"]
	assert.Assert(t, list4Leaf4bOk, "expected to get /cont1a/list4[id=*]/leaf4b")
	assert.Assert(t, !list4Leaf4b.IsAKey)
	assert.Equal(t, list4Leaf4b.ValueType, configapi.ValueType_STRING, "expected //cont1a/list4[id=*]/leaf4b to be STRING")

	list4aFkey1, list4aFkey1Ok := readWritePathsTestDevice1["/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey1"]
	assert.Assert(t, list4aFkey1Ok, "expected to get /cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey1")
	assert.Assert(t, list4aFkey1.IsAKey)
	assert.Equal(t, list4aFkey1.ValueType, configapi.ValueType_STRING, "expected //cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey1 to be STRING")

	list4aFkey2, list4aFkey2Ok := readWritePathsTestDevice1["/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey2"]
	assert.Assert(t, list4aFkey2Ok, "expected to get /cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey2")
	assert.Assert(t, list4aFkey2.IsAKey)
	assert.Equal(t, list4aFkey2.ValueType, configapi.ValueType_STRING, "expected //cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey2 to be STRING")

	list4aDispName, list4aDispNameOk := readWritePathsTestDevice1["/cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/displayname"]
	assert.Assert(t, list4aDispNameOk, "expected to get /cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey1")
	assert.Assert(t, !list4aDispName.IsAKey)
	assert.Equal(t, list4aDispName.ValueType, configapi.ValueType_STRING, "expected //cont1a/list4[id=*]/list4a[fkey1=*][fkey2=*]/fkey1 to be STRING")

	list5Key1, list5Key1Ok := readWritePathsTestDevice1["/cont1a/list5[key1=*][key2=*]/key1"]
	assert.Assert(t, list5Key1Ok, "expected to get /cont1a/list5[key1=*][key2=*]/key1")
	assert.Assert(t, list5Key1.IsAKey)
	assert.Equal(t, list5Key1.ValueType, configapi.ValueType_STRING, "expected //cont1a/list5[key1=*]/key1 to be STRING")

	list5Key2, list5Key2Ok := readWritePathsTestDevice1["/cont1a/list5[key1=*][key2=*]/key2"]
	assert.Assert(t, list5Key2Ok, "expected to get /cont1a/list5[key1=*][key2=*]/key2")
	assert.Assert(t, list5Key2.IsAKey)
	assert.Equal(t, list5Key2.ValueType, configapi.ValueType_UINT, "expected //cont1a/list5[key1=*]/key1 to be UINT")

	list5Leaf5a, list5Leaf5aOk := readWritePathsTestDevice1["/cont1a/list5[key1=*][key2=*]/leaf5a"]
	assert.Assert(t, list5Leaf5aOk, "expected to get /cont1a/list5[key1=*][key2=*]/leaf5a")
	assert.Assert(t, !list5Leaf5a.IsAKey)
	assert.Equal(t, list5Leaf5a.ValueType, configapi.ValueType_STRING, "expected //cont1a/list5[key1=*][key2=*]/leaf5a to be STRING")

}
