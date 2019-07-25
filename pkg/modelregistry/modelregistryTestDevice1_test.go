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
	td1 "github.com/onosproject/onos-config/modelplugin/TestDevice-1.0.0/testdevice_1_0_0"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
	"testing"
)

func Test_SchemaTestDevice1(t *testing.T) {
	td1Schema, _ := td1.UnzipSchema()
	readOnlyPathsTestDevice1 := ExtractReadOnlyPaths(td1Schema["Device"], yang.TSUnset, "", "", "")

	readOnlyPathsKeys := Paths(readOnlyPathsTestDevice1)
	assert.Equal(t, len(readOnlyPathsKeys), 2)
	// Can be in any order
	for _, p := range readOnlyPathsKeys {
		switch p {
		case
			"/test1:cont1b-state",
			"/test1:cont1a/cont2a/leaf2c":

		default:
			t.Fatal("Unexpected readOnlyPath", p)
		}
	}

	leaf2c, leaf2cOk := readOnlyPathsTestDevice1["/test1:cont1a/cont2a/leaf2c"]
	assert.Assert(t, leaf2cOk, "expected to get /test1:cont1a/cont2a/leaf2c")
	assert.Equal(t, len(leaf2c), 1, "expected /test1:cont1a/cont2a/leaf2c to have only 1 subpath")
	leaf2cVt, leaf2cVtOk := leaf2c["/"]
	assert.Assert(t, leaf2cVtOk, "expected /test1:cont1a/cont2a/leaf2c to have subpath /")
	assert.Equal(t, leaf2cVt, change.ValueTypeSTRING)

	cont1b, cont1bOk := readOnlyPathsTestDevice1["/test1:cont1b-state"]
	assert.Assert(t, cont1bOk, "expected to get /test1:cont1b-state")
	assert.Equal(t, len(cont1b), 3, "expected /test1:cont1b-state to have 3 subpaths")

	cont1bVt, cont1bVtOk := cont1b["/leaf2d"]
	assert.Assert(t, cont1bVtOk, "expected /test1:cont1b-state to have subpath /leaf2d")
	assert.Equal(t, cont1bVt, change.ValueTypeUINT)

	l2bIdxVt, l2bIdxVtOk := cont1b["/list2b[index=*]/index"]
	assert.Assert(t, l2bIdxVtOk, "expected /test1:cont1b-state to have subpath /list2b[index[*]/index")
	assert.Equal(t, l2bIdxVt, change.ValueTypeUINT)

	l2bLeaf3cVt, l2bLeaf3cVtOk := cont1b["/list2b[index=*]/leaf3c"]
	assert.Assert(t, l2bLeaf3cVtOk, "expected /test1:cont1b-state to have subpath /list2b[index[*]/leaf3c")
	assert.Equal(t, l2bLeaf3cVt, change.ValueTypeSTRING)
}
