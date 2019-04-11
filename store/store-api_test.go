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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

var change1 Change
var change2 Change
var change3 Change
var change4 Change

var changeStore map[string]Change

var device1V Configuration
var device2V Configuration

func TestMain(m *testing.M) {
	var err error
	config1Value01, _ := CreateChangeValue("/test1:cont1a", "", false)
	config1Value02, _ := CreateChangeValue("/test1:cont1a/cont2a", "", false)
	config1Value03, _ := CreateChangeValue("/test1:cont1a/cont2a/leaf2a", "13", false)
	config1Value04, _ := CreateChangeValue("/test1:cont1a/cont2a/leaf2b", "1.579", false)
	config1Value05, _ := CreateChangeValue("/test1:cont1a/cont2a/leaf2c", "abc", false)
	config1Value06, _ := CreateChangeValue("/test1:cont1a/leaf1a", "abcdef", false)
	config1Value07, _ := CreateChangeValue("/test1:cont1a/list2a=txout1", "", false)
	config1Value08, _ := CreateChangeValue("/test1:cont1a/list2a=txout1/tx-power", "8", false)
	config1Value09, _ := CreateChangeValue("/test1:cont1a/list2a=txout2", "", false)
	config1Value10, _ := CreateChangeValue("/test1:cont1a/list2a=txout2/tx-power", "10", false)
	config1Value11, _ := CreateChangeValue("/test1:leafAtTopLevel", "WXY-1234", false)
	change1, err = CreateChange(ChangeValueCollection{
		config1Value01, config1Value02, config1Value03, config1Value04, config1Value05,
		config1Value06, config1Value07, config1Value08, config1Value09, config1Value10,
		config1Value11,
	}, "Original Config for test switch")
	if (err != nil) {
		fmt.Println(err)
		os.Exit(-1)
	}

	config2Value01, _ := CreateChangeValue("/test1:cont1a/cont2a/leaf2b", "3.14159", false)
	config2Value02, _ := CreateChangeValue("/test1:cont1a/list2a=txout3", "", false)
	config2Value03, _ := CreateChangeValue("/test1:cont1a/list2a=txout3/tx-power", "16", false)
	change2, err = CreateChange(ChangeValueCollection{
		config2Value01, config2Value02, config2Value03,
	}, "Trim power level")
	if (err != nil) {
		fmt.Println(err)
		os.Exit(-1)
	}


	config3Value01, _ := CreateChangeValue("/test1:cont1a/cont2a/leaf2c", "def", false)
	config3Value02, _ := CreateChangeValue("/test1:cont1a/list2a=txout2", "", true)
	change3, err = CreateChange(ChangeValueCollection{
		config3Value01, config3Value02,
	}, "Remove txout 2")
	if (err != nil) {
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


	config4Value01, _ := CreateChangeValue("/test1:cont1a/cont2a/leaf2c", "def", false)
	config4Value02, _ := CreateChangeValue("/test1:cont1a/list2a=txout2", "", true)
	change4, err = CreateChange(ChangeValueCollection{
		config4Value01, config4Value02,
	}, "Remove txout 2")
	if (err != nil) {
		fmt.Println(err)
		os.Exit(-1)
	}

	device2V = Configuration{
		Name:    "Device2VersionMain",
		Device:  "Device2",
		Created: time.Now(),
		Updated: time.Now(),
		User:    "onos",
		Description: "Main Configuration for Device 2",
		Changes: []ChangeId{change1.Id, change2.Id, change4.Id,},
	}

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

func Test_device1_version(t *testing.T) {
	fmt.Println("Configuration" + device1V.Name + " Change:")
	for idx, cid := range device1V.Changes {
		fmt.Printf("%d: %s\n", idx, hex.EncodeToString([]byte(cid)))
	}

	if device1V.Name != "Device1Version" {
		t.Errorf("Unexpected name for Configuration main %s", device1V.Name)
	}

	config := device1V.ExtractFullConfig(changeStore)
	for _, c := range config {
		fmt.Printf("Path %s = %s\n", c.Path, c.Value)
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

