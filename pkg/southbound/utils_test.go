package southbound

import (
	"github.com/openconfig/gnmi/proto/gnmi"
	"testing"
)

const (
    // /network-instances/network-instance[name=DEFAULT]
	elemName = "/network-instances/network-instance"
	elemNameEscaped = "/network-instances/net\\w\\o\\r\\k-instance"
	elemKeyName = "name"
	elemKeyValue = "DEFAULT"
	simplePath = elemName + "[" + elemKeyName + "=" + elemKeyValue + "]"
	escapedPath = elemNameEscaped + "[" + elemKeyName + "=" + elemKeyValue + "]"
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
	elements[0] = simplePath
	parsed, err := ParseGNMIElements(elements)

	if parsed == nil || err != nil {
		t.Errorf("path returned an error")
		return
	}

	checkElement(t, parsed, 0,  elemName, elemKeyName, elemKeyValue)
}

func Test_ParseEscape(t *testing.T) {
	elements := make([]string, 1)
	elements[0] = escapedPath
	parsed, err := ParseGNMIElements(elements)

	if parsed == nil || err != nil {
		t.Errorf("path returned an error")
		return
	}

	checkElement(t, parsed, 0, elemName, elemKeyName, elemKeyValue)
}
