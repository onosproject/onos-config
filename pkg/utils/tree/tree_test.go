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

package tree

import (
	"testing"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/config-models/modelplugin/testdevice-2.0.0/testdevice_2_0_0"
	"gotest.tools/assert"
)

const testJSON1 = `{
  "cont1a": {
    "cont2a": {
      "leaf2a": 12,
      "leaf2b": "1.141590",
      "leaf2c": "myvalue1a2c",
      "leaf2d": "1.14159",
      "leaf2f": "YXMgYnl0ZSBhcnJheQ==",
      "leaf2g": true
    },
    "leaf1a": "myvalue1a1a",
    "list2a": [
      {
        "name": "First",
        "rx-power": 6,
        "tx-power": 5
      },
      {
        "name": "Second",
        "rx-power": 8,
        "tx-power": 7
      }
    ]
  },
  "leafAtTopLevel": "WXY-1234"
}`

const (
	Test1Cont1ALeaf1a       = "/cont1a/leaf1a"
	Test1Cont1AList2a1t     = "/cont1a/list2a[name=First]/tx-power"
	Test1Cont1AList2a1r     = "/cont1a/list2a[name=First]/rx-power"
	Test1Cont1AList2a2t     = "/cont1a/list2a[name=Second]/tx-power"
	Test1Cont1AList2a2r     = "/cont1a/list2a[name=Second]/rx-power"
	Test1Cont1AList3a1t     = "/cont1a/list3a[firstName=First][secondName=Second]/tx-power"
	Test1Cont1ACont2ALeaf2A = "/cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B = "/cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C = "/cont1a/cont2a/leaf2c"
	Test1Cont1ACont2ALeaf2D = "/cont1a/cont2a/leaf2d"
	Test1Cont1ACont2ALeaf2E = "/cont1a/cont2a/leaf2e"
	Test1Cont1ACont2ALeaf2F = "/cont1a/cont2a/leaf2f"
	Test1Cont1ACont2ALeaf2G = "/cont1a/cont2a/leaf2g"
	Test1Leaftoplevel       = "/leafAtTopLevel"
)

const (
	ValueLeaf2A12       = 12
	ValueLeaf2B114      = 1.14159 // AAAA4PND8j8=
	ValueLeaf2CMyValue  = "myvalue1a2c"
	ValueLeaf1AMyValue  = "myvalue1a1a"
	ValueLeaf2D114      = 114159 // precision 5
	ValueList2b1PwrT    = 5
	ValueList2b1PwrR    = 6
	ValueList2b2PwrT    = 7
	ValueList2b2PwrR    = 8
	ValueList2A2F       = "as byte array"
	ValueList2A2G       = true
	ValueList3b2PwrR    = 9
	ValueLeaftopWxy1234 = "WXY-1234"
)

var (
	configValues []*configapi.PathValue
)

func setUpTree() {
	configValues = make([]*configapi.PathValue, 13)
	configValues[0] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2A, Value: *configapi.NewTypedValueUint(ValueLeaf2A12, 8)}
	configValues[1] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2A, Value: *configapi.NewTypedValueUint(ValueLeaf2A12, 8)}
	configValues[1] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2B, Value: *configapi.NewTypedValueFloat(ValueLeaf2B114)}

	configValues[2] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2C, Value: *configapi.NewTypedValueString(ValueLeaf2CMyValue)}
	configValues[3] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2C, Value: *configapi.NewTypedValueString(ValueLeaf2CMyValue)}
	configValues[4] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2C, Value: *configapi.NewTypedValueString(ValueLeaf2CMyValue)}
	configValues[3] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2D, Value: *configapi.NewTypedValueDecimal(ValueLeaf2D114, 5)}
	//configValues[4] = &devicechange.PathValue{Path: Test1Cont1ACont2ALeaf2E, Value: devicechange.NewTypedValueInt(ValueLeaf2A12)}

	configValues[5] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2F, Value: *configapi.NewTypedValueBytes([]byte(ValueList2A2F))}
	configValues[6] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2G, Value: *configapi.NewTypedValueBool(ValueList2A2G)}
	configValues[7] = &configapi.PathValue{Path: Test1Cont1ALeaf1a, Value: *configapi.NewTypedValueString(ValueLeaf1AMyValue)}

	configValues[8] = &configapi.PathValue{Path: Test1Cont1AList2a1t, Value: *configapi.NewTypedValueUint(ValueList2b1PwrT, 16)}
	configValues[9] = &configapi.PathValue{Path: Test1Cont1AList2a1r, Value: *configapi.NewTypedValueUint(ValueList2b1PwrR, 16)}
	configValues[10] = &configapi.PathValue{Path: Test1Cont1AList2a2t, Value: *configapi.NewTypedValueUint(ValueList2b2PwrT, 16)}
	configValues[11] = &configapi.PathValue{Path: Test1Cont1AList2a2r, Value: *configapi.NewTypedValueUint(ValueList2b2PwrR, 16)}
	// TODO add back in a double key list - but must be added to the YANG model first
	configValues[12] = &configapi.PathValue{Path: Test1Leaftoplevel, Value: *configapi.NewTypedValueString(ValueLeaftopWxy1234)}
}

func Test_BuildTree(t *testing.T) {
	setUpTree()

	jsonTree, err := BuildTree(configValues, true)

	assert.NilError(t, err)
	assert.Equal(t, testJSON1, string(jsonTree))

	//Verify we can go back to objects
	model := new(testdevice_2_0_0.Device)
	err = testdevice_2_0_0.Unmarshal(jsonTree, model)
	assert.NilError(t, err, "unexpected error unmarshalling JSON to modelplugin model")
	assert.Equal(t, ValueLeaf2A12, int(*model.Cont1A.Cont2A.Leaf2A))
	// TODO handle Leaf2B (decimal64)
	assert.Equal(t, ValueLeaf2CMyValue, *model.Cont1A.Cont2A.Leaf2C)
	// TODO handle Leaf2D (decimal64)
	// TODO handle Leaf2E (leaf list)
	assert.Equal(t, ValueList2A2F, string(model.Cont1A.Cont2A.Leaf2F))
	assert.Equal(t, ValueList2A2G, *model.Cont1A.Cont2A.Leaf2G)

	assert.Equal(t, ValueLeaf1AMyValue, *model.Cont1A.Leaf1A)

	assert.Equal(t, 2, len(model.Cont1A.List2A))
	list2AFirst, ok := model.Cont1A.List2A["First"]
	assert.Assert(t, ok)
	assert.Equal(t, "First", *list2AFirst.Name)
	//assert.Equal(t, 5, int(*list2AFirst.TxPower)) <<-- https://github.com/onosproject/onos-config/issues/1186
	assert.Equal(t, 6, int(*list2AFirst.RxPower))

	list2ASecond, ok := model.Cont1A.List2A["Second"]
	assert.Assert(t, ok)
	assert.Equal(t, "Second", *list2ASecond.Name)
	//assert.Equal(t, 7, int(*list2ASecond.TxPower)) <<-- https://github.com/onosproject/onos-config/issues/1186
	assert.Equal(t, 8, int(*list2ASecond.RxPower))

	assert.Equal(t, ValueLeaftopWxy1234, *model.LeafAtTopLevel)

}
