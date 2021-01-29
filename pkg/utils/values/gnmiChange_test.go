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
	"encoding/json"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	Test1Cont1ACont2ALeaf2C = "/cont1a/cont2a/leaf2c"
	Test1Cont1AList2ATxout2 = "/cont1a/list2a[name=txout2]"
	ValueLeaf2CDef          = "def"
)

// Test_NativeChangeToGnmiChange tests conversion from an ONOS change to a GNMI change
func Test_NativeChangeToGnmiChange(t *testing.T) {
	// Some test data. One update, one remove
	testValues := []*devicechange.ChangeValue{
		{
			Path:    "/path1/path2/path3",
			Value:   devicechange.NewTypedValueString("value"),
			Removed: false,
		},
		{
			Path:    "/rpath1/rpath2/rpath3",
			Removed: true,
		},
	}
	testChange := &devicechange.Change{
		DeviceID:      "Device1",
		DeviceVersion: "Device1-1.0.0",
		DeviceType:    "devicesim",
		Values:        testValues,
	}

	//  Do the conversion
	request, err := NativeChangeToGnmiChange(testChange)
	assert.NoError(t, err)
	assert.NotNil(t, request)

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

func Test_convertChangeToGnmi(t *testing.T) {
	// Some test data. One update, one remove
	testValues := []*devicechange.ChangeValue{
		{
			Path:    Test1Cont1ACont2ALeaf2C,
			Value:   devicechange.NewTypedValueString(ValueLeaf2CDef),
			Removed: false,
		},
		{
			Path:    Test1Cont1AList2ATxout2,
			Removed: true,
		},
	}
	change3 := &devicechange.Change{
		DeviceID:      "Device1",
		DeviceVersion: "Device1-1.0.0",
		Values:        testValues,
	}

	setRequest, parseError := NativeChangeToGnmiChange(change3)

	assert.NoError(t, parseError, "Parsing error for Gnmi change request")
	assert.Equal(t, len(setRequest.Update), 1)

	jsonstr, _ := json.Marshal(setRequest.Update[0])

	expectedStr := "{\"path\":{\"elem\":[{\"name\":\"cont1a\"},{\"name\":\"cont2a\"}," +
		"{\"name\":\"leaf2c\"}]},\"val\":{\"Value\":{\"StringVal\":\"def\"}}}"

	assert.Equal(t, string(jsonstr), expectedStr)

	assert.Equal(t, len(setRequest.Delete), 1)

	jsonstr2, _ := json.Marshal(setRequest.Delete[0])

	expectedStr2 := "{\"elem\":[{\"name\":\"cont1a\"},{\"name\":\"list2a\",\"key\":{\"name\":\"txout2\"}}]}"
	assert.Equal(t, string(jsonstr2), expectedStr2)
}
