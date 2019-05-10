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

package utils

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"testing"
)

const (
    // /network-instances/network-instance[name=DEFAULT]
	elemName1a = "network-instances"
	elemName1b = "network-instance"
	elemNameEscaped1a = "netwo\\r\\k\\-\\instances"
	elemNameEscaped1b = "net\\w\\o\\r\\k-instance"
	elemKeyName1 = "name"
	elemKeyValue1 = "DEFAULT"
	path1 = "/" + elemName1a + "[" + elemKeyName1 + "=" + elemKeyValue1 + "]/" + elemName1b
	escapedPath1 = "/" + elemNameEscaped1a + "[" + elemKeyName1 + "=" + elemKeyValue1 + "]/" + elemNameEscaped1b

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
	assert.Assert(t, name == elemName, "path Element %d name is incorrect %s", index, name)

	key := elem.Key
	assert.Assert(t, key[elemKeyName] == elemKeyValue, "key 0 is incorrect %s", key[elemKeyName])
}

func Test_ParseSimple(t *testing.T) {
	elements := SplitPath(path1)
	parsed, err := ParseGNMIElements(elements)

	assert.Assert(t, parsed != nil && err == nil, "path returned an error")

	checkElement(t, parsed, 0,  elemName1a, elemKeyName1, elemKeyValue1)
}

func Test_ParseEscape(t *testing.T) {
	elements := SplitPath(escapedPath1)
	parsed, err := ParseGNMIElements(elements)

	assert.Assert(t, parsed != nil && err == nil, "path returned an error")

	checkElement(t, parsed, 0, elemName1a, elemKeyName1, elemKeyValue1)
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

func Test_ParseNamespace(t *testing.T) {
	elements := SplitPath("/ns:a[x=y]/b/c")
	parsed, err := ParseGNMIElements(elements)

	assert.NilError(t, err, "Path with NS returns error")
	assert.Assert(t, parsed != nil,"path with NS returns nil")

	checkElement(t, parsed, 0,  "a", "x", "y")
}

func Test_StrPath(t *testing.T) {
	elements := SplitPath(path1)
	parsed, err := ParseGNMIElements(elements)
	assert.NilError(t, err)

	generatedPath := StrPath(parsed)
	assert.Equal(t, generatedPath, path1)
}

func Test_StrPathV03(t *testing.T) {
	const path = "/a/b/c"
	elements := SplitPath(path)
	parsed, err := ParseGNMIElements(elements)
	assert.NilError(t, err)

	//  V3 used Element
	parsed.Element = make([]string, len(parsed.Elem))
	for i, elm := range parsed.Elem {
		parsed.Element[i] = elm.Name
	}
	generatedPath := strPathV03(parsed)
	assert.Equal(t, generatedPath, path)
}

func Test_StrPathV04(t *testing.T) {
	elements := SplitPath(path1)
	parsed, err := ParseGNMIElements(elements)
	assert.NilError(t, err)

	generatedPath := strPathV04(parsed)
	assert.Equal(t, generatedPath, path1)
}
