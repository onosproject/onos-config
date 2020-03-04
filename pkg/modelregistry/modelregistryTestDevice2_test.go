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
	td2 "github.com/onosproject/config-models/modelplugin/testdevice-2.0.0/testdevice_2_0_0"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
	"testing"
)

func Test_SchemaTestDevice2(t *testing.T) {
	td2Schema, _ := td2.UnzipSchema()
	readOnlyPathsTestDevice2, readWritePathsTestDevice2 :=
		ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")

	readOnlyPathsKeys := Paths(readOnlyPathsTestDevice2)
	assert.Equal(t, len(readOnlyPathsKeys), 2)
	// Can be in any order
	for _, p := range readOnlyPathsKeys {
		switch p {
		case
			"/cont1b-state",
			"/cont1a/cont2a/leaf2c":

		default:
			t.Fatal("Unexpected readOnlyPath", p)
		}
	}

	leaf2c, leaf2cOk := readOnlyPathsTestDevice2["/cont1a/cont2a/leaf2c"]
	assert.Assert(t, leaf2cOk, "expected to get /cont1a/cont2a/leaf2c")
	assert.Equal(t, len(leaf2c), 1, "expected /cont1a/cont2a/leaf2c to have only 1 subpath")
	leaf2cVt, leaf2cVtOk := leaf2c["/"]
	assert.Assert(t, leaf2cVtOk, "expected /cont1a/cont2a/leaf2c to have subpath /")
	assert.Equal(t, leaf2cVt.Datatype, devicechange.ValueType_STRING)

	cont1b, cont1bOk := readOnlyPathsTestDevice2["/cont1b-state"]
	assert.Assert(t, cont1bOk, "expected to get /cont1b-state")
	assert.Equal(t, len(cont1b), 5, "expected /cont1b-state to have 5 subpaths")

	cont1bVt, cont1bVtOk := cont1b["/leaf2d"]
	assert.Assert(t, cont1bVtOk, "expected /cont1b-state to have subpath /leaf2d")
	assert.Equal(t, cont1bVt.Datatype, devicechange.ValueType_UINT)

	l2bIdxVt, l2bIdxVtOk := cont1b["/list2b[index=*]/index"]
	assert.Assert(t, l2bIdxVtOk, "expected /cont1b-state to have subpath /list2b[index[*]/index")
	assert.Equal(t, l2bIdxVt.Datatype, devicechange.ValueType_UINT)

	l2bLeaf3cVt, l2bLeaf3cVtOk := cont1b["/list2b[index=*]/leaf3c"]
	assert.Assert(t, l2bLeaf3cVtOk, "expected /cont1b-state to have subpath /list2b[index[*]/leaf3c")
	assert.Equal(t, l2bLeaf3cVt.Datatype, devicechange.ValueType_STRING)

	////////////////////////////////////////////////////
	/// Read write paths
	////////////////////////////////////////////////////
	readWritePathsKeys := PathsRW(readWritePathsTestDevice2)
	assert.Equal(t, len(readWritePathsKeys), 15)

	// Can be in any order
	for _, p := range readWritePathsKeys {
		switch p {
		case
			"/cont1a/leaf1a",
			"/cont1a/list2a[name=*]/name",
			"/cont1a/list2a[name=*]/tx-power",
			"/cont1a/list2a[name=*]/rx-power",
			"/cont1a/cont2a/leaf2a",
			"/cont1a/cont2a/leaf2b",
			"/cont1a/cont2a/leaf2d",
			"/cont1a/cont2a/leaf2e",
			"/cont1a/cont2a/leaf2f",
			"/cont1a/cont2a/leaf2g",
			"/leafAtTopLevel",
			"/cont1a/cont2d/beer",
			"/cont1a/cont2d/pretzel",
			"/cont1a/cont2d/chocolate",
			"/cont1a/cont2d/leaf2d3c":

		default:
			t.Fatal("Unexpected readWritePath", p)
		}
	}

	list2aName, list2aNameOk := readWritePathsTestDevice2["/cont1a/list2a[name=*]/name"]
	assert.Assert(t, list2aNameOk, "expected to get /cont1a/list2a[name=*]/name")
	assert.Equal(t, list2aName.ValueType, devicechange.ValueType_STRING, "expected /cont1a/list2a[name=*]/name to be STRING")

	list2aTxp, list2aTxpOk := readWritePathsTestDevice2["/cont1a/list2a[name=*]/tx-power"]
	assert.Assert(t, list2aTxpOk, "expected to get /cont1a/list2a[name=*]/tx-power")
	assert.Equal(t, list2aTxp.ValueType, devicechange.ValueType_UINT, "expected /cont1a/list2a[name=*]/tx-power to be UINT")

	list2aRxp, list2aRxpOk := readWritePathsTestDevice2["/cont1a/list2a[name=*]/rx-power"]
	assert.Assert(t, list2aRxpOk, "expected to get /cont1a/list2a[name=*]/rx-power")
	assert.Equal(t, list2aRxp.ValueType, devicechange.ValueType_UINT, "expected /cont1a/list2a[name=*]/rx-power to be UINT")

	leaf1a, leaf1aOk := readWritePathsTestDevice2["/cont1a/leaf1a"]
	assert.Assert(t, leaf1aOk, "expected to get /cont1a/leaf1a")
	assert.Equal(t, leaf1a.ValueType, devicechange.ValueType_STRING, "expected /cont1a/leaf1a to be STRING")

	leaf2a, leaf2aOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2a"]
	assert.Assert(t, leaf2aOk, "expected to get /cont1a/cont2a/leaf2a")
	assert.Equal(t, leaf2a.ValueType, devicechange.ValueType_UINT, "expected /cont1a/cont2a/leaf2a to be UINT")

	leaf2b, leaf2bOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2b"]
	assert.Assert(t, leaf2bOk, "expected to get /cont1a/cont2a/leaf2b")
	assert.Equal(t, leaf2b.ValueType, devicechange.ValueType_DECIMAL, "expected /cont1a/cont2a/leaf2b to be DECIMAL")

	leaf2d, leaf2dOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2d"]
	assert.Assert(t, leaf2dOk, "expected to get /cont1a/cont2a/leaf2d")
	assert.Equal(t, leaf2d.ValueType, devicechange.ValueType_DECIMAL, "expected /cont1a/cont2a/leaf2d to be DECIMAL")

	leaf2e, leaf2eOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2e"]
	assert.Assert(t, leaf2eOk, "expected to get /cont1a/cont2a/leaf2e")
	assert.Equal(t, leaf2e.ValueType, devicechange.ValueType_LEAFLIST_INT, "expected /cont1a/cont2a/leaf2d to be Leaf List INT")

	leaf2f, leaf2fOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2f"]
	assert.Assert(t, leaf2fOk, "expected to get /cont1a/cont2a/leaf2f")
	assert.Equal(t, leaf2f.ValueType, devicechange.ValueType_BYTES, "expected /cont1a/cont2a/leaf2f to be BYTES")

	leaf2g, leaf2gOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2g"]
	assert.Assert(t, leaf2gOk, "expected to get /cont1a/cont2a/leaf2g")
	assert.Equal(t, leaf2g.ValueType, devicechange.ValueType_BOOL, "expected /cont1a/cont2a/leaf2g to be BOOL")

	leafTopLevel, leafTopLevelOk := readWritePathsTestDevice2["/leafAtTopLevel"]
	assert.Assert(t, leafTopLevelOk, "expected to get /leafAtTopLevel")
	assert.Equal(t, leafTopLevel.ValueType, devicechange.ValueType_STRING, "expected /leafAtTopLevel to be STRING")

	leaf2d3c, leaf2d3cOk := readWritePathsTestDevice2["/cont1a/cont2d/leaf2d3c"]
	assert.Assert(t, leaf2d3cOk, "expected to get /cont1a/cont2d/leaf2d3c")
	assert.Equal(t, leaf2d3c.ValueType, devicechange.ValueType_STRING, "expected /cont1a/cont2d/leaf2d3c to be STRING")

	beer, beerOk := readWritePathsTestDevice2["/cont1a/cont2d/beer"]
	assert.Assert(t, beerOk, "expected to get /cont1a/cont2d/beer")
	assert.Equal(t, beer.ValueType, devicechange.ValueType_EMPTY, "expected /cont1a/cont2d/beer to be EMPTY")

	pretzel, pretzelOk := readWritePathsTestDevice2["/cont1a/cont2d/pretzel"]
	assert.Assert(t, pretzelOk, "expected to get /cont1a/cont2d/pretzel")
	assert.Equal(t, pretzel.ValueType, devicechange.ValueType_EMPTY, "expected /cont1a/cont2d/pretzel to be EMPTY")

	chocolate, chocolateOk := readWritePathsTestDevice2["/cont1a/cont2d/chocolate"]
	assert.Assert(t, chocolateOk, "expected to get /cont1a/cont2d/chocolate")
	assert.Equal(t, chocolate.ValueType, devicechange.ValueType_STRING, "expected /cont1a/cont2d/chocolate to be STRING")
}
