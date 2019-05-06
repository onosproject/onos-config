// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

var (
	change1, change2, change3, change4 *Change
)

const (
	Test1Cont1A                  = "/test1:cont1a"
	Test1Cont1ACont2A            = "/test1:cont1a/cont2a"
	Test1Cont1ACont2ALeaf2A      = "/test1:cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B      = "/test1:cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C      = "/test1:cont1a/cont2a/leaf2c"
	Test1Cont1ALeaf1A            = "/test1:cont1a/leaf1a"
	Test1Cont1AList2ATxout1      = "/test1:cont1a/list2a=txout1"
	Test1Cont1AList2ATxout1Txpwr = "/test1:cont1a/list2a=txout1/tx-power"
	Test1Cont1AList2ATxout2      = "/test1:cont1a/list2a=txout2"
	Test1Cont1AList2ATxout2Txpwr = "/test1:cont1a/list2a=txout2/tx-power"
	Test1Cont1AList2ATxout3      = "/test1:cont1a/list2a=txout3"
	Test1Cont1AList2ATxout3Txpwr = "/test1:cont1a/list2a=txout3/tx-power"
	Test1Leaftoplevel            = "/test1:leafAtTopLevel"
)

const (
	ValueEmpty          = ""
	ValueLeaf2A13       = "13"
	ValueLeaf2B159      = "1.579"
	ValueLeaf2B314      = "3.14159"
	ValueLeaf2CAbc      = "abc"
	ValueLeaf2CDef      = "def"
	ValueLeaf2CGhi      = "ghi"
	ValueLeaf1AAbcdef   = "abcdef"
	ValueTxout1Txpwr8   = "8"
	ValueTxout2Txpwr10  = "10"
	ValueTxout3Txpwr16  = "16"
	ValueLeaftopWxy1234 = "WXY-1234"
)

func TestMain(m *testing.M) {
	var err error
	config1Value01, _ := CreateChangeValue(Test1Cont1A, ValueEmpty, false)
	config1Value02, _ := CreateChangeValue(Test1Cont1ACont2A, ValueEmpty, false)
	config1Value03, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2A, ValueLeaf2A13, false)
	config1Value04, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2B, ValueLeaf2B159, false)
	config1Value05, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2C, ValueLeaf2CAbc, false)
	config1Value06, _ := CreateChangeValue(Test1Cont1ALeaf1A, ValueLeaf1AAbcdef, false)
	config1Value07, _ := CreateChangeValue(Test1Cont1AList2ATxout1, ValueEmpty, false)
	config1Value08, _ := CreateChangeValue(Test1Cont1AList2ATxout1Txpwr, ValueTxout1Txpwr8, false)
	config1Value09, _ := CreateChangeValue(Test1Cont1AList2ATxout2, ValueEmpty, false)
	config1Value10, _ := CreateChangeValue(Test1Cont1AList2ATxout2Txpwr, ValueTxout2Txpwr10, false)
	config1Value11, _ := CreateChangeValue(Test1Leaftoplevel, ValueLeaftopWxy1234, false)
	change1, err = CreateChange(ValueCollections{
		config1Value01, config1Value02, config1Value03, config1Value04, config1Value05,
		config1Value06, config1Value07, config1Value08, config1Value09, config1Value10,
		config1Value11,
	}, "Original Config for test switch")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	os.Exit(m.Run())
}

func Test_ConfigValue(t *testing.T) {
	// Create ConfigValue from strings
	path := "/test1:cont1a/cont2a/leaf2a"
	value := "13"

	configValue2a, _ := CreateChangeValue(path, value, false)

	if configValue2a.Path != path {
		t.Errorf("Retrieval of ConfigValue.Path failed. Expected %q Got %q",
			path, configValue2a.Path)
	}

	if string(configValue2a.Value) != value {
		t.Errorf("Retrieval of ConfigValue.Path failed. Expected %q Got %q",
			value, configValue2a.Value)
	}
}

