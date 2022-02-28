package path

import (
	"fmt"
	"github.com/openconfig/goyang/pkg/yang"
	"gotest.tools/assert"
	"testing"
)

// test the most basic scenario with one RO and one RW path, no recursion
func TestExtractPathsSimpleCase(t *testing.T) {
	var (
		roLeaf = &yang.Entry{
			Name:   "roLeaf",
			Config: yang.TSFalse,
			Type: &yang.YangType{
				Name: "string",
				Kind: yang.Ystring,
			},
		}
		rwLeaf = &yang.Entry{
			Name:   "rwLeaf",
			Config: yang.TSTrue,
			Type: &yang.YangType{
				Name: "string",
				Kind: yang.Ystring,
			},
			Default: "default",
		}
		simpleEntry = &yang.Entry{
			Name:   "device",
			Kind:   yang.DirectoryEntry,
			Config: yang.TSUnset,
			Annotation: map[string]interface{}{
				"isFakeRoot": true,
				"schemapath": "/",
				"structname": "Device",
			},
		}
	)

	// populate the yang tree
	roLeaf.Parent = simpleEntry
	rwLeaf.Parent = simpleEntry
	simpleEntry.Dir = map[string]*yang.Entry{
		"roLeaf": roLeaf,
		"rwLeaf": rwLeaf,
	}

	ro, rw := ExtractPaths(simpleEntry, yang.TSUnset, "", "")
	fmt.Println(ro.PrettyString(), rw.PrettyString())
	assert.Equal(t, len(ro), 1)
	assert.Equal(t, len(rw), 1)

	assert.Equal(t, ro["/roLeaf"]["/"].AttrName, "roLeaf")
	assert.Equal(t, rw["/rwLeaf"].Default, "default")
	assert.Equal(t, rw["/rwLeaf"].ReadOnlyAttrib.AttrName, "rwLeaf")
}

// test recursion with no RO paths
func TestExtractPathsNested(t *testing.T) {
	var (
		testDeviceEntry = &yang.Entry{
			Name:   "device",
			Kind:   yang.DirectoryEntry,
			Config: yang.TSUnset,
			Annotation: map[string]interface{}{
				"isFakeRoot": true,
				"schemapath": "/",
				"structname": "Device",
			},
			// Dir: cont1a
		}
		cont1a = &yang.Entry{
			Name: "cont1a",
			Kind: yang.DirectoryEntry,
			Config: yang.TSUnset,
			// Dir: list2a
		}
		list2a = &yang.Entry{
			Name: "list2a",
			Kind: yang.DirectoryEntry,
			Config: yang.TSUnset,
			ListAttr: &yang.ListAttr{
				MinElements: 0,
				MaxElements: 4,
				OrderedBy:   nil,
			},
			Key: "name",
			// Dir: name
		}
		name = &yang.Entry{
			Name: "name",
			Kind: yang.LeafEntry,
			Config: yang.TSUnset,
			Type: &yang.YangType{
				Name: "string",
				Kind: yang.Ystring,
			},
		}
	)

	name.Parent = list2a

	list2a.Parent = cont1a
	list2a.Dir = map[string]*yang.Entry{
		"name": name,
	}

	cont1a.Parent = testDeviceEntry
	cont1a.Dir = map[string]*yang.Entry{
		"list2a": list2a,
	}

	testDeviceEntry.Dir = map[string]*yang.Entry{
		"cont1a": cont1a,
	}

	ro, rw := ExtractPaths(testDeviceEntry, yang.TSUnset, "", "")
	fmt.Println(ro.PrettyString(), rw.PrettyString())
	assert.Equal(t, len(ro), 0)
	assert.Equal(t, len(rw), 1)
}

// test recursion with RO paths
func TestExtractPathsNestedRO(t *testing.T) {
	var (
		testDeviceEntry = &yang.Entry{
			Name:   "device",
			Kind:   yang.DirectoryEntry,
			Config: yang.TSUnset,
			Annotation: map[string]interface{}{
				"isFakeRoot": true,
				"schemapath": "/",
				"structname": "Device",
			},
			// Dir: cont1a
		}
		cont1a = &yang.Entry{
			Name: "cont1a",
			Kind: yang.DirectoryEntry,
			Config: yang.TSUnset,
			// Dir: list2a
		}
		list2a = &yang.Entry{
			Name: "list2a",
			Kind: yang.DirectoryEntry,
			Config: yang.TSUnset,
			ListAttr: &yang.ListAttr{
				MinElements: 0,
				MaxElements: 4,
				OrderedBy:   nil,
			},
			Key: "name",
			// Dir: name
		}
		name = &yang.Entry{
			Name: "name",
			Kind: yang.LeafEntry,
			Config: yang.TSFalse,
			Type: &yang.YangType{
				Name: "string",
				Kind: yang.Ystring,
			},
		}
	)

	name.Parent = list2a

	list2a.Parent = cont1a
	list2a.Dir = map[string]*yang.Entry{
		"name": name,
	}

	cont1a.Parent = testDeviceEntry
	cont1a.Dir = map[string]*yang.Entry{
		"list2a": list2a,
	}

	testDeviceEntry.Dir = map[string]*yang.Entry{
		"cont1a": cont1a,
	}

	ro, rw := ExtractPaths(testDeviceEntry, yang.TSUnset, "", "")
	fmt.Println(ro.PrettyString())
	assert.Equal(t, len(ro), 1)
	assert.Equal(t, len(rw), 0)
}