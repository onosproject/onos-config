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

package change

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"gotest.tools/assert"
	"io"
	log "k8s.io/klog"
	"os"
	"strings"
	"testing"
)

var (
	change1 *Change
)

const (
	Test1Cont1A                  = "/cont1a"
	Test1Cont1ACont2A            = "/cont1a/cont2a"
	Test1Cont1ACont2ALeaf2A      = "/cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B      = "/cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C      = "/cont1a/cont2a/leaf2c"
	Test1Cont1ACont2ALeaf2D      = "/cont1a/cont2a/leaf2d"
	Test1Cont1ACont2ALeaf2E      = "/cont1a/cont2a/leaf2e"
	Test1Cont1ACont2ALeaf2F      = "/cont1a/cont2a/leaf2f"
	Test1Cont1ALeaf1A            = "/cont1a/leaf1a"
	Test1Cont1AList2ATxout1      = "/cont1a/list2a[name=txout1]"
	Test1Cont1AList2ATxout1Txpwr = "/cont1a/list2a[name=txout1]/tx-power"
	Test1Cont1AList2ATxout2      = "/cont1a/list2a[name=txout2]"
	Test1Cont1AList2ATxout2Txpwr = "/cont1a/list2a[name=txout2]/tx-power"
	Test1Leaftoplevel            = "/leafAtTopLevel"
)

const (
	ValueLeaf2A13       = 13
	ValueLeaf2B159D     = 1579
	ValueLeaf2B159P     = 3
	ValueLeaf2CAbc      = "abc"
	ValueLeaf2E1        = -32
	ValueLeaf2E2        = 99
	ValueLeaf2E3        = 123
	ValueLeaf2F         = "b25vcyByb2Nrcwo="
	ValueLeaf1AAbcdef   = "abcdef"
	ValueTxout1Txpwr8   = 8
	ValueTxout2Txpwr10  = 10
	ValueLeaftopWxy1234 = "WXY-1234"
)

func TestMain(m *testing.M) {
	var err error
	log.SetOutput(os.Stdout)
	config1Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1A, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2A, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value03, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2A, devicechangetypes.NewTypedValueUint64(ValueLeaf2A13), false)
	config1Value04, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2B, devicechangetypes.NewTypedValueDecimal64(ValueLeaf2B159D, ValueLeaf2B159P), false)
	config1Value05, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2C, devicechangetypes.NewTypedValueString(ValueLeaf2CAbc), false)
	config1Value06, _ := devicechangetypes.NewChangeValue(Test1Cont1ALeaf1A, devicechangetypes.NewTypedValueString(ValueLeaf1AAbcdef), false)
	config1Value07, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout1, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value08, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout1Txpwr, devicechangetypes.NewTypedValueUint64(ValueTxout1Txpwr8), false)
	config1Value09, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout2, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value10, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout2Txpwr, devicechangetypes.NewTypedValueUint64(ValueTxout2Txpwr10), false)
	config1Value11, _ := devicechangetypes.NewChangeValue(Test1Leaftoplevel, devicechangetypes.NewTypedValueString(ValueLeaftopWxy1234), false)
	config1Value12, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2D, devicechangetypes.NewTypedValueBool(true), false)
	config1Value13, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2E, devicechangetypes.NewLeafListInt64Tv([]int{ValueLeaf2E1, ValueLeaf2E2, ValueLeaf2E3}), false)
	valueLeaf2FBa, _ := base64.StdEncoding.DecodeString(ValueLeaf2F)
	config1Value14, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2F, devicechangetypes.NewTypedValueBytes(valueLeaf2FBa), false)

	change1, err = NewChange([]*devicechangetypes.ChangeValue{
		config1Value01, config1Value02, config1Value03, config1Value04, config1Value05,
		config1Value06, config1Value07, config1Value08, config1Value09, config1Value10,
		config1Value11, config1Value12, config1Value13, config1Value14,
	}, "Original Config for test switch")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	os.Exit(m.Run())
}