func Test_changecreation(t *testing.T) {

	fmt.Printf("Change %x created\n", change1.ID)
	h := sha1.New()
	jsonstr, _ := json.Marshal(change1.Config)
	_, err1 := io.WriteString(h, string(jsonstr))
	_, err2 := io.WriteString(h, change1.Description)
	_, err3 := io.WriteString(h, change1.Created.String())
	if err1 != nil || err2 != nil || err3 != nil {
		t.Errorf("Error when writing objects to json")
	}
	hash := h.Sum(nil)

	expectedID := b64(hash)
	actualID := b64(change1.ID)
	if actualID != expectedID {
		t.Errorf("Creation of Change failed Expected %s got %s",
			expectedID, actualID)
	}
	err := change1.IsValid()
	if err != nil {
		t.Errorf("Checking of Change failed %s", err)
	}

	changeEmpty := Change{}
	errEmpty := changeEmpty.IsValid()
	if errEmpty == nil {
		t.Errorf("Checking of Change failed %s", errEmpty)
	}
	if errEmpty.Error() != "Empty Change" {
		t.Errorf("Expecting error 'Empty Change' Got: %s", errEmpty)
	}

	oddID := [10]byte{10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
	changeOdd := Change{ID: oddID[:]}
	errOdd := changeOdd.IsValid()
	if errOdd == nil {
		t.Errorf("Checking of Change failed %s", errOdd)
	}
	if !strings.Contains(errOdd.Error(), "does not match") {
		t.Errorf("Expecting error 'does not match' Got: %s", errOdd)
	}
}

func Test_badpath(t *testing.T) {
	badpath := "does_not_have_any_slash"
	conf1, err1 := CreateChangeValue(badpath, "123", false)

	if err1 == nil {
		t.Errorf("Expected '%s' to produce error", badpath)
	} else if err1.Error() != badpath {
		t.Errorf("Expected error to be '%s' Got: '%s'", badpath, err1)
	}
	if conf1 != nil {
		t.Errorf("Expected config to be empty on error")
	}

	badpath = "//two/contiguous/slashes"
	_, err2 := CreateChangeValue(badpath, "123", false)

	if err2 == nil {
		t.Errorf("Expected '%s' to produce error", badpath)
	}

	badpath = "/test*"
	_, err3 := CreateChangeValue(badpath, "123", false)

	if err3 == nil {
		t.Errorf("Expected '%s' to produce error", badpath)
	}

}

func Test_changeValueString(t *testing.T) {
	cv1, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2A, "123", false)

	var expected = "/test1:cont1a/cont2a/leaf2a 123 false"
	if cv1.String() != expected {
		t.Errorf("Expected changeValue to produce string %s. Got: %s",
			expected, cv1.String())
	}

	//Test the error
	cv2 := Value{}
	if cv2.String() != "InvalidChange" {
		t.Errorf("Expected empty changeValue to produce InvalidChange Got: %s",
			cv2.String())
	}
}

func Test_changeString(t *testing.T) {
	cv1, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2A, "123", false)
	cv2, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2B, "ABC", false)
	cv3, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2C, "Hello", false)

	changeObj, _ := CreateChange(ValueCollections{cv1, cv2, cv3}, "Test Change")

	var expected = `"Config":[` +
		`{"Path":"/test1:cont1a/cont2a/leaf2a","Value":"123","Remove":false},` +
		`{"Path":"/test1:cont1a/cont2a/leaf2b","Value":"ABC","Remove":false},` +
		`{"Path":"/test1:cont1a/cont2a/leaf2c","Value":"Hello","Remove":false}]}`

	if !strings.Contains(changeObj.String(), expected) {
		t.Errorf("Expected change to produce string %s. Got: %s",
			expected, changeObj.String())
	}

	change2 := Change{}
	if !strings.Contains(change2.String(), "") {
		t.Errorf("Expected change2 to produce empty string. Got: %s",
			change2.String())
	}

}

func Test_duplicate_path(t *testing.T) {
	cv1, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2B, ValueLeaf2B314, false)
	cv2, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2C, ValueLeaf2CAbc, false)

	cv3, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2B, ValueLeaf2B159, false)

	change, err := CreateChange(ValueCollections{cv1, cv2, cv3}, "Test Change")

	if err == nil {
		t.Errorf("Expected %s to produce error for duplicate path", Test1Cont1ACont2ALeaf2B)
	}

	if change != nil {
		t.Errorf("Expected change to be nil because of duplicate path")
	}
}
