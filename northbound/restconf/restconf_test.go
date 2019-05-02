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

package restconf

import (
	"encoding/json"
	"fmt"
	"github.com/opennetworkinglab/onos-config/store"
	"os"
	"testing"
)

const (
	configStoreDefaultFileName = "../../store/testout/configStore-sample.json"
	changeStoreDefaultFileName = "../../store/testout/changeStore-sample.json"
)

var (
	configStoreTest store.ConfigurationStore
	changeStoreTest store.ChangeStore
)

func TestMain(m *testing.M) {
	var err error
	configStoreTest, err = store.LoadConfigStore(configStoreDefaultFileName)
	if err != nil {
		wd, _ := os.Getwd()
		fmt.Println("Cannot load config store ", err, wd)
		return
	}
	fmt.Println("Configuration store loaded from", configStoreDefaultFileName)

	changeStoreTest, err = store.LoadChangeStore(changeStoreDefaultFileName)
	if err != nil {
		fmt.Println("Cannot load change store ", err)
		return
	}

	os.Exit(m.Run())
}

func Test_device1_tree(t *testing.T) {
	device1V := configStoreTest.Store["Device1Version"]
	fmt.Println("Configuration", device1V.Name, " (latest) Changes:")

	configTree, err := BuildTree(device1V.ExtractFullConfig(changeStoreTest.Store, 0))
	if err != nil {
		t.Errorf("Unexpected error in extracting full tree %s", err)
	}
	const fullTreeLen = 244
	if len(configTree) != fullTreeLen {
		t.Errorf("Unexpected length %d extracting full tree. Got %d",
			fullTreeLen, len(configTree))
	}

	const fullTree = `{"(root)": [{"test1:cont1a": [{"cont2a": [{"leaf2a":"13",` +
		`"leaf2b":"3.14159","leaf2c":"def"}]},{"list2a": [{"id":"txout1","tx-power":"8"}]},` +
		`{"list2a": [{"id":"txout3","tx-power":"16"}]},{"leaf1a":"abcdef"}]},` +
		`{"test1:leafAtTopLevel":"WXY-1234"}]}`

	if string(configTree) != fullTree {
		t.Errorf("Expecting full tree. %s Got %s",
			fullTree, string(configTree))
	}

	var data interface{}
	err = json.Unmarshal(configTree, &data)
	if err != nil {
		t.Errorf("Unexpected error unmarshalling full tree %s", err.Error())
	}
	dataMap := data.(map[string]interface{})
	treeObj := dataMap["(root)"]
	topObjectsSlice := treeObj.([]interface{})
	if len(topObjectsSlice) != 2 {
		t.Errorf("Unexpected tree map from full tree %v", len(topObjectsSlice))
	}

	treeSampleFile, err := os.Create("../../store/testout/tree-sample.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer treeSampleFile.Close()

	fmt.Fprintln(treeSampleFile, string(configTree))

}
