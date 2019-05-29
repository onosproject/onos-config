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

package gnmi

import (
	"context"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"strconv"
	"testing"

	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
)

// Test_doSingleSet shows how a value of 1 path can be set on a target
func Test_doSingleSet(t *testing.T) {
	server := setUp(false)
	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_StringVal{StringVal: "newValue2a"}
	value := gnmi.TypedValue{Value: &typedValue}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "Device1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	ext100Name := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  GnmiExtensionNetwkChangeID,
			Msg: []byte("TestChange"),
		},
	}

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Extension: []*gnmi_ext.Extension{&gnmi_ext.Extension{
			Ext: &ext100Name,
		}},
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 1)

	assert.Assert(t, setResponse.Message == nil, "Unexpected gnmi error message")

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_UPDATE.String())

	path := setResponse.Response[0].Path

	assert.Equal(t, path.Target, "Device1")

	assert.Equal(t, len(path.Elem), 3, "Expected 3 path elements")

	assert.Equal(t, path.Elem[0].Name, "cont1a")
	assert.Equal(t, path.Elem[1].Name, "cont2a")
	assert.Equal(t, path.Elem[2].Name, "leaf2a")

	// Check that an the network change ID is given in extension 10
	assert.Equal(t, len(setResponse.Extension), 1)

	extension := setResponse.Extension[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(GnmiExtensionNetwkChangeID))
	assert.Equal(t, string(extension.Msg), "TestChange")
}

// Test_do2SetsOnSameTarget shows how 2 paths can be changed on a target
func Test_do2SetsOnSameTarget(t *testing.T) {
	server := setUp(false)
	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)

	pathElemsRefs1, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	value1Str := gnmi.TypedValue_StringVal{StringVal: "newValue2b"}
	value := gnmi.TypedValue{Value: &value1Str}

	updatePath1 := gnmi.Path{Elem: pathElemsRefs1.Elem, Target: "localhost:10161"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath1, Val: &value})

	pathElemsRefs2, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2c"})
	value2Str := gnmi.TypedValue_StringVal{StringVal: "newValue2c"}
	value2 := gnmi.TypedValue{Value: &value2Str}

	updatePath2 := gnmi.Path{Elem: pathElemsRefs2.Elem, Target: "localhost:10161"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath2, Val: &value2})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 2)

	assert.Assert(t, setResponse.Message == nil, "Unexpected gnmi error message")

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_UPDATE.String())
	assert.Equal(t, setResponse.Response[1].Op.String(), gnmi.UpdateResult_UPDATE.String())

	// Response parts might not be in the same order because changes are stored in a map
	for _, res := range setResponse.Response {
		assert.Equal(t, res.Path.Target, "localhost:10161")
	}
}

// Test_do2SetsOnDiffTargets shows how paths on multiple targets can be Set at
// same time
func Test_do2SetsOnDiffTargets(t *testing.T) {
	server := setUp(false)
	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)

	// Make the same change to 2 targets
	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_StringVal{StringVal: "2ndValue2a"}
	value := gnmi.TypedValue{Value: &typedValue}

	updatePathTgt1 := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "localhost:10161"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePathTgt1, Val: &value})

	updatePathTgt2 := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "localhost:10162"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePathTgt2, Val: &value})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 2)

	assert.Assert(t, setResponse.Message == nil, "Unexpected gnmi error message")

	// It's a map, so the order of element's is not guaranteed
	for _, result := range setResponse.Response {
		assert.Equal(t, result.Op.String(), gnmi.UpdateResult_UPDATE.String())
		assert.Equal(t, result.Path.Elem[2].Name, "leaf2a")
	}
}

