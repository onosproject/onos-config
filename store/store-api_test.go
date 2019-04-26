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

package store

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	change1, change2, change3, change4 *Change
)

var (
	changeStore        map[string]*Change
	configurationStore map[string]Configuration
)

var (
	device1V, device2V Configuration
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

var Config1Paths = [11]string{
	Test1Cont1A, // 0
	Test1Cont1ACont2A,
	Test1Cont1ACont2ALeaf2A,
	Test1Cont1ACont2ALeaf2B, // 3
	Test1Cont1ACont2ALeaf2C,
	Test1Cont1ALeaf1A, // 5
	Test1Cont1AList2ATxout1,
	Test1Cont1AList2ATxout1Txpwr,
	Test1Cont1AList2ATxout3, //10
	Test1Cont1AList2ATxout3Txpwr,
	Test1Leaftoplevel,
}

var Config1Values = [11]string{
	ValueEmpty, // 0
	ValueEmpty,
	ValueLeaf2A13,
	ValueLeaf2B314, // 3
	ValueLeaf2CDef,
	ValueLeaf1AAbcdef, // 5
	ValueEmpty,
	ValueTxout1Txpwr8,
	ValueEmpty, // 10
	ValueTxout3Txpwr16,
	ValueLeaftopWxy1234,
}

var Config1PreviousPaths = [13]string{
	Test1Cont1A, // 0
	Test1Cont1ACont2A,
	Test1Cont1ACont2ALeaf2A,
	Test1Cont1ACont2ALeaf2B, // 3
	Test1Cont1ACont2ALeaf2C,
	Test1Cont1ALeaf1A, // 5
	Test1Cont1AList2ATxout1,
	Test1Cont1AList2ATxout1Txpwr,
	Test1Cont1AList2ATxout2,
	Test1Cont1AList2ATxout2Txpwr,
	Test1Cont1AList2ATxout3, //10
	Test1Cont1AList2ATxout3Txpwr,
	Test1Leaftoplevel,
}

var Config1PreviousValues = [13]string{
	ValueEmpty, // 0
	ValueEmpty,
	ValueLeaf2A13,
	ValueLeaf2B314, // 3
	ValueLeaf2CAbc,
	ValueLeaf1AAbcdef, // 5
	ValueEmpty,
	ValueTxout1Txpwr8,
	ValueEmpty,
	ValueTxout2Txpwr10,
	ValueEmpty, // 10
	ValueTxout3Txpwr16,
	ValueLeaftopWxy1234,
}

var Config1FirstPaths = [11]string{
	Test1Cont1A, // 0
	Test1Cont1ACont2A,
	Test1Cont1ACont2ALeaf2A,
	Test1Cont1ACont2ALeaf2B, // 3
	Test1Cont1ACont2ALeaf2C,
	Test1Cont1ALeaf1A, // 5
	Test1Cont1AList2ATxout1,
	Test1Cont1AList2ATxout1Txpwr,
	Test1Cont1AList2ATxout2,
	Test1Cont1AList2ATxout2Txpwr,
	Test1Leaftoplevel, //10
}

var Config1FirstValues = [11]string{
	ValueEmpty, // 0
	ValueEmpty,
	ValueLeaf2A13,
	ValueLeaf2B159, // 3
	ValueLeaf2CAbc,
	ValueLeaf1AAbcdef, // 5
	ValueEmpty,
	ValueTxout1Txpwr8,
	ValueEmpty,
	ValueTxout2Txpwr10,
	ValueLeaftopWxy1234, //10
}

var Config2Paths = [11]string{
	Test1Cont1A, // 0
	Test1Cont1ACont2A,
	Test1Cont1ACont2ALeaf2A,
	Test1Cont1ACont2ALeaf2B, // 3
	Test1Cont1ACont2ALeaf2C,
	Test1Cont1ALeaf1A, // 5
	Test1Cont1AList2ATxout2,
	Test1Cont1AList2ATxout2Txpwr,
	Test1Cont1AList2ATxout3, //10
	Test1Cont1AList2ATxout3Txpwr,
	Test1Leaftoplevel,
}

var Config2Values = [11]string{
	ValueEmpty, // 0
	ValueEmpty,
	ValueLeaf2A13,
	ValueLeaf2B314, // 3
	ValueLeaf2CGhi,
	ValueLeaf1AAbcdef, // 5
	ValueEmpty,
	ValueTxout2Txpwr10,
	ValueEmpty, // 10
	ValueTxout3Txpwr16,
	ValueLeaftopWxy1234,
}

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
	change1, err = CreateChange(ChangeValueCollection{
		config1Value01, config1Value02, config1Value03, config1Value04, config1Value05,
		config1Value06, config1Value07, config1Value08, config1Value09, config1Value10,
		config1Value11,
	}, "Original Config for test switch")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	config2Value01, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2B, ValueLeaf2B314, false)
	config2Value02, _ := CreateChangeValue(Test1Cont1AList2ATxout3, ValueEmpty, false)
	config2Value03, _ := CreateChangeValue(Test1Cont1AList2ATxout3Txpwr, ValueTxout3Txpwr16, false)
	change2, err = CreateChange(ChangeValueCollection{
		config2Value01, config2Value02, config2Value03,
	}, "Trim power level")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	config3Value01, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2C, ValueLeaf2CDef, false)
	config3Value02, _ := CreateChangeValue(Test1Cont1AList2ATxout2, ValueEmpty, true)
	change3, err = CreateChange(ChangeValueCollection{
		config3Value01, config3Value02,
	}, "Remove txout 2")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	changeStore = make(map[string]*Change)
	changeStore[B64(change1.ID)] = change1
	changeStore[B64(change2.ID)] = change2
	changeStore[B64(change3.ID)] = change3

	device1V = Configuration{
		Name:        "Device1Version",
		Device:      "Device1",
		Created:     time.Now(),
		Updated:     time.Now(),
		User:        "onos",
		Description: "Configuration for Device 1",
		Changes:     []ChangeID{change1.ID, change2.ID, change3.ID},
	}
	configurationStore = make(map[string]Configuration)
	configurationStore["Device1Version"] = device1V

	config4Value01, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2C, ValueLeaf2CGhi, false)
	config4Value02, _ := CreateChangeValue(Test1Cont1AList2ATxout1, ValueEmpty, true)
	change4, err = CreateChange(ChangeValueCollection{
		config4Value01, config4Value02,
	}, "Remove txout 1")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	changeStore[B64(change4.ID)] = change4

	device2V = Configuration{
		Name:        "Device2VersionMain",
		Device:      "Device2",
		Created:     time.Now(),
		Updated:     time.Now(),
		User:        "onos",
		Description: "Main Configuration for Device 2",
		Changes:     []ChangeID{change1.ID, change2.ID, change4.ID},
	}

	configurationStore["Device2VersionMain"] = device2V

	os.Exit(m.Run())
}

