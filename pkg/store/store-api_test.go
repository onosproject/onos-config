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
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"gotest.tools/assert"
	log "k8s.io/klog"
	"os"
	"strconv"
	"testing"
)

const (
	Test1Cont1A                  = "/cont1a"
	Test1Cont1ACont2A            = "/cont1a/cont2a"
	Test1Cont1ACont2ALeaf2A      = "/cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B      = "/cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C      = "/cont1a/cont2a/leaf2c"
	Test1Cont1ACont2ALeaf2D      = "/cont1a/cont2a/leaf2d"
	Test1Cont1ACont2ALeaf2E      = "/cont1a/cont2a/leaf2e"
	Test1Cont1ACont2ALeaf2F      = "/cont1a/cont2a/leaf2f"
	Test1Cont1ACont2ALeaf2G      = "/cont1a/cont2a/leaf2g"
	Test1Cont1ALeaf1A            = "/cont1a/leaf1a"
	Test1Cont1AList2ATxout1      = "/cont1a/list2a[name=txout1]"
	Test1Cont1AList2ATxout1Txpwr = "/cont1a/list2a[name=txout1]/tx-power"
	Test1Cont1AList2ATxout2      = "/cont1a/list2a[name=txout2]"
	Test1Cont1AList2ATxout2Txpwr = "/cont1a/list2a[name=txout2]/tx-power"
	Test1Cont1AList2ATxout3      = "/cont1a/list2a[name=txout3]"
	Test1Cont1AList2ATxout3Txpwr = "/cont1a/list2a[name=txout3]/tx-power"
	Test1Leaftoplevel            = "/leafAtTopLevel"
)

const (
	ValueLeaf2A13       = 13
	ValueLeaf2B159      = 1.579   // AAAAAPohCUA=
	ValueLeaf2B314      = 3.14159 // AAAAgJVD+T8=
	ValueLeaf2CAbc      = "abc"
	ValueLeaf2CDef      = "def"
	ValueLeaf2CGhi      = "ghi"
	ValueLeaf1AAbcdef   = "abcdef"
	ValueTxout1Txpwr8   = 8
	ValueTxout2Txpwr10  = 10
	ValueTxout3Txpwr16  = 16
	ValueLeaftopWxy1234 = "WXY-1234"
)

var c1ID, c2ID, c3ID change.ID

func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	os.Exit(m.Run())
}

func setUp() (device1V, device2V *Configuration, changeStore map[string]*change.Change) {
	var err error

	var (
		change1, change2, change3, change4 *change.Change
	)

	config1Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1A, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2A, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value03, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2A, devicechangetypes.NewTypedValueUint64(ValueLeaf2A13), false)
	config1Value04, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2B, devicechangetypes.NewTypedValueFloat(ValueLeaf2B159), false)
	config1Value05, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2C, devicechangetypes.NewTypedValueString(ValueLeaf2CAbc), false)
	config1Value06, _ := devicechangetypes.NewChangeValue(Test1Cont1ALeaf1A, devicechangetypes.NewTypedValueString(ValueLeaf1AAbcdef), false)
	config1Value07, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout1, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value08, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout1Txpwr, devicechangetypes.NewTypedValueUint64(ValueTxout1Txpwr8), false)
	config1Value09, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout2, devicechangetypes.NewTypedValueEmpty(), false)
	config1Value10, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout2Txpwr, devicechangetypes.NewTypedValueUint64(ValueTxout2Txpwr10), false)
	config1Value11, _ := devicechangetypes.NewChangeValue(Test1Leaftoplevel, devicechangetypes.NewTypedValueString(ValueLeaftopWxy1234), false)
	change1, err = change.NewChange([]*devicechangetypes.ChangeValue{
		config1Value01, config1Value02, config1Value03, config1Value04, config1Value05,
		config1Value06, config1Value07, config1Value08, config1Value09, config1Value10,
		config1Value11,
	}, "Original Config for test switch")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	config2Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2B, devicechangetypes.NewTypedValueFloat(ValueLeaf2B314), false)
	config2Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout3, devicechangetypes.NewTypedValueEmpty(), false)
	config2Value03, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout3Txpwr, devicechangetypes.NewTypedValueUint64(ValueTxout3Txpwr16), false)
	change2, err = change.NewChange([]*devicechangetypes.ChangeValue{
		config2Value01, config2Value02, config2Value03,
	}, "Trim power level")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	config3Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2C, devicechangetypes.NewTypedValueString(ValueLeaf2CDef), false)
	config3Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout2, devicechangetypes.NewTypedValueEmpty(), true)
	change3, err = change.NewChange([]*devicechangetypes.ChangeValue{
		config3Value01, config3Value02,
	}, "Remove txout 2")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	changeStore = make(map[string]*change.Change)
	changeStore[B64(change1.ID)] = change1
	changeStore[B64(change2.ID)] = change2
	changeStore[B64(change3.ID)] = change3

	c1ID = change1.ID
	c2ID = change2.ID
	c3ID = change2.ID

	device1V, err = NewConfiguration("Device1", "1.0.0", "TestDevice",
		[]change.ID{change1.ID, change2.ID, change3.ID})
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	config4Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2C, devicechangetypes.NewTypedValueString(ValueLeaf2CGhi), false)
	config4Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout1, devicechangetypes.NewTypedValueEmpty(), true)
	change4, err = change.NewChange([]*devicechangetypes.ChangeValue{
		config4Value01, config4Value02,
	}, "Remove txout 1")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
	changeStore[B64(change4.ID)] = change4

	device2V, err = NewConfiguration("Device2", "10.0.100", "TestDevice",
		[]change.ID{change1.ID, change2.ID, change4.ID})
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	return device1V, device2V, changeStore
}