// Test_do2SetsOnOneTargetOneOnDiffTarget shows how multiple paths on multiple
// targets can be Set at same time
func Test_do2SetsOnOneTargetOneOnDiffTarget(t *testing.T) {
	server := setUp(false)
	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)

	// Make the same change to 2 targets
	pathElemsRefs2a, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	valueStr2a := gnmi.TypedValue_StringVal{StringVal: "3rdValue2a"}

	pathElemsRefs2b, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	valueStr2b := gnmi.TypedValue_StringVal{StringVal: "3rdValue2b"}

	pathElemsRefs2c, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2c"})

	// 2 changes on Target 1
	updatedPaths = append(updatedPaths, &gnmi.Update{
		Path: &gnmi.Path{Elem: pathElemsRefs2a.Elem, Target: "localhost:10161"},
		Val:  &gnmi.TypedValue{Value: &valueStr2a},
	})
	updatedPaths = append(updatedPaths, &gnmi.Update{
		Path: &gnmi.Path{Elem: pathElemsRefs2b.Elem, Target: "localhost:10161"},
		Val:  &gnmi.TypedValue{Value: &valueStr2b},
	})

	// 2 changes on Target 2 - one of them is a delete
	updatedPaths = append(updatedPaths, &gnmi.Update{
		Path: &gnmi.Path{Elem: pathElemsRefs2a.Elem, Target: "localhost:10162"},
		Val:  &gnmi.TypedValue{Value: &valueStr2a},
	})
	deletePaths = append(deletePaths,
		&gnmi.Path{Elem: pathElemsRefs2c.Elem, Target: "localhost:10162"})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 4)

	// The order of response items is not ordered because it is held in a map
	for _, res := range setResponse.Response {
		if res.Op.String() == gnmi.UpdateResult_DELETE.String() {
			assert.Equal(t, res.Path.Target, "localhost:10162")
		}
	}
}

// Test_doDuplicateSetSingleTarget shows how duplicate combineation of paths on
// a single target fails
func Test_doDuplicateSetSingleTarget(t *testing.T) {
	server := setUp(false)
	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)

	// Make 2 changes
	pathElemsRefs2a, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	valueStr2a := gnmi.TypedValue_StringVal{StringVal: "4thValue2a"}
	updatedPaths = append(updatedPaths,
		&gnmi.Update{
			Path: &gnmi.Path{
				Elem:   pathElemsRefs2a.Elem,
				Target: "localhost:10161",
			},
			Val: &gnmi.TypedValue{Value: &valueStr2a},
		})

	pathElemsRefs2b, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	valueStr2b := gnmi.TypedValue_StringVal{StringVal: "4thValue2a"}
	updatedPaths = append(updatedPaths,
		&gnmi.Update{
			Path: &gnmi.Path{
				Elem:   pathElemsRefs2b.Elem,
				Target: "localhost:10161",
			},
			Val: &gnmi.TypedValue{Value: &valueStr2b},
		})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 2)

	assert.Assert(t, setResponse.Message == nil, "Unexpected gnmi error message")

	/////////////////////////////////////////////////////////////////////////////
	// Now try again - should fail
	/////////////////////////////////////////////////////////////////////////////
	setResponse, setError = server.Set(context.Background(), &setRequest)

	assert.ErrorContains(t, setError, "duplicate", "Expected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse == nil, "Expected setResponse to be nil")
}

// Test_doDuplicateSet2Targets shows how if all paths on all targets are
// duplicates it should fail
func Test_doDuplicateSet2Targets(t *testing.T) {
	server := setUp(false)
	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)

	// Make 2 changes
	pathElemsRefs2a, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	valueStr2a := gnmi.TypedValue_StringVal{StringVal: "5thValue2a"}
	updatedPaths = append(updatedPaths,
		&gnmi.Update{
			Path: &gnmi.Path{
				Elem:   pathElemsRefs2a.Elem,
				Target: "localhost:10161",
			},
			Val: &gnmi.TypedValue{Value: &valueStr2a},
		})
	updatedPaths = append(updatedPaths,
		&gnmi.Update{
			Path: &gnmi.Path{
				Elem:   pathElemsRefs2a.Elem,
				Target: "localhost:10162",
			},
			Val: &gnmi.TypedValue{Value: &valueStr2a},
		})

	pathElemsRefs2b, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	valueStr2b := gnmi.TypedValue_StringVal{StringVal: "5thValue2a"}
	updatedPaths = append(updatedPaths,
		&gnmi.Update{
			Path: &gnmi.Path{
				Elem:   pathElemsRefs2b.Elem,
				Target: "localhost:10161",
			},
			Val: &gnmi.TypedValue{Value: &valueStr2b},
		})

	updatedPaths = append(updatedPaths,
		&gnmi.Update{
			Path: &gnmi.Path{
				Elem:   pathElemsRefs2b.Elem,
				Target: "localhost:10162",
			},
			Val: &gnmi.TypedValue{Value: &valueStr2b},
		})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 4)

	assert.Assert(t, setResponse.Message == nil, "Unexpected gnmi error message")

	/////////////////////////////////////////////////////////////////////////////
	// Now try again - should fail
	/////////////////////////////////////////////////////////////////////////////
	setResponse, setError = server.Set(context.Background(), &setRequest)

	assert.ErrorContains(t, setError, "duplicate", "Expected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse == nil, "Expected setResponse to be nil")
}

