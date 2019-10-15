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

// Package values tests for native to GNMI change conversion

package values

import (
	"github.com/onosproject/onos-config/pkg/store/change"
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"testing"
)

// Test_NativeChangeToGnmiChange tests conversion from an ONOS change to a GNMI change
func Test_NativeChangeToGnmiChange(t *testing.T) {
	// Some test data. One update, one remove
	testValues := []*types.Value{
		{
			Path:    "/path1/path2/path3",
			Value:   types.NewTypedValueString("value"),
			Removed: false,
		},
		{
			Path:    "/rpath1/rpath2/rpath3",
			Removed: true,
		},
	}
	testChange, err := change.NewChange(testValues, "Test Change")
	assert.NilError(t, err)

	//  Do the conversion
	request, err := NativeChangeToGnmiChange(testChange)
	assert.NilError(t, err)
	assert.Assert(t, request != nil)

	// Check that the Update portion of the GNMI request is correct
	assert.Equal(t, len(request.Update), 1, "Update array size is incorrect")
	assert.Equal(t, len(request.Update[0].Path.Elem), 3, "Path array size is incorrect")
	assert.Equal(t, request.Update[0].Path.Elem[0].Name, "path1", "Path[0] is incorrect")
	assert.Equal(t, request.Update[0].Path.Elem[1].Name, "path2", "Path[1] is incorrect")
	assert.Equal(t, request.Update[0].Path.Elem[2].Name, "path3", "Path[2] is incorrect")
	requestValueString, ok := request.Update[0].Val.Value.(*gnmi.TypedValue_StringVal)
	assert.Equal(t, ok, true, "request value has wrong type")
	assert.Equal(t, requestValueString.StringVal, "value", "Value is incorrect")

	// Check that the Delete portion of the request is correct
	assert.Equal(t, len(request.Delete), 1, "Delete array size is incorrect")
	assert.Equal(t, len(request.Delete[0].Elem), 3, "Path array size is incorrect")
	assert.Equal(t, request.Delete[0].Elem[0].Name, "rpath1", "Delete Path[0] is incorrect")
	assert.Equal(t, request.Delete[0].Elem[1].Name, "rpath2", "Delete Path[1] is incorrect")
	assert.Equal(t, request.Delete[0].Elem[2].Name, "rpath3", "Delete Path[2] is incorrect")
}
