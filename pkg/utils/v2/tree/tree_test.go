// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package tree

import (
	"encoding/binary"
	"fmt"
	testdevice_1_0_0 "github.com/onosproject/config-models/models/testdevice-1.0.x/api"
	"testing"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"gotest.tools/assert"
)

const testJSON1 = `{
  "cont1a": {
    "cont2a": {
      "leaf2a": 12,
      "leaf2b": "1.141590",
      "leaf2c": "myvalue1a2c",
      "leaf2d": "1.14159",
      "leaf2e": [
        12,
        0,
        305,
        -32768,
        32767
      ],
      "leaf2f": "YXMgYnl0ZSBhcnJheQ==",
      "leaf2g": true
    },
    "leaf1a": "myvalue1a1a",
    "list2a": [
      {
        "name": "First",
        "range-max": 7,
        "range-min": 6,
        "tx-power": 5
      },
      {
        "name": "Second",
        "range-max": 9,
        "range-min": 8,
        "tx-power": 7
      }
    ],
    "list5": [
      {
        "key1": "five",
        "key2": 5,
        "leaf5a": "5A five-5"
      },
      {
        "key1": "one",
        "key2": 1,
        "leaf5a": "5A one-1"
      },
      {
        "key1": "two",
        "key2": 1,
        "leaf5a": "5A two-1"
      },
      {
        "key1": "two",
        "key2": 2,
        "leaf5a": "5A two-2"
      },
      {
        "key1": "two",
        "key2": 3,
        "leaf5a": "5A two-3"
      }
    ]
  },
  "leafAtTopLevel": "WXY-1234"
}`

const (
	Test1Cont1ALeaf1a       = "/cont1a/leaf1a"
	Test1Cont1AList2a       = "/cont1a/list2a"
	Test1Cont1AList2a1      = "/cont1a/list2a[name=First]"
	Test1Cont1AList2a1t     = "/cont1a/list2a[name=First]/tx-power"
	Test1Cont1AList2a1min   = "/cont1a/list2a[name=First]/range-min"
	Test1Cont1AList2a1max   = "/cont1a/list2a[name=First]/range-max"
	Test1Cont1AList2a2t     = "/cont1a/list2a[name=Second]/tx-power"
	Test1Cont1AList2a2min   = "/cont1a/list2a[name=Second]/range-min"
	Test1Cont1AList2a2max   = "/cont1a/list2a[name=Second]/range-max"
	Test1Cont1ACont2ALeaf2A = "/cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B = "/cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C = "/cont1a/cont2a/leaf2c"
	Test1Cont1ACont2ALeaf2D = "/cont1a/cont2a/leaf2d"
	Test1Cont1ACont2ALeaf2E = "/cont1a/cont2a/leaf2e"
	Test1Cont1ACont2ALeaf2F = "/cont1a/cont2a/leaf2f"
	Test1Cont1ACont2ALeaf2G = "/cont1a/cont2a/leaf2g"
	Test1Leaftoplevel       = "/leafAtTopLevel"
	Test1list5One1          = "/cont1a/list5[key1=one][key2=1]"
	Test1list5One1K1        = Test1list5One1 + "/key1"
	Test1list5One1K2        = Test1list5One1 + "/key2"
	Test1list5Two1          = "/cont1a/list5[key1=two][key2=1]"
	Test1list5Two1K1        = Test1list5Two1 + "/key1"
	Test1list5Two1K2        = Test1list5Two1 + "/key2"
	Test1list5Two2          = "/cont1a/list5[key1=two][key2=2]"
	Test1list5Two2K1        = Test1list5Two2 + "/key1"
	Test1list5Two2K2        = Test1list5Two2 + "/key2"
	Test1list5Two3          = "/cont1a/list5[key1=two][key2=3]"
	Test1list5Two3K1        = Test1list5Two3 + "/key1"
	Test1list5Two3K2        = Test1list5Two3 + "/key2"
	Test1list5Five5         = "/cont1a/list5[key1=five][key2=5]"
	Test1list5Five5K1       = Test1list5Five5 + "/key1"
	Test1list5Five5K2       = Test1list5Five5 + "/key2"
	Test1list5One1L5a       = "/cont1a/list5[key1=one][key2=1]/leaf5a"
	Test1list5Two1L5a       = "/cont1a/list5[key1=two][key2=1]/leaf5a"
	Test1list5Two2L5a       = "/cont1a/list5[key1=two][key2=2]/leaf5a"
	Test1list5Two3L5a       = "/cont1a/list5[key1=two][key2=3]/leaf5a"
	Test1list5Five5L5a      = "/cont1a/list5[key1=five][key2=5]/leaf5a"
)

