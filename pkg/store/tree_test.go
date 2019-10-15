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

package store

import (
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	"gotest.tools/assert"
	"testing"
)

const testJSON1 = `{"cont1a":{"cont2a":{"leaf2a":12,"leaf2b":1.14159,"leaf2c":"myvalue1a2c","leaf2d":1.14159,"leaf2e":12,"leaf2f":"YXMgYnl0ZSBhcnJheQ==","leaf2g":true},"leaf1a":"myvalue1a1a","list2a":[{"name":"First","rx-power":6,"tx-power":5},{"name":"Second","rx-power":8,"tx-power":7}]},"leafAtTopLevel":"WXY-1234"}`

const (
	Test1Cont1ALeaf1a   = "/cont1a/leaf1a"
	Test1Cont1AList2a1t = "/cont1a/list2a[name=First]/tx-power"
	Test1Cont1AList2a1r = "/cont1a/list2a[name=First]/rx-power"
	Test1Cont1AList2a2t = "/cont1a/list2a[name=Second]/tx-power"
	Test1Cont1AList2a2r = "/cont1a/list2a[name=Second]/rx-power"
)

const (
	ValueLeaf2A12      = 12
	ValueLeaf2B114     = 1.14159 // AAAA4PND8j8=
	ValueLeaf2CMyValue = "myvalue1a2c"
	ValueLeaf1AMyValue = "myvalue1a1a"
	ValueLeaf2D114     = 114159 // precision 5
	ValueList2b1PwrT   = 5
	ValueList2b1PwrR   = 6
	ValueList2b2PwrT   = 7
	ValueList2b2PwrR   = 8
	ValueList2A2F      = "as byte array"
	ValueList2A2G      = true
)

var (
	configValues []*types.PathValue
)

func setUpTree() {
	configValues = make([]*types.PathValue, 13)
	configValues[0] = &types.PathValue{Path: Test1Cont1ACont2ALeaf2A, Value: types.NewTypedValueUint64(ValueLeaf2A12)}

	configValues[1] = &types.PathValue{Path: Test1Cont1ACont2ALeaf2B, Value: types.NewTypedValueFloat(ValueLeaf2B114)}
	configValues[2] = &types.PathValue{Path: Test1Cont1ACont2ALeaf2C, Value: types.NewTypedValueString(ValueLeaf2CMyValue)}
	configValues[3] = &types.PathValue{Path: Test1Cont1ACont2ALeaf2D, Value: types.NewTypedValueDecimal64(ValueLeaf2D114, 5)}
	configValues[4] = &types.PathValue{Path: Test1Cont1ACont2ALeaf2E, Value: types.NewTypedValueInt64(ValueLeaf2A12)}
	configValues[5] = &types.PathValue{Path: Test1Cont1ACont2ALeaf2F, Value: types.NewTypedValueBytes([]byte(ValueList2A2F))}
	configValues[6] = &types.PathValue{Path: Test1Cont1ACont2ALeaf2G, Value: types.NewTypedValueBool(ValueList2A2G)}
	configValues[7] = &types.PathValue{Path: Test1Cont1ALeaf1a, Value: types.NewTypedValueString(ValueLeaf1AMyValue)}

	configValues[8] = &types.PathValue{Path: Test1Cont1AList2a1t, Value: types.NewTypedValueUint64(ValueList2b1PwrT)}
	configValues[9] = &types.PathValue{Path: Test1Cont1AList2a1r, Value: types.NewTypedValueUint64(ValueList2b1PwrR)}
	configValues[10] = &types.PathValue{Path: Test1Cont1AList2a2t, Value: types.NewTypedValueUint64(ValueList2b2PwrT)}
	configValues[11] = &types.PathValue{Path: Test1Cont1AList2a2r, Value: types.NewTypedValueUint64(ValueList2b2PwrR)}

	configValues[12] = &types.PathValue{Path: Test1Leaftoplevel, Value: types.NewTypedValueString(ValueLeaftopWxy1234)}

}

func Test_BuildTree(t *testing.T) {
	setUpTree()

	jsonTree, err := BuildTree(configValues, false)

	assert.NilError(t, err)
	assert.Equal(t, string(jsonTree), testJSON1)
}
