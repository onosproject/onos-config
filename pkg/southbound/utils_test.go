package southbound

import (
	"github.com/openconfig/gnmi/proto/gnmi"
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
)

func checkElement(t *testing.T, parsed *gnmi.Path, index int, elemName string, elemKeyName string, elemKeyValue string) {
	elem := parsed.Elem[index]

	if elem == nil {
		t.Errorf("path Element %d does not exist", index)
		return
	}
	name := elem.Name
	if name != elemName {
		t.Errorf("path Element 0 name is incorrect %s", name)
		return
	}

	key := elem.Key
	if key[elemKeyName] != elemKeyValue {
		t.Errorf("key 0 is incorrect %s", key[elemKeyName])
		return
	}
}

func Test_ParseSimple(t *testing.T) {
	elements := make([]string, 1)
	elements[0] = path1
	parsed, err := ParseGNMIElements(elements)

	if parsed == nil || err != nil {
		t.Errorf("path returned an error")
		return
	}

	checkElement(t, parsed, 0,  elemName1, elemKeyName1, elemKeyValue1)
}

func Test_ParseEscape(t *testing.T) {
	elements := make([]string, 1)
	elements[0] = escapedPath1
	parsed, err := ParseGNMIElements(elements)

	if parsed == nil || err != nil {
		t.Errorf("path returned an error")
		return
	}

	checkElement(t, parsed, 0, elemName1, elemKeyName1, elemKeyValue1)
}

func Test_ParseMultiple(t *testing.T) {
	elements := make([]string, 2)
	elements[0] = path1
	elements[1] = path2
	parsed, err := ParseGNMIElements(elements)

	if parsed == nil || err != nil {
		t.Errorf("path returned an error")
		return
	}

	checkElement(t, parsed, 0,  elemName1, elemKeyName1, elemKeyValue1)
	checkElement(t, parsed, 1,  elemName2, elemKeyName2, elemKeyValue2)
}

func Test_Split(t *testing.T) {
	paths := make([]string, 2)
	paths[0] = pathToSplit
	paths[1] = path2ToSplit
	splitPaths := SplitPaths(paths)
	if len(splitPaths) != 2 {
		t.Errorf("paths split length is incorrect %d", len(splitPaths))
		return
	}

	splitPaths1 := splitPaths[0]
	if splitPaths1[0] != pathSegment1 {
		t.Errorf("paths split segment 1 is incorrect %s", splitPaths[0])
		return
	}

	if splitPaths1[3] != pathSegment4 {
		t.Errorf("paths split segment 4 is incorrect %s", splitPaths[3])
		return
	}
}