const (
	ValueLeaf2A12       = 12
	ValueLeaf2B114      = 1.14159 // AAAA4PND8j8=
	ValueLeaf2CMyValue  = "myvalue1a2c"
	ValueLeaf1AMyValue  = "myvalue1a1a"
	ValueLeaf2D114      = 114159 // precision 5
	ValueLeaf2EInt1     = 12
	ValueLeaf2EInt2     = 0
	ValueLeaf2EInt3     = 305
	ValueLeaf2EInt4     = -32768
	ValueLeaf2EInt5     = 32767
	ValueList2a1PwrT    = 5
	ValueList2a1Rmin    = 6
	ValueList2a1Rmax    = 7
	ValueList2a2PwrT    = 7
	ValueList2a2Rmin    = 8
	ValueList2a2Rmax    = 9
	ValueList2A2F       = "as byte array"
	ValueList2A2G       = true
	ValueLeaftopWxy1234 = "WXY-1234"
	ValueList5aOne1L5a  = "5A one-1"
	ValueList5aTwo1L5a  = "5A two-1"
	ValueList5aTwo2L5a  = "5A two-2"
	ValueList5aTwo3L5a  = "5A two-3"
	ValueList5aFive5L5a = "5A five-5"
)

var (
	configValues []*configapi.PathValue
)

