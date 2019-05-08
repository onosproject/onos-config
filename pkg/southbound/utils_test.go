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

	// /test1:cont1a/list2a=txout1/tx-power
	elemName2 = "/test/test"
	elemKeyName2 = "list2a"
	elemKeyValue2 = "txout1"
	path2 = "/test/test[list2a=txout1]"

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
