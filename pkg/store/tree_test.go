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
	"github.com/onosproject/onos-config/pkg/store/change"
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
	configValues []*change.ConfigValue
)

func setUpTree() {
	configValues = make([]*change.ConfigValue, 13)
	configValues[0] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2A, TypedValue: *change.CreateTypedValueUint64(ValueLeaf2A12)}

	configValues[1] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2B, TypedValue: *change.CreateTypedValueFloat(ValueLeaf2B114)}
	configValues[2] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2C, TypedValue: *change.CreateTypedValueString(ValueLeaf2CMyValue)}
	configValues[3] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2D, TypedValue: *change.CreateTypedValueDecimal64(ValueLeaf2D114, 5)}
	configValues[4] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2E, TypedValue: *change.CreateTypedValueInt64(ValueLeaf2A12)}
	configValues[5] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2F, TypedValue: *change.CreateTypedValueBytes([]byte(ValueList2A2F))}
	configValues[6] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2G, TypedValue: *change.CreateTypedValueBool(ValueList2A2G)}
	configValues[7] = &change.ConfigValue{Path: Test1Cont1ALeaf1a, TypedValue: *change.CreateTypedValueString(ValueLeaf1AMyValue)}

	configValues[8] = &change.ConfigValue{Path: Test1Cont1AList2a1t, TypedValue: *change.CreateTypedValueUint64(ValueList2b1PwrT)}
	configValues[9] = &change.ConfigValue{Path: Test1Cont1AList2a1r, TypedValue: *change.CreateTypedValueUint64(ValueList2b1PwrR)}
	configValues[10] = &change.ConfigValue{Path: Test1Cont1AList2a2t, TypedValue: *change.CreateTypedValueUint64(ValueList2b2PwrT)}
	configValues[11] = &change.ConfigValue{Path: Test1Cont1AList2a2r, TypedValue: *change.CreateTypedValueUint64(ValueList2b2PwrR)}

	configValues[12] = &change.ConfigValue{Path: Test1Leaftoplevel, TypedValue: *change.CreateTypedValueString(ValueLeaftopWxy1234)}

}

func Test_BuildTree(t *testing.T) {
	setUpTree()

	jsonTree, err := BuildTree(configValues, false)

	assert.NilError(t, err)
	assert.Equal(t, string(jsonTree), testJSON1)
}