func TestCreateConfiguration_badname(t *testing.T) {
	_, err :=
		NewConfiguration("", "1.0.0", "TestDevice",
			[]change.ID{})
	assert.ErrorContains(t, err, "deviceName, version and deviceType must have values", "Empty")

	_, err =
		NewConfiguration("abc", "1.0.0", "TestDevice",
			[]change.ID{})
	assert.ErrorContains(t, err, "name abc does not match pattern", "Too short")

	_, err =
		NewConfiguration("abc???", "1.0.0", "TestDevice",
			[]change.ID{})
	assert.ErrorContains(t, err, "name abc??? does not match pattern", "Illegal char")

	_, err =
		NewConfiguration("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO",
			"1.0.0", "TestDevice",
			[]change.ID{})
	assert.ErrorContains(t, err, "name abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNO does not match pattern", "Too long")
}

func TestCreateConfiguration_badversion(t *testing.T) {
	_, err :=
		NewConfiguration("localhost-1", "1.234567890", "TestDevice",
			[]change.ID{})
	assert.ErrorContains(t, err, "version 1.234567890 does not match pattern", "Too long")

	_, err =
		NewConfiguration("localhost-1", "a", "TestDevice",
			[]change.ID{})
	assert.ErrorContains(t, err, "version a does not match pattern", "has letter")

	_, err =
		NewConfiguration("localhost-1", "1:0:0", "TestDevice",
			[]change.ID{})
	assert.ErrorContains(t, err, "version 1:0:0 does not match pattern", "Illegal char")
}

func TestCreateConfiguration_badtype(t *testing.T) {
	_, err :=
		NewConfiguration("localhost-1", "1.0.0", "TestD&eviceType",
			[]change.ID{})
	assert.ErrorContains(t, err, "does not match pattern", "bad char")
}

// Test_createnetStore tests that a valid network config name is accepted
// Note: the testing against duplicate names is done in northbound/set_test.go
func Test_createnetStore(t *testing.T) {
	nwStore, err := NewNetworkConfiguration("testnwstore", "onos", make(map[ConfigName]change.ID))
	assert.NilError(t, err, "Unexpected error")

	assert.Equal(t, nwStore.User, "onos")
	assert.Equal(t, nwStore.Name, "testnwstore")
	assert.Equal(t, len(nwStore.ConfigurationChanges), 0)
}

func Test_createnetStore_badname(t *testing.T) {
	nwStore, err := NewNetworkConfiguration("???????", "onos", make(map[ConfigName]change.ID))
	assert.ErrorContains(t, err, "Error in name ???????")
	assert.Check(t, nwStore == nil, "unexpected result", nwStore)
}

// Test_createnetStore tests that a valid network config name is accepted
// Note: the testing of name with no extension 100 is done in northbound/set_test.go
func Test_createnetStore_noname(t *testing.T) {
	var noname string
	_, err := NewNetworkConfiguration(noname, "onos", make(map[ConfigName]change.ID))
	assert.ErrorContains(t, err, "Empty name not allowed")
}

func BenchmarkCreateChangeValue(b *testing.B) {

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/test-%s", strconv.Itoa(b.N))
		cv, _ := devicechangetypes.NewChangeValue(path, devicechangetypes.NewTypedValueUint64(uint(i)), false)
		err := devicechangetypes.IsPathValid(cv.Path)
		assert.NilError(b, err, "path not valid %s", err)

	}
}

func BenchmarkCreateChange(b *testing.B) {

	changeValues := []*devicechangetypes.ChangeValue{}
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/test%d", i)
		cv, _ := devicechangetypes.NewChangeValue(path, devicechangetypes.NewTypedValueUint64(uint(i)), false)
		changeValues = append(changeValues, cv)
	}

	newChange, _ := change.NewChange(changeValues, "Benchmarked Change")

	err := newChange.IsValid()
	assert.NilError(b, err, "Invalid change %s", err)
}