// Test_doDuplicateSet1TargetNewOnOther shows how when there are dups on some
// targets but non dups on other targets the dups can be quietly ignored
// Note how the SetResponse does not include the dups
func Test_doDuplicateSet1TargetNewOnOther(t *testing.T) {
	server := setUp(false)
	// Make 2 changes
	pathElemsRefs2a, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	valueStr2a := gnmi.TypedValue_StringVal{StringVal: "6thValue2a"}

	update1a := gnmi.Update{
		Path: &gnmi.Path{
			Elem:   pathElemsRefs2a.Elem,
			Target: "localhost:10161",
		},
		Val: &gnmi.TypedValue{Value: &valueStr2a},
	}

	update2a := gnmi.Update{
		Path: &gnmi.Path{
			Elem:   pathElemsRefs2a.Elem,
			Target: "localhost:10162",
		},
		Val: &gnmi.TypedValue{Value: &valueStr2a},
	}

	pathElemsRefs2b, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	valueStr2b := gnmi.TypedValue_StringVal{StringVal: "6thValue2a"}

	update1b := gnmi.Update{
		Path: &gnmi.Path{
			Elem:   pathElemsRefs2b.Elem,
			Target: "localhost:10161",
		},
		Val: &gnmi.TypedValue{Value: &valueStr2b},
	}

	update2b := gnmi.Update{
		Path: &gnmi.Path{
			Elem:   pathElemsRefs2b.Elem,
			Target: "localhost:10162",
		},
		Val: &gnmi.TypedValue{Value: &valueStr2b},
	}

	var setRequest = gnmi.SetRequest{
		Delete:  make([]*gnmi.Path, 0),
		Replace: make([]*gnmi.Update, 0),
		Update:  []*gnmi.Update{&update1a, &update2a, &update1b, &update2b},
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 4)

	assert.Assert(t, setResponse.Message == nil, "Unexpected gnmi error message")

	/////////////////////////////////////////////////////////////////////////////
	// Now try again - should NOT fail as target 2 is not duplicate
	/////////////////////////////////////////////////////////////////////////////
	valueStr2aCh := gnmi.TypedValue_StringVal{StringVal: "6thValue2aChanged"}
	update2b1 := gnmi.Update{
		Path: &gnmi.Path{
			Elem:   pathElemsRefs2b.Elem,
			Target: "localhost:10162",
		},
		Val: &gnmi.TypedValue{Value: &valueStr2aCh},
	}

	var setRequest2 = gnmi.SetRequest{
		Delete:  make([]*gnmi.Path, 0),
		Replace: make([]*gnmi.Update, 0),
		Update:  []*gnmi.Update{&update1a, &update2a, &update1b, &update2b1},
	}

	setResponse, setError = server.Set(context.Background(), &setRequest2)

	assert.NilError(t, setError, "Unexpected error doing Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to be not nil")

	// Even though we send 4 down, should only get 2 in response because those on
	// target 10161 were duplicates - quietly ignore duplicates where some valid
	// changes are made
	assert.Equal(t, len(setResponse.Response), 2)
}