func setUpTree() {
	configValues = make([]*configapi.PathValue, 30)
	configValues[0] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2A, Value: *configapi.NewTypedValueUint(ValueLeaf2A12, 8)}
	configValues[1] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2B, Value: *configapi.NewTypedValueFloat(ValueLeaf2B114)}

	configValues[2] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2C, Value: *configapi.NewTypedValueString(ValueLeaf2CMyValue)}
	configValues[3] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2D, Value: *configapi.NewTypedValueDecimal(ValueLeaf2D114, 5)}

	// Leaf list of int16 has no special constructor - use the generic NewTypedValue()
	valueLeaf2EBytes := []byte{
		ValueLeaf2EInt1,
		ValueLeaf2EInt2,
	}
	int3Bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(int3Bytes, ValueLeaf2EInt3)
	valueLeaf2EBytes = append(valueLeaf2EBytes, int3Bytes...)
	int4Bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(int4Bytes, ValueLeaf2EInt4*-1)
	valueLeaf2EBytes = append(valueLeaf2EBytes, int4Bytes...)
	int5Bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(int5Bytes, ValueLeaf2EInt5)
	valueLeaf2EBytes = append(valueLeaf2EBytes, int5Bytes...)
	opts := []uint8{16, 1, 0, 1, 0, 2, 0, 2, 1, 2, 0} // int width and pair (#bytes and neg) per entry
	leaf2EValue, err := configapi.NewTypedValue(valueLeaf2EBytes, configapi.ValueType_LEAFLIST_INT, opts)
	if err != nil {
		panic(err)
	}
	configValues[4] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2E, Value: *leaf2EValue}

	configValues[5] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2F, Value: *configapi.NewTypedValueBytes([]byte(ValueList2A2F))}
	configValues[6] = &configapi.PathValue{Path: Test1Cont1ACont2ALeaf2G, Value: *configapi.NewTypedValueBool(ValueList2A2G)}
	configValues[7] = &configapi.PathValue{Path: Test1Cont1ALeaf1a, Value: *configapi.NewTypedValueString(ValueLeaf1AMyValue)}

	configValues[8] = &configapi.PathValue{Path: Test1Cont1AList2a1t, Value: *configapi.NewTypedValueUint(ValueList2a1PwrT, 16)}
	configValues[9] = &configapi.PathValue{Path: Test1Cont1AList2a1min, Value: *configapi.NewTypedValueUint(ValueList2a1Rmin, 8)}
	configValues[10] = &configapi.PathValue{Path: Test1Cont1AList2a1max, Value: *configapi.NewTypedValueUint(ValueList2a1Rmax, 8)}
	configValues[11] = &configapi.PathValue{Path: Test1Cont1AList2a2t, Value: *configapi.NewTypedValueUint(ValueList2a2PwrT, 16)}
	configValues[12] = &configapi.PathValue{Path: Test1Cont1AList2a2min, Value: *configapi.NewTypedValueUint(ValueList2a2Rmin, 8)}
	configValues[13] = &configapi.PathValue{Path: Test1Cont1AList2a2max, Value: *configapi.NewTypedValueUint(ValueList2a2Rmax, 8)}
	configValues[14] = &configapi.PathValue{Path: Test1Leaftoplevel, Value: *configapi.NewTypedValueString(ValueLeaftopWxy1234)}

	// Adding back in a double key list - the key must be given explicitly when it's not a string
	configValues[15] = &configapi.PathValue{Path: Test1list5One1K1, Value: *configapi.NewTypedValueString("one")}
	configValues[16] = &configapi.PathValue{Path: Test1list5One1K2, Value: *configapi.NewTypedValueUint(1, 8)}
	configValues[17] = &configapi.PathValue{Path: Test1list5One1L5a, Value: *configapi.NewTypedValueString(ValueList5aOne1L5a)}

	configValues[18] = &configapi.PathValue{Path: Test1list5Two1K1, Value: *configapi.NewTypedValueString("two")}
	configValues[19] = &configapi.PathValue{Path: Test1list5Two1K2, Value: *configapi.NewTypedValueUint(1, 8)}
	configValues[20] = &configapi.PathValue{Path: Test1list5Two1L5a, Value: *configapi.NewTypedValueString(ValueList5aTwo1L5a)}

	configValues[21] = &configapi.PathValue{Path: Test1list5Two2K1, Value: *configapi.NewTypedValueString("two")}
	configValues[22] = &configapi.PathValue{Path: Test1list5Two2K2, Value: *configapi.NewTypedValueUint(2, 8)}
	configValues[23] = &configapi.PathValue{Path: Test1list5Two2L5a, Value: *configapi.NewTypedValueString(ValueList5aTwo2L5a)}

	configValues[24] = &configapi.PathValue{Path: Test1list5Two3K1, Value: *configapi.NewTypedValueString("two")}
	configValues[25] = &configapi.PathValue{Path: Test1list5Two3K2, Value: *configapi.NewTypedValueUint(3, 8)}
	configValues[26] = &configapi.PathValue{Path: Test1list5Two3L5a, Value: *configapi.NewTypedValueString(ValueList5aTwo3L5a)}

	configValues[27] = &configapi.PathValue{Path: Test1list5Five5K1, Value: *configapi.NewTypedValueString("five")}
	configValues[28] = &configapi.PathValue{Path: Test1list5Five5K2, Value: *configapi.NewTypedValueUint(5, 8)}
	configValues[29] = &configapi.PathValue{Path: Test1list5Five5L5a, Value: *configapi.NewTypedValueString(ValueList5aFive5L5a)}
}