func Test_ConfigValue(t *testing.T) {
	// Create ConfigValue from strings
	path := "/cont1a/cont2a/leaf2a"
	value := 13

	configValue2a, _ := devicechangetypes.NewChangeValue(path, devicechangetypes.NewTypedValueUint64(uint(value)), false)

	assert.Equal(t, configValue2a.Path, path)
	assert.Equal(t, configValue2a.GetValue().ValueToString(), "13")
	assert.Equal(t, configValue2a.GetValue().GetType(), devicechangetypes.ValueType_UINT)
	assert.Equal(t, configValue2a.GetPath(), "/cont1a/cont2a/leaf2a")
	assert.Equal(t, configValue2a.GetRemoved(), false)
}

func Test_changecreation(t *testing.T) {

	log.Infof("Change %x created\n", change1.ID)
	h := sha1.New()
	jsonstr, _ := json.Marshal(change1.Config)
	_, err1 := io.WriteString(h, string(jsonstr))
	assert.NilError(t, err1, "Error when writing objects to json")
	hash := h.Sum(nil)

	expectedID := b64(hash)
	actualID := b64(change1.ID)
	assert.Equal(t, actualID, expectedID, "Creation of Change failed")

	err := change1.IsValid()
	assert.NilError(t, err, "Checking of Change failed")

	assert.Equal(t, len(change1.Config), 14)
	assert.Equal(t, change1.Config[13].Path, Test1Leaftoplevel) // Are added in alpha order

	changeEmpty := Change{}
	errEmpty := changeEmpty.IsValid()
	assert.Error(t, errEmpty, "Empty Change")
	assert.ErrorContains(t, errEmpty, "Empty Change")

	oddID := [10]byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	changeOdd := Change{ID: oddID[:]}
	errOdd := changeOdd.IsValid()
	assert.ErrorContains(t, errOdd, "does not match", "Checking of Change failed")
}

func Test_badpath(t *testing.T) {
	badpath := "does_not_have_any_slash"
	conf1, err1 := devicechangetypes.NewChangeValue(badpath, devicechangetypes.NewTypedValueString("123"), false)

	assert.Error(t, err1, badpath, "Expected error on ", badpath)

	assert.Assert(t, conf1 == nil, "Expected config to be empty on error")

	badpath = "//two/contiguous/slashes"
	_, err2 := devicechangetypes.NewChangeValue(badpath, devicechangetypes.NewTypedValueString("123"), false)
	assert.ErrorContains(t, err2, badpath, "Expected error on path", badpath)

	badpath = "/test*"
	_, err3 := devicechangetypes.NewChangeValue(badpath, devicechangetypes.NewTypedValueString("123"), false)
	assert.ErrorContains(t, err3, badpath, "Expected error on path", badpath)
}

func Test_changeValueString(t *testing.T) {
	cv1, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2A, devicechangetypes.NewTypedValueUint64(123), false)

	assert.Equal(t, cv1.GetValue().ValueToString(), "123",
		"Expected changeValue to produce string")
}

func Test_changeString(t *testing.T) {
	cv1, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2A, devicechangetypes.NewTypedValueUint64(123), false)
	cv2, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2B, devicechangetypes.NewTypedValueString("ABC"), false)
	cv3, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2C, devicechangetypes.NewTypedValueString("Hello"), false)
	cv4, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2D, devicechangetypes.NewTypedValueBool(true), false)

	changeObj, _ := NewChange([]*devicechangetypes.ChangeValue{cv1, cv2, cv3, cv4}, "Test Change")

	var expected = `"Config":[` +
		`{"Path":"/cont1a/cont2a/leaf2a","Value":{"Bytes":"ewAAAAAAAAA=","Type":3}},` +
		`{"Path":"/cont1a/cont2a/leaf2b","Value":{"Bytes":"QUJD","Type":1}},` +
		`{"Path":"/cont1a/cont2a/leaf2c","Value":{"Bytes":"SGVsbG8=","Type":1}},` +
		`{"Path":"/cont1a/cont2a/leaf2d","Value":{"Bytes":"AQ==","Type":4}}]}`

	assert.Assert(t, strings.Contains(changeObj.String(), expected),
		"Expected change to produce string. Got", changeObj.String())
}
