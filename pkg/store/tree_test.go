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

const testJSON1 = `{"test1:cont1a":{"cont2a":{"leaf2a":12,"leaf2b":"1.14159","leaf2c":"myvalue1a2c"},"leaf1a":"myvalue1a1a","list2a":[{"name":"First","rx-power":6,"tx-power":5},{"name":"Second","rx-power":8,"tx-power":7}]},"test1:leafAtTopLevel":"WXY-1234"}`

const (
	Test1Cont1ALeaf1a   = "/test1:cont1a/leaf1a"
	Test1Cont1AList2a1t = "/test1:cont1a/list2a[name=First]/tx-power"
	Test1Cont1AList2a1r = "/test1:cont1a/list2a[name=First]/rx-power"
	Test1Cont1AList2a2t = "/test1:cont1a/list2a[name=Second]/tx-power"
	Test1Cont1AList2a2r = "/test1:cont1a/list2a[name=Second]/rx-power"
)

const (
	ValueLeaf2A12      = "12"
	ValueLeaf2B114     = "1.14159"
	ValueLeaf2CMyValue = "myvalue1a2c"
	ValueLeaf1AMyValue = "myvalue1a1a"
	ValueList2b1PwrT   = "5"
	ValueList2b1PwrR   = "6"
	ValueList2b2PwrT   = "7"
	ValueList2b2PwrR   = "8"
)

var (
	configValues []*change.ConfigValue
)

func setUpTree() {
	configValues = make([]*change.ConfigValue, 9)

	configValues[0] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2A, Value: ValueLeaf2A12}
	configValues[1] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2B, Value: ValueLeaf2B114}
	configValues[2] = &change.ConfigValue{Path: Test1Cont1ACont2ALeaf2C, Value: ValueLeaf2CMyValue}
	configValues[3] = &change.ConfigValue{Path: Test1Cont1ALeaf1a, Value: ValueLeaf1AMyValue}
	configValues[4] = &change.ConfigValue{Path: Test1Cont1AList2a1t, Value: ValueList2b1PwrT}
	configValues[5] = &change.ConfigValue{Path: Test1Cont1AList2a1r, Value: ValueList2b1PwrR}
	configValues[6] = &change.ConfigValue{Path: Test1Cont1AList2a2t, Value: ValueList2b2PwrT}
	configValues[7] = &change.ConfigValue{Path: Test1Cont1AList2a2r, Value: ValueList2b2PwrR}
	configValues[8] = &change.ConfigValue{Path: Test1Leaftoplevel, Value: ValueLeaftopWxy1234}

}

func Test_BuildTree(t *testing.T) {
	setUpTree()

	jsonTree, err := BuildTree(configValues)

	assert.NilError(t, err)
	assert.Equal(t, string(jsonTree), testJSON1)
}
