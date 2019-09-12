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
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	"gotest.tools/assert"
	"io/ioutil"
	"regexp"
	"strings"
	"testing"
)

func setUpJSONToValues(filename string) ([]byte, error) {
	sampleTree, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return sampleTree, nil
}

func Test_DecomposeTree(t *testing.T) {
	const matchAllChars = `(/[a-zA-Z0-9\-:\[\]]*)*`
	const matchOnIndex = `(\[[0-9]]).*?`
	sampleTree, err := setUpJSONToValues("./testdata/sample-tree.json")
	assert.NilError(t, err)

	assert.Assert(t, len(sampleTree) > 0, "Empty sample tree", (len(sampleTree)))

	values, err := DecomposeTree(sampleTree)
	assert.NilError(t, err)

	for _, v := range values {
		fmt.Printf("%s %s\n", (*v).Path, (*v).String())
	}
	assert.Equal(t, len(values), 22)

	rAllChars := regexp.MustCompile(matchAllChars)
	rOnIndex := regexp.MustCompile(matchOnIndex)
	for _, v := range values {
		match := rAllChars.FindString(v.Path)
		assert.Equal(t, match, v.Path)

		matches := rOnIndex.FindAllStringSubmatch(v.Path, -1)
		newPath := v.Path
		for _, m := range matches {
			newPath = strings.Replace(newPath, m[1], "[*]", -1)
		}

		switch newPath {
		case
			"/interfaces/default-type",
			"/interfaces/interface[*]/type",
			"/system/openflow/controllers/controller[*]/name",
			"/system/openflow/controllers/controller[*]/type",
			"/system/openflow/controllers/controller[*]/connections/example-ll[*]",
			"/system/openflow/controllers/controller[*]/connections/connection[*]/conn-type",
			"/interfaces/interface[*]/name":
			assert.Equal(t, v.Type, change.ValueTypeSTRING, newPath)
		case
			"/system/openflow/controllers/controller[*]/connections/connection[*]/discombobulator":
			assert.Equal(t, v.Type, change.ValueTypeBOOL, newPath)
		case
			"/system/openflow/controllers/controller[*]/connections/connections-type",
			"/system/openflow/controllers/controller[*]/connections/connection[*]/aux-id",
			"/system/openflow/controllers/controller[*]/connections/connections-freq":
			assert.Equal(t, v.Type, change.ValueTypeFLOAT, newPath)
		default:
			t.Fatal("Unexpected jsonPath", newPath)
		}
	}
}

func Test_DecomposeTree2(t *testing.T) {
	const matchAllChars = `(/[a-zA-Z0-9\-:\[\]]*)*`
	const matchOnIndex = `(\[[0-9]]).*?`
	sampleTree, err := setUpJSONToValues("./testdata/sample-tree-from-devicesim.json")
	assert.NilError(t, err)

	assert.Assert(t, len(sampleTree) > 0, "Empty sample tree", (len(sampleTree)))

	values, err := DecomposeTree(sampleTree)
	assert.NilError(t, err)

	for _, v := range values {
		fmt.Printf("%s %s\n", (*v).Path, (*v).String())
	}
	assert.Equal(t, len(values), 31)

	rAllChars := regexp.MustCompile(matchAllChars)
	rOnIndex := regexp.MustCompile(matchOnIndex)
	for _, v := range values {
		match := rAllChars.FindString(v.Path)
		assert.Equal(t, match, v.Path)

		matches := rOnIndex.FindAllStringSubmatch(v.Path, -1)
		newPath := v.Path
		for _, m := range matches {
			newPath = strings.Replace(newPath, m[1], "[*]", -1)
		}

		switch newPath {
		case "/interfaces/interface[*]/name",
			"/system/openflow/controllers/controller[*]/name",
			"/system/openflow/controllers/controller[*]/connections/connection[*]/state/address",
			"/system/openflow/controllers/controller[*]/connections/connection[*]/state/source-interface",
			"/system/openflow/controllers/controller[*]/connections/connection[*]/state/transport":
			assert.Equal(t, v.Type, change.ValueTypeSTRING, newPath)
		case
			"/system/openflow/controllers/controller[*]/connections/connection[*]/aux-id",
			"/system/openflow/controllers/controller[*]/connections/connection[*]/state/aux-id",
			"/system/openflow/controllers/controller[*]/connections/connection[*]/state/priority",
			"/system/openflow/controllers/controller[*]/connections/connection[*]/state/port":
			assert.Equal(t, v.Type, change.ValueTypeFLOAT, newPath)
		default:
			t.Fatal("Unexpected jsonPath", newPath)
		}
	}
}
