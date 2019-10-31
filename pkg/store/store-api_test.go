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

func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	os.Exit(m.Run())
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