func Test_1(t *testing.T) {
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

	expectedID := B64(hash)
	actualID := B64(change1.ID)
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
	cv2 := ChangeValue{}
	if cv2.String() != "InvalidChange" {
		t.Errorf("Expected empty changeValue to produce InvalidChange Got: %s",
			cv2.String())
	}
}

func Test_changeString(t *testing.T) {
	cv1, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2A, "123", false)
	cv2, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2B, "ABC", false)
	cv3, _ := CreateChangeValue(Test1Cont1ACont2ALeaf2C, "Hello", false)

	change, _ := CreateChange(ChangeValueCollection{cv1, cv2, cv3}, "Test Change")

	var expected = `"Config":[` +
		`{"Path":"/test1:cont1a/cont2a/leaf2a","Value":"123","Remove":false},` +
		`{"Path":"/test1:cont1a/cont2a/leaf2b","Value":"ABC","Remove":false},` +
		`{"Path":"/test1:cont1a/cont2a/leaf2c","Value":"Hello","Remove":false}]}`

	if !strings.Contains(change.String(), expected) {
		t.Errorf("Expected change to produce string %s. Got: %s",
			expected, change.String())
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

	change, err := CreateChange(ChangeValueCollection{cv1, cv2, cv3}, "Test Change")

	if err == nil {
		t.Errorf("Expected %s to produce error for duplicate path", Test1Cont1ACont2ALeaf2B)
	}

	if change != nil {
		t.Errorf("Expected change to be nil because of duplicate path")
	}
}

