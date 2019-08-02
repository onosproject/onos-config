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
	td2 "github.com/onosproject/onos-config/modelplugin/TestDevice-2.0.0/testdevice_2_0_0"
	"github.com/onosproject/onos-config/pkg/store/change"
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
	assert.Equal(t, leaf2cVt, change.ValueTypeSTRING)

	cont1b, cont1bOk := readOnlyPathsTestDevice2["/cont1b-state"]
	assert.Assert(t, cont1bOk, "expected to get /cont1b-state")
	assert.Equal(t, len(cont1b), 5, "expected /cont1b-state to have 5 subpaths")

	cont1bVt, cont1bVtOk := cont1b["/leaf2d"]
	assert.Assert(t, cont1bVtOk, "expected /cont1b-state to have subpath /leaf2d")
	assert.Equal(t, cont1bVt, change.ValueTypeUINT)

	l2bIdxVt, l2bIdxVtOk := cont1b["/list2b[index=*]/index"]
	assert.Assert(t, l2bIdxVtOk, "expected /cont1b-state to have subpath /list2b[index[*]/index")
	assert.Equal(t, l2bIdxVt, change.ValueTypeUINT)

	l2bLeaf3cVt, l2bLeaf3cVtOk := cont1b["/list2b[index=*]/leaf3c"]
	assert.Assert(t, l2bLeaf3cVtOk, "expected /cont1b-state to have subpath /list2b[index[*]/leaf3c")
	assert.Equal(t, l2bLeaf3cVt, change.ValueTypeSTRING)

	////////////////////////////////////////////////////
	/// Read write paths
	////////////////////////////////////////////////////
	readWritePathsKeys := PathsRW(readWritePathsTestDevice2)
	assert.Equal(t, len(readWritePathsKeys), 11)

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
			"/leafAtTopLevel":

		default:
			t.Fatal("Unexpected readWritePath", p)
		}
	}

	list2aName, list2aNameOk := readWritePathsTestDevice2["/cont1a/list2a[name=*]/name"]
	assert.Assert(t, list2aNameOk, "expected to get /cont1a/list2a[name=*]/name")
	assert.Equal(t, list2aName.ValueType, change.ValueTypeSTRING, "expected /cont1a/list2a[name=*]/name to be STRING")

	list2aTxp, list2aTxpOk := readWritePathsTestDevice2["/cont1a/list2a[name=*]/tx-power"]
	assert.Assert(t, list2aTxpOk, "expected to get /cont1a/list2a[name=*]/tx-power")
	assert.Equal(t, list2aTxp.ValueType, change.ValueTypeUINT, "expected /cont1a/list2a[name=*]/tx-power to be UINT")

	list2aRxp, list2aRxpOk := readWritePathsTestDevice2["/cont1a/list2a[name=*]/rx-power"]
	assert.Assert(t, list2aRxpOk, "expected to get /cont1a/list2a[name=*]/rx-power")
	assert.Equal(t, list2aRxp.ValueType, change.ValueTypeUINT, "expected /cont1a/list2a[name=*]/rx-power to be UINT")

	leaf1a, leaf1aOk := readWritePathsTestDevice2["/cont1a/leaf1a"]
	assert.Assert(t, leaf1aOk, "expected to get /cont1a/leaf1a")
	assert.Equal(t, leaf1a.ValueType, change.ValueTypeSTRING, "expected /cont1a/leaf1a to be STRING")

	leaf2a, leaf2aOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2a"]
	assert.Assert(t, leaf2aOk, "expected to get /cont1a/cont2a/leaf2a")
	assert.Equal(t, leaf2a.ValueType, change.ValueTypeUINT, "expected /cont1a/cont2a/leaf2a to be UINT")

	leaf2b, leaf2bOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2b"]
	assert.Assert(t, leaf2bOk, "expected to get /cont1a/cont2a/leaf2b")
	assert.Equal(t, leaf2b.ValueType, change.ValueTypeDECIMAL, "expected /cont1a/cont2a/leaf2b to be DECIMAL")

	leaf2d, leaf2dOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2d"]
	assert.Assert(t, leaf2dOk, "expected to get /cont1a/cont2a/leaf2d")
	assert.Equal(t, leaf2d.ValueType, change.ValueTypeDECIMAL, "expected /cont1a/cont2a/leaf2d to be DECIMAL")

	leaf2e, leaf2eOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2e"]
	assert.Assert(t, leaf2eOk, "expected to get /cont1a/cont2a/leaf2e")
	assert.Equal(t, leaf2e.ValueType, change.ValueTypeLeafListINT, "expected /cont1a/cont2a/leaf2d to be Leaf List INT")

	leaf2f, leaf2fOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2f"]
	assert.Assert(t, leaf2fOk, "expected to get /cont1a/cont2a/leaf2f")
	assert.Equal(t, leaf2f.ValueType, change.ValueTypeBYTES, "expected /cont1a/cont2a/leaf2f to be BYTES")

	leaf2g, leaf2gOk := readWritePathsTestDevice2["/cont1a/cont2a/leaf2g"]
	assert.Assert(t, leaf2gOk, "expected to get /cont1a/cont2a/leaf2g")
	assert.Equal(t, leaf2g.ValueType, change.ValueTypeBOOL, "expected /cont1a/cont2a/leaf2g to be BOOL")

	leafTopLevel, leafTopLevelOk := readWritePathsTestDevice2["/leafAtTopLevel"]
	assert.Assert(t, leafTopLevelOk, "expected to get /leafAtTopLevel")
	assert.Equal(t, leafTopLevel.ValueType, change.ValueTypeSTRING, "expected /leafAtTopLevel to be STRING")

}
