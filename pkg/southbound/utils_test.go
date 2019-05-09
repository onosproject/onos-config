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

package southbound

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"testing"
)

const (
    // /network-instances/network-instance[name=DEFAULT]
	elemName1 = "/network-instances/network-instance"
	elemNameEscaped1 = "/network-instances/net\\w\\o\\r\\k-instance"
	elemKeyName1 = "name"
	elemKeyValue1 = "DEFAULT"
	path1 = elemName1 + "[" + elemKeyName1 + "=" + elemKeyValue1 + "]"
	escapedPath1 = elemNameEscaped1 + "[" + elemKeyName1 + "=" + elemKeyValue1 + "]"

	// /test1[list2a=txout1]
	elemName2 = "/test1"
	elemKeyName2 = "list2a"
	elemKeyValue2 = "txout1"
	path2 = elemName2 + "[" + elemKeyName2 + "=" + elemKeyValue2 + "]"

	pathSegment1 = "1"
	pathSegment2 = "2[a=b]"
	pathSegment3 = "3"
	pathSegment4 = "\\4"
	pathToSplit = "/" + pathSegment1 + "/" + pathSegment2 + "/" + pathSegment3 + "/" + pathSegment4

	path2Segment1 = "11"
	path2Segment2 = "22[a=b]"
	path2Segment3 = "33"
	path2Segment4 = "\\44"
	path2ToSplit = "/" + path2Segment1 + "/" + path2Segment2 + "/" + path2Segment3 + "/" + path2Segment4

	pathNoClose = "/a/b[name=aaa"
	pathNoKeyName = "/a/b/[=aaa]"
)

func checkElement(t *testing.T, parsed *gnmi.Path, index int, elemName string, elemKeyName string, elemKeyValue string) {
	elem := parsed.Elem[index]
	assert.Assert(t, elem != nil, "path Element %d does not exist", index)

	name := elem.Name
	assert.Assert(t, name == elemName, "path Element 0 name is incorrect %s", name)

	key := elem.Key
	assert.Assert(t, key[elemKeyName] == elemKeyValue, "key 0 is incorrect %s", key[elemKeyName])
}

func Test_ParseSimple(t *testing.T) {
	elements := make([]string, 1)
	elements[0] = path1
	parsed, err := ParseGNMIElements(elements)

	assert.Assert(t, parsed != nil && err == nil, "path returned an error")

	checkElement(t, parsed, 0,  elemName1, elemKeyName1, elemKeyValue1)
}

func Test_ParseEscape(t *testing.T) {
	elements := make([]string, 1)
	elements[0] = escapedPath1
	parsed, err := ParseGNMIElements(elements)

	assert.Assert(t, parsed != nil && err == nil, "path returned an error")

	checkElement(t, parsed, 0, elemName1, elemKeyName1, elemKeyValue1)
}

func Test_ParseMultiple(t *testing.T) {
	elements := make([]string, 2)
	elements[0] = path1
	elements[1] = path2
	parsed, err := ParseGNMIElements(elements)

	assert.Assert(t, parsed != nil && err == nil,"path returned an error")

	checkElement(t, parsed, 0,  elemName1, elemKeyName1, elemKeyValue1)
	checkElement(t, parsed, 1,  elemName2, elemKeyName2, elemKeyValue2)
}

func Test_ParseErrorNoClose(t *testing.T) {
	elements := make([]string, 1)
	elements[0] = pathNoClose
	parsed, err := ParseGNMIElements(elements)

	assert.Assert(t, err != nil && parsed == nil,"path with no closing bracket did not generate an error")
}

func Test_ParseErrorNoKeyName(t *testing.T) {
	elements := make([]string, 1)
	elements[0] = pathNoKeyName
	parsed, err := ParseGNMIElements(elements)

	assert.Assert(t, err != nil && parsed == nil,"path with no opening bracket did not generate an error")
}

func Test_Split(t *testing.T) {
	paths := make([]string, 2)
	paths[0] = pathToSplit
	paths[1] = path2ToSplit
	splitPaths := SplitPaths(paths)
	assert.Equal(t, len(splitPaths), 2)

	splitPaths1 := splitPaths[0]
	assert.Equal(t, splitPaths1[0], pathSegment1)
	assert.Equal(t, splitPaths1[3], pathSegment4)
}