func Test_device1_version(t *testing.T) {
	fmt.Println("Configuration", device1V.Name, " (latest) Changes:")
	for idx, cid := range device1V.Changes {
		fmt.Printf("%d: %s\n", idx, B64([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore, 0)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
	}

	for i := 0; i < len(Config1Paths); i++ {
		checkPathvalue(t, config, i,
			Config1Paths[0:11], Config1Values[0:11])
	}
}

func Test_device1_prev_version(t *testing.T) {
	const changePrevious = 1
	fmt.Println("Configuration", device1V.Name, " (n-1) Changes:")
	for idx, cid := range device1V.Changes[0 : len(device1V.Changes)-changePrevious] {
		fmt.Printf("%d: %s\n", idx, B64([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore, changePrevious)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
	}

	for i := 0; i < len(Config1PreviousPaths); i++ {
		checkPathvalue(t, config, i,
			Config1PreviousPaths[0:13], Config1PreviousValues[0:13])
	}
}

func Test_device1_first_version(t *testing.T) {
	const changePrevious = 2
	fmt.Println("Configuration", device1V.Name, " (n-2) Changes:")
	for idx, cid := range device1V.Changes[0 : len(device1V.Changes)-changePrevious] {
		fmt.Printf("%d: %s\n", idx, B64([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore, changePrevious)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
	}

	for i := 0; i < len(Config1FirstPaths); i++ {
		checkPathvalue(t, config, i,
			Config1FirstPaths[0:11], Config1FirstValues[0:11])
	}
}

func Test_device1_invalid_version(t *testing.T) {
	const changePrevious = 3
	fmt.Println("Configuration", device1V.Name, " (n-3) Changes:")
	for idx, cid := range device1V.Changes[0 : len(device1V.Changes)-changePrevious] {
		fmt.Printf("%d: %s\n", idx, B64([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore, changePrevious)
	if len(config) > 0 {
		t.Errorf("Not expecting any values for change (n-3). Got %d", len(config))
	}
}

func Test_device2_version(t *testing.T) {
	fmt.Println("Configuration", device2V.Name, " (latest) Changes:")
	for idx, cid := range device2V.Changes {
		fmt.Printf("%d: %s\n", idx, B64([]byte(cid)))
	}

	if device2V.Name != "Device2VersionMain" {
		t.Errorf("Unexpected name for Configuration main %s", device2V.Name)
	}

	config := device2V.ExtractFullConfig(changeStore, 0)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
	}

	for i := 0; i < len(Config2Paths); i++ {
		checkPathvalue(t, config, i,
			Config2Paths[0:11], Config2Values[0:11])
	}
}

func checkPathvalue(t *testing.T, config []ConfigValue, index int,
	expPaths []string, expValues []string) {

	// Check that they are kept in a consistent order
	if config[index].Path != expPaths[index] {
		t.Errorf("Unexpected change %d Exp: %s, Got %s", index,
			expPaths[index], config[index].Path)
	}
	if config[index].Value != expValues[index] {
		t.Errorf("Unexpected change %d Exp: %s, Got %s", index,
			expValues[index], config[index].Value)
	}
}

func Test_convertChangeToGnmi(t *testing.T) {
	setRequest := change3.GnmiChange()

	if len(setRequest.Update) != 1 {
		t.Errorf("Expected %d Update elements. Had %d",
			1, len(setRequest.Update))
	}
	jsonstr, _ := json.Marshal(setRequest.Update[0])

	expectedStr := "{\"path\":{\"elem\":[{\"name\":\"cont1a\"},{\"name\":\"cont2a\"}," +
		"{\"name\":\"leaf2c\"}]},\"val\":{\"Value\":{\"StringVal\":\"def\"}}}"
	if string(jsonstr) != expectedStr {
		t.Errorf("Expected Update[0] to be %s. Was %s",
			expectedStr, string(jsonstr))
	}

	if len(setRequest.Delete) != 1 {
		t.Errorf("Expected %d Delete elements. Had %d",
			1, len(setRequest.Delete))
	}

	jsonstr2, _ := json.Marshal(setRequest.Delete[0])

	expectedStr2 := "{\"elem\":[{\"name\":\"cont1a\"},{\"name\":\"list2a\",\"key\":{\"name\":\"txout2\"}}]}"
	if string(jsonstr2) != expectedStr2 {
		t.Errorf("Expected Delete[0] to be %s. Was %s",
			expectedStr2, string(jsonstr2))
	}
}

func Test_writeOutChangeFile(t *testing.T) {
	changeStoreFile, _ := os.Create("testout/changeStore-sample.json")
	jsonEncoder := json.NewEncoder(changeStoreFile)
	var csf = ChangeStore{Version: StoreVersion,
		Storetype: StoreTypeChange, Store: changeStore}
	err := jsonEncoder.Encode(csf)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer changeStoreFile.Close()
}

func Test_loadChangeStoreFile(t *testing.T) {
	changeStore, err := LoadChangeStore("testout/changeStore-sample.json")

	if err != nil {
		t.Errorf("Unexpected error when loading Change Store from file %s", err)
	}

	if changeStore.Version != StoreVersion {
		t.Errorf("Unexpected version %s when loading Change Store %s",
			changeStore.Version, StoreVersion)
	}
}

func Test_loadChangeStoreFileError(t *testing.T) {
	changeStore, err := LoadChangeStore("nonexistent.json")

	if err == nil {
		t.Errorf("Expected an error when loading Change Store from invalid file")
	}

	if changeStore.Version != "" {
		t.Errorf("Expected version to be empty on error. Got: %s",
			changeStore.Version)
	}
}

func Test_writeOutConfigFile(t *testing.T) {
	configStoreFile, _ := os.Create("testout/configStore-sample.json")
	jsonEncoder := json.NewEncoder(configStoreFile)
	err := jsonEncoder.Encode(ConfigurationStore{Version: StoreVersion,
		Storetype: StoreTypeConfig, Store: configurationStore})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer configStoreFile.Close()
}

func Test_loadConfigStoreFile(t *testing.T) {
	configStore, err := LoadConfigStore("testout/configStore-sample.json")

	if err != nil {
		t.Errorf("Unexpected error when loading Config Store from file %s", err)
	}

	if configStore.Version != StoreVersion {
		t.Errorf("Unexpected version %s when loading Config Store %s",
			configStore.Version, StoreVersion)
	}
}

func Test_loadConfigStoreFileError(t *testing.T) {
	configStore, err := LoadConfigStore("nonexistent.json")

	if err == nil {
		t.Errorf("Expected an error when loading Config Store from invalid file")
	}

	if configStore.Version != "" {
		t.Errorf("Expected version to be empty on error. Got: %s",
			configStore.Version)
	}
}

func BenchmarkCreateChangeValue(b *testing.B) {

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/test-%s", strconv.Itoa(b.N))
		cv, _ := CreateChangeValue(path, strconv.Itoa(i), false)
		err := cv.IsPathValid()
		if err != nil {
			b.Errorf("path not valid %s", err)
		}
	}
}

func BenchmarkCreateChange(b *testing.B) {

	changeValues := ChangeValueCollection{}
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/test%d", i)
		cv, _ := CreateChangeValue(path, strconv.Itoa(i), false)
		changeValues = append(changeValues, cv)
	}

	change, _ := CreateChange(changeValues, "Benchmarked Change")

	err := change.IsValid()
	if err != nil {
		b.Errorf("Invalid change %s", err)
	}
}