func Test_BuildTree(t *testing.T) {
	setUpTree()

	jsonTree, err := BuildTree(configValues, true)

	assert.NilError(t, err)
	assert.Equal(t, testJSON1, string(jsonTree))

	//Verify we can go back to objects
	model := new(testdevice_1_0_0.Device)
	err = testdevice_1_0_0.Unmarshal(jsonTree, model)
	assert.NilError(t, err, "unexpected error unmarshalling JSON to modelplugin model")
	assert.Equal(t, ValueLeaf2A12, int(*model.Cont1A.Cont2A.Leaf2A))
	assert.Equal(t, ValueLeaf2B114, *model.Cont1A.Cont2A.Leaf2B)
	assert.Equal(t, ValueLeaf2CMyValue, *model.Cont1A.Cont2A.Leaf2C)
	assert.Equal(t, float64(ValueLeaf2D114)/1e5, *model.Cont1A.Cont2A.Leaf2D)
	assert.DeepEqual(t, []int16{ValueLeaf2EInt1, ValueLeaf2EInt2, ValueLeaf2EInt3, ValueLeaf2EInt4, ValueLeaf2EInt5},
		model.Cont1A.Cont2A.Leaf2E)

	assert.Equal(t, ValueList2A2F, string(model.Cont1A.Cont2A.Leaf2F))
	assert.Equal(t, ValueList2A2G, *model.Cont1A.Cont2A.Leaf2G)

	assert.Equal(t, ValueLeaf1AMyValue, *model.Cont1A.Leaf1A)

	assert.Equal(t, 2, len(model.Cont1A.List2A))
	list2AFirst, ok := model.Cont1A.List2A["First"]
	assert.Assert(t, ok)
	assert.Equal(t, "First", *list2AFirst.Name)
	assert.Equal(t, 5, int(*list2AFirst.TxPower))
	assert.Equal(t, 6, int(*list2AFirst.RangeMin))

	list2ASecond, ok := model.Cont1A.List2A["Second"]
	assert.Assert(t, ok)
	assert.Equal(t, "Second", *list2ASecond.Name)
	assert.Equal(t, 7, int(*list2ASecond.TxPower))
	assert.Equal(t, 8, int(*list2ASecond.RangeMin))

	assert.Equal(t, ValueLeaftopWxy1234, *model.LeafAtTopLevel)

	assert.Equal(t, 5, len(model.Cont1A.List5))
	for list5Key, list5Item := range model.Cont1A.List5 {
		keyAsStr := fmt.Sprintf("%s-%d", list5Key.Key1, list5Key.Key2)
		assert.Equal(t, fmt.Sprintf("5A %s", keyAsStr), *list5Item.Leaf5A)
		switch keyAsStr {
		case "one-1", "two-1", "two-2", "two-3", "five-5":
		default:
			t.Fatalf("unexpected List5 entry '%s'", keyAsStr)
		}
	}
}

func Test_PrunedPaths(t *testing.T) {
	setUpTree()

	// Nothing is pruned, but list is ordered
	pruned := PrunePathValues(configValues, true)
	assert.Equal(t, len(pruned), len(configValues)) // nothing to prune
	prunedMap := PrunePathMap(ListToMap(configValues), true)
	assert.Equal(t, len(prunedMap), len(pruned))

	configValues = append(configValues, &configapi.PathValue{Path: Test1Cont1AList2a1, Deleted: true})
	pruned = PrunePathValues(configValues, true)
	assert.Equal(t, len(pruned), len(configValues)-3) // three sub-paths (tx-power, range-mix, range-max)
	prunedMap = PrunePathMap(ListToMap(configValues), true)
	assert.Equal(t, len(prunedMap), len(pruned))

	pruned = PrunePathValues(configValues, false)
	assert.Equal(t, len(pruned), len(configValues)-4) // path + three sub-paths
	prunedMap = PrunePathMap(ListToMap(configValues), false)
	assert.Equal(t, len(prunedMap), len(pruned))

	configValues = append(configValues, &configapi.PathValue{Path: Test1Cont1AList2a, Deleted: true})
	pruned = PrunePathValues(configValues, true)
	assert.Equal(t, len(pruned), len(configValues)-7) // path + 1 explicitly deleted sub-path + 6 sub-paths
	prunedMap = PrunePathMap(ListToMap(configValues), true)
	assert.Equal(t, len(prunedMap), len(pruned))

	// Test double keyed list
	configValues = append(configValues, &configapi.PathValue{Path: Test1list5Two1, Deleted: true})
	pruned = PrunePathValues(configValues, true)
	assert.Equal(t, len(pruned), len(configValues)-10) // 1 path + 2 explicitly deleted sub-path + 9 sub-paths
	prunedMap = PrunePathMap(ListToMap(configValues), true)
	assert.Equal(t, len(prunedMap), len(pruned))
}

func ListToMap(paths []*configapi.PathValue) map[string]*configapi.PathValue {
	pm := make(map[string]*configapi.PathValue, len(paths))
	for _, pv := range paths {
		pm[pv.Path] = pv
	}
	return pm
}

// PrintPaths prints out the give list of paths for convenience
func PrintPaths(paths []*configapi.PathValue) {
	for i, pv := range paths {
		fmt.Printf("%4d: %s: %5t: %+v\n", i, pv.Path, pv.Deleted, pv.Value)
	}
}

// PrintPathMap prints out the give list of paths for convenience
func PrintPathMap(paths map[string]*configapi.PathValue) {
	for p, pv := range paths {
		fmt.Printf("%s: %s: %5t: %+v\n", p, pv.Path, pv.Deleted, pv.Value)
	}
}
