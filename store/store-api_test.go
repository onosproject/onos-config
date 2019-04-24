/*
 * Copyright 2019-present Open Networking Foundation
 *
 * Licensed under the Apache License, Configuration 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package store

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	change1, change2, change3, change4 Change
)

var (
	changeStore map[string]Change
	configurationStore map[string]Configuration
)

var (
	device1V, device2V Configuration
)

const (
	TEST1_CONT1A = "/test1:cont1a"
	TEST1_CONT1A_CONT2A = "/test1:cont1a/cont2a"
	TEST1_CONT1A_CONT2A_LEAF2A = "/test1:cont1a/cont2a/leaf2a"
	TEST1_CONT1A_CONT2A_LEAF2B = "/test1:cont1a/cont2a/leaf2b"
	TEST1_CONT1A_CONT2A_LEAF2C = "/test1:cont1a/cont2a/leaf2c"
	TEST1_CONT1A_LEAF1A = "/test1:cont1a/leaf1a"
	TEST1_CONT1A_LIST2A_TXOUT1 = "/test1:cont1a/list2a=txout1"
	TEST1_CONT1A_LIST2A_TXOUT1_TXPWR = "/test1:cont1a/list2a=txout1/tx-power"
	TEST1_CONT1A_LIST2A_TXOUT2 = "/test1:cont1a/list2a=txout2"
	TEST1_CONT1A_LIST2A_TXOUT2_TXPWR = "/test1:cont1a/list2a=txout2/tx-power"
	TEST1_CONT1A_LIST2A_TXOUT3 = "/test1:cont1a/list2a=txout3"
	TEST1_CONT1A_LIST2A_TXOUT3_TXPWR = "/test1:cont1a/list2a=txout3/tx-power"
	TEST1_LEAFTOPLEVEL = "/test1:leafAtTopLevel"
)

const (
	VALUE_EMPTY            = ""
	VALUE_LEAF2A_13        = "13"
	VALUE_LEAF2B_1_59      = "1.579"
	VALUE_LEAF2B_3_14      = "3.14159"
	VALUE_LEAF2C_ABC       = "abc"
	VALUE_LEAF2C_DEF       = "def"
	VALUE_LEAF2C_GHI       = "ghi"
	VALUE_LEAF1A_ABCDEF    = "abcdef"
	VALUE_TXOUT1_TXPWR_8   = "8"
	VALUE_TXOUT2_TXPWR_10  = "10"
	VALUE_TXOUT3_TXPWR_16  = "16"
	VALUE_LEAFTOP_WXY_1234 = "WXY-1234"
)

var CONFIG1_PATHS = [11]string{
	TEST1_CONT1A, // 0
	TEST1_CONT1A_CONT2A,
	TEST1_CONT1A_CONT2A_LEAF2A,
	TEST1_CONT1A_CONT2A_LEAF2B, // 3
	TEST1_CONT1A_CONT2A_LEAF2C,
	TEST1_CONT1A_LEAF1A, // 5
	TEST1_CONT1A_LIST2A_TXOUT1,
	TEST1_CONT1A_LIST2A_TXOUT1_TXPWR,
	TEST1_CONT1A_LIST2A_TXOUT3, //10
	TEST1_CONT1A_LIST2A_TXOUT3_TXPWR,
	TEST1_LEAFTOPLEVEL,
}

var CONFIG1_VALUES = [11]string{
	VALUE_EMPTY, // 0
	VALUE_EMPTY,
	VALUE_LEAF2A_13,
	VALUE_LEAF2B_3_14, // 3
	VALUE_LEAF2C_DEF,
	VALUE_LEAF1A_ABCDEF, // 5
	VALUE_EMPTY,
	VALUE_TXOUT1_TXPWR_8,
	VALUE_EMPTY, // 10
	VALUE_TXOUT3_TXPWR_16,
	VALUE_LEAFTOP_WXY_1234,
}

var CONFIG1_PREVIOUS_PATHS = [13]string{
	TEST1_CONT1A, // 0
	TEST1_CONT1A_CONT2A,
	TEST1_CONT1A_CONT2A_LEAF2A,
	TEST1_CONT1A_CONT2A_LEAF2B, // 3
	TEST1_CONT1A_CONT2A_LEAF2C,
	TEST1_CONT1A_LEAF1A, // 5
	TEST1_CONT1A_LIST2A_TXOUT1,
	TEST1_CONT1A_LIST2A_TXOUT1_TXPWR,
	TEST1_CONT1A_LIST2A_TXOUT2,
	TEST1_CONT1A_LIST2A_TXOUT2_TXPWR,
	TEST1_CONT1A_LIST2A_TXOUT3, //10
	TEST1_CONT1A_LIST2A_TXOUT3_TXPWR,
	TEST1_LEAFTOPLEVEL,
}

var CONFIG1_PREVIOUS_VALUES = [13]string{
	VALUE_EMPTY, // 0
	VALUE_EMPTY,
	VALUE_LEAF2A_13,
	VALUE_LEAF2B_3_14, // 3
	VALUE_LEAF2C_ABC,
	VALUE_LEAF1A_ABCDEF, // 5
	VALUE_EMPTY,
	VALUE_TXOUT1_TXPWR_8,
	VALUE_EMPTY,
	VALUE_TXOUT2_TXPWR_10,
	VALUE_EMPTY, // 10
	VALUE_TXOUT3_TXPWR_16,
	VALUE_LEAFTOP_WXY_1234,
}

var CONFIG1_FIRST_PATHS = [11]string{
	TEST1_CONT1A, // 0
	TEST1_CONT1A_CONT2A,
	TEST1_CONT1A_CONT2A_LEAF2A,
	TEST1_CONT1A_CONT2A_LEAF2B, // 3
	TEST1_CONT1A_CONT2A_LEAF2C,
	TEST1_CONT1A_LEAF1A, // 5
	TEST1_CONT1A_LIST2A_TXOUT1,
	TEST1_CONT1A_LIST2A_TXOUT1_TXPWR,
	TEST1_CONT1A_LIST2A_TXOUT2,
	TEST1_CONT1A_LIST2A_TXOUT2_TXPWR,
	TEST1_LEAFTOPLEVEL, //10
}

var CONFIG1_FIRST_VALUES = [11]string{
	VALUE_EMPTY, // 0
	VALUE_EMPTY,
	VALUE_LEAF2A_13,
	VALUE_LEAF2B_1_59, // 3
	VALUE_LEAF2C_ABC,
	VALUE_LEAF1A_ABCDEF, // 5
	VALUE_EMPTY,
	VALUE_TXOUT1_TXPWR_8,
	VALUE_EMPTY,
	VALUE_TXOUT2_TXPWR_10,
	VALUE_LEAFTOP_WXY_1234, //10
}

var CONFIG2_PATHS = [11]string{
	TEST1_CONT1A, // 0
	TEST1_CONT1A_CONT2A,
	TEST1_CONT1A_CONT2A_LEAF2A,
	TEST1_CONT1A_CONT2A_LEAF2B, // 3
	TEST1_CONT1A_CONT2A_LEAF2C,
	TEST1_CONT1A_LEAF1A, // 5
	TEST1_CONT1A_LIST2A_TXOUT2,
	TEST1_CONT1A_LIST2A_TXOUT2_TXPWR,
	TEST1_CONT1A_LIST2A_TXOUT3, //10
	TEST1_CONT1A_LIST2A_TXOUT3_TXPWR,
	TEST1_LEAFTOPLEVEL,
}

var CONFIG2_VALUES = [11]string{
	VALUE_EMPTY, // 0
	VALUE_EMPTY,
	VALUE_LEAF2A_13,
	VALUE_LEAF2B_3_14, // 3
	VALUE_LEAF2C_GHI,
	VALUE_LEAF1A_ABCDEF, // 5
	VALUE_EMPTY,
	VALUE_TXOUT2_TXPWR_10,
	VALUE_EMPTY, // 10
	VALUE_TXOUT3_TXPWR_16,
	VALUE_LEAFTOP_WXY_1234,
}

func TestMain(m *testing.M) {
	var err error
	config1Value01, _ := CreateChangeValue(TEST1_CONT1A, VALUE_EMPTY, false)
	config1Value02, _ := CreateChangeValue(TEST1_CONT1A_CONT2A, VALUE_EMPTY, false)
	config1Value03, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2A, VALUE_LEAF2A_13, false)
	config1Value04, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2B, VALUE_LEAF2B_1_59, false)
	config1Value05, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2C, VALUE_LEAF2C_ABC, false)
	config1Value06, _ := CreateChangeValue(TEST1_CONT1A_LEAF1A, VALUE_LEAF1A_ABCDEF, false)
	config1Value07, _ := CreateChangeValue(TEST1_CONT1A_LIST2A_TXOUT1, VALUE_EMPTY, false)
	config1Value08, _ := CreateChangeValue(TEST1_CONT1A_LIST2A_TXOUT1_TXPWR, VALUE_TXOUT1_TXPWR_8, false)
	config1Value09, _ := CreateChangeValue(TEST1_CONT1A_LIST2A_TXOUT2, VALUE_EMPTY, false)
	config1Value10, _ := CreateChangeValue(TEST1_CONT1A_LIST2A_TXOUT2_TXPWR, VALUE_TXOUT2_TXPWR_10, false)
	config1Value11, _ := CreateChangeValue(TEST1_LEAFTOPLEVEL, VALUE_LEAFTOP_WXY_1234, false)
	change1, err = CreateChange(ChangeValueCollection{
		config1Value01, config1Value02, config1Value03, config1Value04, config1Value05,
		config1Value06, config1Value07, config1Value08, config1Value09, config1Value10,
		config1Value11,
	}, "Original Config for test switch")
	if (err != nil) {
		fmt.Println(err)
		os.Exit(-1)
	}

	config2Value01, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2B, VALUE_LEAF2B_3_14, false)
	config2Value02, _ := CreateChangeValue(TEST1_CONT1A_LIST2A_TXOUT3, VALUE_EMPTY, false)
	config2Value03, _ := CreateChangeValue(TEST1_CONT1A_LIST2A_TXOUT3_TXPWR, VALUE_TXOUT3_TXPWR_16, false)
	change2, err = CreateChange(ChangeValueCollection{
		config2Value01, config2Value02, config2Value03,
	}, "Trim power level")
	if (err != nil) {
		fmt.Println(err)
		os.Exit(-1)
	}


	config3Value01, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2C, VALUE_LEAF2C_DEF, false)
	config3Value02, _ := CreateChangeValue(TEST1_CONT1A_LIST2A_TXOUT2, VALUE_EMPTY, true)
	change3, err = CreateChange(ChangeValueCollection{
		config3Value01, config3Value02,
	}, "Remove txout 2")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	changeStore = make(map[string]Change)
	changeStore[hex.EncodeToString(change1.Id)] = change1
	changeStore[hex.EncodeToString(change2.Id)] = change2
	changeStore[hex.EncodeToString(change3.Id)] = change3

	device1V = Configuration{
		Name:    "Device1Version",
		Device:  "Device1",
		Created: time.Now(),
		Updated: time.Now(),
		User:    "onos",
		Description: "Configuration for Device 1",
		Changes: []ChangeId{change1.Id, change2.Id, change3.Id,},
	}
	configurationStore = make(map[string]Configuration)
	configurationStore["Device1Version"] = device1V


	config4Value01, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2C, VALUE_LEAF2C_GHI, false)
	config4Value02, _ := CreateChangeValue(TEST1_CONT1A_LIST2A_TXOUT1, VALUE_EMPTY, true)
	change4, err = CreateChange(ChangeValueCollection{
		config4Value01, config4Value02,
	}, "Remove txout 1")
	if (err != nil) {
		fmt.Println(err)
		os.Exit(-1)
	}
	changeStore[hex.EncodeToString(change4.Id)] = change4

	device2V = Configuration{
		Name:    "Device2VersionMain",
		Device:  "Device2",
		Created: time.Now(),
		Updated: time.Now(),
		User:    "onos",
		Description: "Main Configuration for Device 2",
		Changes: []ChangeId{change1.Id, change2.Id, change4.Id,},
	}

	configurationStore["Device2VersionMain"] = device2V

	os.Exit(m.Run())
}

func Test_1(t *testing.T) {
	// Create ConfigValue from strings
	path := "/test1:cont1a/cont2a/leaf2a"
	value := "13"

	configValue2a, _ := CreateChangeValue(path, value, false)

	if (configValue2a.Path != path) {
		t.Errorf("Retrieval of ConfigValue.Path failed. Expected %q Got %q",
			path, configValue2a.Path)
	}

	if (string(configValue2a.Value) != value) {
		t.Errorf("Retrieval of ConfigValue.Path failed. Expected %q Got %q",
			value, configValue2a.Value)
	}
}

func Test_changecreation(t *testing.T) {

	fmt.Printf("Change %x created\n", change1.Id)
	h := sha1.New()
	jsonstr, _ := json.Marshal(change1.Config)
	_, err1 := io.WriteString(h, string(jsonstr))
	_, err2 := io.WriteString(h, change1.Description)
	_, err3 := io.WriteString(h, change1.Created.String())
	if err1 != nil || err2 != nil || err3 != nil {
		t.Errorf("Error when writing objects to json")
	}
	hash := h.Sum(nil)

	expectedId := hex.EncodeToString(hash)
	actualId := hex.EncodeToString(change1.Id)
	if (actualId != expectedId) {
		t.Errorf("Creation of Change failed Expected %s got %s",
			expectedId, actualId)
	}
	err := change1.IsValid()
	if err != nil {
		t.Errorf("Checking of Change failed %s", err)
	}
}

func Test_badpath(t *testing.T) {
	badpath := "does_not_have_any_slash"
	conf1, err := CreateChangeValue(badpath, "", false)

	if err == nil {
		t.Errorf("Expected %s to produce error", badpath)
	}
	emptyChange := ChangeValue{}
	if conf1 != emptyChange {
		t.Errorf("Expected config to be empty on error")
	}
}

func Test_duplicate_path(t *testing.T) {
	cv1, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2B, VALUE_LEAF2B_3_14, false)
	cv2, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2C, VALUE_LEAF2C_ABC, false)

	cv3, _ := CreateChangeValue(TEST1_CONT1A_CONT2A_LEAF2B, VALUE_LEAF2B_1_59, false)

	change, err := CreateChange(ChangeValueCollection{cv1, cv2, cv3}, "Test Change")

	if err == nil {
		t.Errorf("Expected %s to produce error for duplicate path", TEST1_CONT1A_CONT2A_LEAF2B)
	}

	if len(change.Config) > 0 {
		t.Errorf("Expected change to be empty because of duplicate path")
	}
}

func Test_device1_version(t *testing.T) {
	fmt.Println("Configuration", device1V.Name, " (latest) Changes:")
	for idx, cid := range device1V.Changes {
		fmt.Printf("%d: %s\n", idx, base64.StdEncoding.EncodeToString([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore, 0)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
	}

	for i := 0; i < len(CONFIG1_PATHS); i++ {
		checkPathvalue(t, config, i,
			CONFIG1_PATHS[0:11], CONFIG1_VALUES[0:11])
	}
}

func Test_device1_prev_version(t *testing.T) {
	const CHANGE_PREVIOUS = 1
	fmt.Println("Configuration", device1V.Name, " (n-1) Changes:")
	for idx, cid := range device1V.Changes[0:len(device1V.Changes)-CHANGE_PREVIOUS] {
		fmt.Printf("%d: %s\n", idx, base64.StdEncoding.EncodeToString([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore, CHANGE_PREVIOUS)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
	}

	for i := 0; i < len(CONFIG1_PREVIOUS_PATHS); i++ {
		checkPathvalue(t, config, i,
			CONFIG1_PREVIOUS_PATHS[0:13], CONFIG1_PREVIOUS_VALUES[0:13])
	}
}

func Test_device1_first_version(t *testing.T) {
	const CHANGE_PREVIOUS = 2
	fmt.Println("Configuration", device1V.Name, " (n-2) Changes:")
	for idx, cid := range device1V.Changes[0:len(device1V.Changes)-CHANGE_PREVIOUS] {
		fmt.Printf("%d: %s\n", idx, base64.StdEncoding.EncodeToString([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore, CHANGE_PREVIOUS)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
	}

	for i := 0; i < len(CONFIG1_FIRST_PATHS); i++ {
		checkPathvalue(t, config, i,
			CONFIG1_FIRST_PATHS[0:11], CONFIG1_FIRST_VALUES[0:11])
	}
}

func Test_device1_invalid_version(t *testing.T) {
	const CHANGE_PREVIOUS = 3
	fmt.Println("Configuration", device1V.Name, " (n-3) Changes:")
	for idx, cid := range device1V.Changes[0:len(device1V.Changes)-CHANGE_PREVIOUS] {
		fmt.Printf("%d: %s\n", idx, base64.StdEncoding.EncodeToString([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore, CHANGE_PREVIOUS)
	if len(config) > 0 {
		t.Errorf("Not expecting any values for change (n-3). Got %d", len(config))
	}
}

func Test_device2_version(t *testing.T) {
	fmt.Println("Configuration", device2V.Name, " (latest) Changes:")
	for idx, cid := range device2V.Changes {
		fmt.Printf("%d: %s\n", idx, base64.StdEncoding.EncodeToString([]byte(cid)))
	}

	if device2V.Name != "Device2VersionMain" {
		t.Errorf("Unexpected name for Configuration main %s", device2V.Name)
	}

	config := device2V.ExtractFullConfig(changeStore, 0)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
	}

	for i := 0; i < len(CONFIG2_PATHS); i++ {
		checkPathvalue(t, config, i,
			CONFIG2_PATHS[0:11], CONFIG2_VALUES[0:11])
	}
}

func checkPathvalue(t *testing.T, config []ConfigValue, index int,
		expPaths []string, expValues []string) {

	// Check that they are kept in a consistent order
	if strings.Compare(config[index].Path, expPaths[index]) != 0 {
		t.Errorf("Unexpected change %d Exp: %s, Got %s", index,
			expPaths[index], config[index].Path)
	}
	if strings.Compare(config[index].Value, expValues[index]) != 0 {
		t.Errorf("Unexpected change %d Exp: %s, Got %s", index,
			expValues[index], config[index].Value)
	}
}

func Test_convertChangeToGnmi(t *testing.T) {
	setRequest := change3.GnmiChange()

	if (len(setRequest.Update) != 1) {
		t.Errorf("Expected %d Update elements. Had %d",
			1, len(setRequest.Update))
	}
	jsonstr, _ := json.Marshal(setRequest.Update[0])

	expectedStr := "{\"path\":{\"elem\":[{\"name\":\"cont1a\"},{\"name\":\"cont2a\"}," +
		"{\"name\":\"leaf2c\"}]},\"val\":{\"Value\":{\"StringVal\":\"def\"}}}"
	if (strings.Compare(string(jsonstr), expectedStr) != 0) {
		t.Errorf("Expected Update[0] to be %s. Was %s",
			expectedStr, string(jsonstr))
	}



	if (len(setRequest.Delete) != 1) {
		t.Errorf("Expected %d Delete elements. Had %d",
			1, len(setRequest.Delete))
	}

	jsonstr2, _ := json.Marshal(setRequest.Delete[0])

	expectedStr2 := "{\"elem\":[{\"name\":\"cont1a\"},{\"name\":\"list2a\",\"key\":{\"name\":\"txout2\"}}]}"
	if (strings.Compare(string(jsonstr2), expectedStr2) != 0) {
		t.Errorf("Expected Delete[0] to be %s. Was %s",
			expectedStr2, string(jsonstr2))
	}
}

func Test_writeOutChangeFile(t *testing.T) {
	changeStoreFile, _ := os.Create("testout/changeStore-sample.json")
	jsonEncoder := json.NewEncoder(changeStoreFile)
	var csf ChangeStore = ChangeStore{Version:STORE_VERSION,
		Storetype:STORE_TYPE_CHANGE, Store:changeStore}
	err := jsonEncoder.Encode(csf)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer changeStoreFile.Close()
}

func Test_writeOutConfigFile(t *testing.T) {
	configStoreFile, _ := os.Create("testout/configStore-sample.json")
	jsonEncoder := json.NewEncoder(configStoreFile)
	err := jsonEncoder.Encode(ConfigurationStore{Version:STORE_VERSION,
		Storetype:STORE_TYPE_CONFIG, Store:configurationStore})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer configStoreFile.Close()
}
