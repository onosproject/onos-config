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
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	types "github.com/onosproject/onos-config/pkg/types/change/device"
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

var Config1Paths = [11]string{
	Test1Cont1A, // 0
	Test1Cont1ACont2A,
	Test1Cont1ACont2ALeaf2A,
	Test1Cont1ACont2ALeaf2B, // 3
	Test1Cont1ACont2ALeaf2C,
	Test1Cont1ALeaf1A, // 5
	Test1Cont1AList2ATxout1,
	Test1Cont1AList2ATxout1Txpwr,
	Test1Cont1AList2ATxout3, //10
	Test1Cont1AList2ATxout3Txpwr,
	Test1Leaftoplevel,
}

var Config1Values = [11][]byte{
	make([]byte, 0), // 0
	make([]byte, 0),
	{13, 0, 0, 0, 0, 0, 0, 0},    // ValueLeaf2A13
	{0, 0, 0, 0, 250, 33, 9, 64}, // ValueLeaf2B314 3
	{100, 101, 102},              // ValueLeaf2CDef
	{97, 98, 99, 100, 101, 102},  // ValueLeaf1AAbcdef 5
	make([]byte, 0),
	{8, 0, 0, 0, 0, 0, 0, 0},         // ValueTxout1Txpwr8
	make([]byte, 0),                  // 10
	{16, 0, 0, 0, 0, 0, 0, 0},        // ValueTxout3Txpwr16
	{87, 88, 89, 45, 49, 50, 51, 52}, // ValueLeaftopWxy1234
}

var Config1Types = [11]types.ValueType{
	types.ValueType_EMPTY, // 0
	types.ValueType_EMPTY, // 0
	types.ValueType_UINT,
	types.ValueType_FLOAT, // 3
	types.ValueType_STRING,
	types.ValueType_STRING, // 5
	types.ValueType_EMPTY,
	types.ValueType_UINT,
	types.ValueType_EMPTY,
	types.ValueType_UINT,
	types.ValueType_STRING, // 10
}

var Config1PreviousPaths = [13]string{
	Test1Cont1A, // 0
	Test1Cont1ACont2A,
	Test1Cont1ACont2ALeaf2A,
	Test1Cont1ACont2ALeaf2B, // 3
	Test1Cont1ACont2ALeaf2C,
	Test1Cont1ALeaf1A, // 5
	Test1Cont1AList2ATxout1,
	Test1Cont1AList2ATxout1Txpwr,
	Test1Cont1AList2ATxout2,
	Test1Cont1AList2ATxout2Txpwr,
	Test1Cont1AList2ATxout3, //10
	Test1Cont1AList2ATxout3Txpwr,
	Test1Leaftoplevel,
}

var Config1PreviousValues = [13][]byte{
	{}, // 0
	{},
	{13, 0, 0, 0, 0, 0, 0, 0},    // ValueLeaf2A13
	{0, 0, 0, 0, 250, 33, 9, 64}, // ValueLeaf2B314 3
	{97, 98, 99},                 // ValueLeaf2CAbc
	{97, 98, 99, 100, 101, 102},  // ValueLeaf1AAbcdef 5
	{},
	{8, 0, 0, 0, 0, 0, 0, 0}, // ValueTxout1Txpwr8
	{},
	{10, 0, 0, 0, 0, 0, 0, 0},        // ValueTxout2Txpwr10
	{},                               // 10
	{16, 0, 0, 0, 0, 0, 0, 0},        // ValueTxout3Txpwr16,
	{87, 88, 89, 45, 49, 50, 51, 52}, // ValueLeaftopWxy1234,
}

var Config1PreviousTypes = [13]types.ValueType{
	types.ValueType_EMPTY, // 0
	types.ValueType_EMPTY, // 0
	types.ValueType_UINT,
	types.ValueType_FLOAT, // 3
	types.ValueType_STRING,
	types.ValueType_STRING, // 5
	types.ValueType_EMPTY,
	types.ValueType_UINT,
	types.ValueType_EMPTY,
	types.ValueType_UINT,
	types.ValueType_EMPTY, // 10
	types.ValueType_UINT,
	types.ValueType_STRING,
}

var Config1FirstPaths = [11]string{
	Test1Cont1A, // 0
	Test1Cont1ACont2A,
	Test1Cont1ACont2ALeaf2A,
	Test1Cont1ACont2ALeaf2B, // 3
	Test1Cont1ACont2ALeaf2C,
	Test1Cont1ALeaf1A, // 5
	Test1Cont1AList2ATxout1,
	Test1Cont1AList2ATxout1Txpwr,
	Test1Cont1AList2ATxout2,
	Test1Cont1AList2ATxout2Txpwr,
	Test1Leaftoplevel, //10
}

var Config1FirstValues = [11][]byte{
	{}, // 0
	{},
	{13, 0, 0, 0, 0, 0, 0, 0},        // ValueLeaf2A13
	{0, 0, 0, 128, 149, 67, 249, 63}, // ValueLeaf2B159 3
	{97, 98, 99},                     // ValueLeaf2CAbc
	{97, 98, 99, 100, 101, 102},      // ValueLeaf1AAbcdef 5
	{},
	{8, 0, 0, 0, 0, 0, 0, 0}, // ValueTxout1Txpwr8
	{},
	{10, 0, 0, 0, 0, 0, 0, 0},        // ValueTxout2Txpwr10
	{87, 88, 89, 45, 49, 50, 51, 52}, //ValueLeaftopWxy1234, 10
}

var Config1FirstTypes = [11]types.ValueType{
	types.ValueType_EMPTY, // 0
	types.ValueType_EMPTY, // 0
	types.ValueType_UINT,
	types.ValueType_FLOAT, // 3
	types.ValueType_STRING,
	types.ValueType_STRING, // 5
	types.ValueType_EMPTY,
	types.ValueType_UINT,
	types.ValueType_EMPTY,
	types.ValueType_UINT,
	types.ValueType_STRING, // 10
}

var Config2Paths = [11]string{
	Test1Cont1A, // 0
	Test1Cont1ACont2A,
	Test1Cont1ACont2ALeaf2A,
	Test1Cont1ACont2ALeaf2B, // 3
	Test1Cont1ACont2ALeaf2C,
	Test1Cont1ALeaf1A, // 5
	Test1Cont1AList2ATxout2,
	Test1Cont1AList2ATxout2Txpwr,
	Test1Cont1AList2ATxout3, //10
	Test1Cont1AList2ATxout3Txpwr,
	Test1Leaftoplevel,
}

var Config2Values = [11][]byte{
	{}, // 0
	{},
	{13, 0, 0, 0, 0, 0, 0, 0},    // ValueLeaf2A13
	{0, 0, 0, 0, 250, 33, 9, 64}, // ValueLeaf2B314 3
	{103, 104, 105},              // ValueLeaf2CGhi
	{97, 98, 99, 100, 101, 102},  // ValueLeaf1AAbcdef 5
	{},
	{10, 0, 0, 0, 0, 0, 0, 0}, // ValueTxout1Txpwr8
	{},
	{16, 0, 0, 0, 0, 0, 0, 0},        // ValueTxout2Txpwr10
	{87, 88, 89, 45, 49, 50, 51, 52}, //ValueLeaftopWxy1234, 10
}

var Config2Types = [11]types.ValueType{
	types.ValueType_EMPTY, // 0
	types.ValueType_EMPTY, // 0
	types.ValueType_UINT,
	types.ValueType_FLOAT, // 3
	types.ValueType_STRING,
	types.ValueType_STRING, // 5
	types.ValueType_EMPTY,
	types.ValueType_UINT,
	types.ValueType_EMPTY,
	types.ValueType_UINT,
	types.ValueType_STRING, // 10
}

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

	config1Value01, _ := types.NewChangeValue(Test1Cont1A, types.NewTypedValueEmpty(), false)
	config1Value02, _ := types.NewChangeValue(Test1Cont1ACont2A, types.NewTypedValueEmpty(), false)
	config1Value03, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2A, types.NewTypedValueUint64(ValueLeaf2A13), false)
	config1Value04, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2B, types.NewTypedValueFloat(ValueLeaf2B159), false)
	config1Value05, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2C, types.NewTypedValueString(ValueLeaf2CAbc), false)
	config1Value06, _ := types.NewChangeValue(Test1Cont1ALeaf1A, types.NewTypedValueString(ValueLeaf1AAbcdef), false)
	config1Value07, _ := types.NewChangeValue(Test1Cont1AList2ATxout1, types.NewTypedValueEmpty(), false)
	config1Value08, _ := types.NewChangeValue(Test1Cont1AList2ATxout1Txpwr, types.NewTypedValueUint64(ValueTxout1Txpwr8), false)
	config1Value09, _ := types.NewChangeValue(Test1Cont1AList2ATxout2, types.NewTypedValueEmpty(), false)
	config1Value10, _ := types.NewChangeValue(Test1Cont1AList2ATxout2Txpwr, types.NewTypedValueUint64(ValueTxout2Txpwr10), false)
	config1Value11, _ := types.NewChangeValue(Test1Leaftoplevel, types.NewTypedValueString(ValueLeaftopWxy1234), false)
	change1, err = change.NewChange([]*types.ChangeValue{
		config1Value01, config1Value02, config1Value03, config1Value04, config1Value05,
		config1Value06, config1Value07, config1Value08, config1Value09, config1Value10,
		config1Value11,
	}, "Original Config for test switch")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	config2Value01, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2B, types.NewTypedValueFloat(ValueLeaf2B314), false)
	config2Value02, _ := types.NewChangeValue(Test1Cont1AList2ATxout3, types.NewTypedValueEmpty(), false)
	config2Value03, _ := types.NewChangeValue(Test1Cont1AList2ATxout3Txpwr, types.NewTypedValueUint64(ValueTxout3Txpwr16), false)
	change2, err = change.NewChange([]*types.ChangeValue{
		config2Value01, config2Value02, config2Value03,
	}, "Trim power level")
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}

	config3Value01, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2C, types.NewTypedValueString(ValueLeaf2CDef), false)
	config3Value02, _ := types.NewChangeValue(Test1Cont1AList2ATxout2, types.NewTypedValueEmpty(), true)
	change3, err = change.NewChange([]*types.ChangeValue{
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

	config4Value01, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2C, types.NewTypedValueString(ValueLeaf2CGhi), false)
	config4Value02, _ := types.NewChangeValue(Test1Cont1AList2ATxout1, types.NewTypedValueEmpty(), true)
	change4, err = change.NewChange([]*types.ChangeValue{
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

func Test_writeOutChangeFile(t *testing.T) {
	_, _, changeStore := setUp()
	if _, err := os.Stat("testout"); os.IsNotExist(err) {
		_ = os.Mkdir("testout", os.ModePerm)
	}
	changeStoreFile, err := os.Create("testout/changeStore-sample.json")
	if err != nil {
		t.Errorf("%s", err)
	}

	jsonEncoder := json.NewEncoder(changeStoreFile)
	var csf = ChangeStore{Version: StoreVersion,
		Storetype: StoreTypeChange, Store: changeStore}
	err = jsonEncoder.Encode(csf)
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
	defer changeStoreFile.Close()
}

func Test_loadChangeStoreFile(t *testing.T) {
	changeStore, err := LoadChangeStore("testout/changeStore-sample.json")
	assert.NilError(t, err, "Unexpected error when loading Change Store from file %s", err)
	assert.Equal(t, changeStore.Version, StoreVersion)
}

func Test_loadChangeStoreFileError(t *testing.T) {
	changeStore, err := LoadChangeStore("nonexistent.json")
	assert.Assert(t, err != nil, "Expected an error when loading Change Store from invalid file")
	assert.Equal(t, changeStore.Version, "")
}

func Test_loadConfigStoreFileBadVersion(t *testing.T) {
	_, err := LoadConfigStore("testdata/configStore-badVersion.json")
	assert.ErrorContains(t, err, "Store version invalid")
}

func Test_loadConfigStoreFileBadType(t *testing.T) {
	_, err := LoadConfigStore("testdata/configStore-badType.json")
	assert.ErrorContains(t, err, "Store type invalid")
}

func Test_loadChangeStoreFileBadVersion(t *testing.T) {
	_, err := LoadChangeStore("testdata/changeStore-badVersion.json")
	assert.ErrorContains(t, err, "Store version invalid")
}

func Test_loadChangeStoreFileBadType(t *testing.T) {
	_, err := LoadChangeStore("testdata/changeStore-badType.json")
	assert.ErrorContains(t, err, "Store type invalid")
}

func Test_loadNetworkStoreFileError(t *testing.T) {
	networkStore, err := LoadNetworkStore("nonexistent.json")
	assert.Assert(t, err != nil, "Expected an error when loading Change Store from invalid file")
	assert.Assert(t, networkStore == nil, "")
}

func Test_loadNetworkStoreFileBadVersion(t *testing.T) {
	_, err := LoadNetworkStore("testdata/networkStore-badVersion.json")
	assert.ErrorContains(t, err, "Store version invalid")
}

func Test_loadNetworkStoreFileBadType(t *testing.T) {
	_, err := LoadNetworkStore("testdata/networkStore-badType.json")
	assert.ErrorContains(t, err, "Store type invalid")
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

func Test_writeOutConfigFile(t *testing.T) {
	device1V, device2V, _ := setUp()
	configurationStore := make(map[ConfigName]Configuration)
	configurationStore[device1V.Name] = *device1V
	configurationStore[device2V.Name] = *device2V

	configStoreFile, _ := os.Create("testout/configStore-sample.json")
	jsonEncoder := json.NewEncoder(configStoreFile)
	err := jsonEncoder.Encode(ConfigurationStore{Version: StoreVersion,
		Storetype: StoreTypeConfig, Store: configurationStore})
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
	defer configStoreFile.Close()
}

func Test_loadConfigStoreFile(t *testing.T) {
	configStore, err := LoadConfigStore("testout/configStore-sample.json")

	assert.NilError(t, err, "Unexpected error when loading Config Store from file %s", err)
	assert.Equal(t, configStore.Version, StoreVersion)
}

func Test_loadConfigStoreFileError(t *testing.T) {
	configStore, err := LoadConfigStore("nonexistent.json")
	assert.Assert(t, err != nil, "Expected an error when loading Config Store from invalid file")
	assert.Equal(t, configStore.Version, "")
}

func Test_writeOutNetworkFile(t *testing.T) {
	networkStore := make([]NetworkConfiguration, 0)
	ccs := make(map[ConfigName]change.ID)
	ccs["Device2VersionMain"] = []byte("DCuMG07l01g2BvMdEta+7DyxMxk=")
	ccs["Device2VersionMain"] = []byte("LsDuwm2XJjdOq+u9QEcUJo/HxaM=")
	nw1, err := NewNetworkConfiguration("testChange", "nw1", ccs)
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
	networkStore = append(networkStore, *nw1)

	networkStoreFile, _ := os.Create("testout/networkStore-sample.json")
	jsonEncoder := json.NewEncoder(networkStoreFile)
	err = jsonEncoder.Encode(NetworkStore{Version: StoreVersion,
		Storetype: StoreTypeNetwork, Store: networkStore})
	if err != nil {
		log.Error(err)
		os.Exit(-1)
	}
	defer networkStoreFile.Close()
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

func Test_loadNetworkStoreFile(t *testing.T) {
	networkStore, err := LoadNetworkStore("testout/networkStore-sample.json")
	assert.NilError(t, err, "Unexpected error when loading Network Store from file %s", err)
	assert.Equal(t, networkStore.Version, StoreVersion)
}

func BenchmarkCreateChangeValue(b *testing.B) {

	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/test-%s", strconv.Itoa(b.N))
		cv, _ := types.NewChangeValue(path, types.NewTypedValueUint64(uint(i)), false)
		err := types.IsPathValid(cv.Path)
		assert.NilError(b, err, "path not valid %s", err)

	}
}

func BenchmarkCreateChange(b *testing.B) {

	changeValues := []*types.ChangeValue{}
	for i := 0; i < b.N; i++ {
		path := fmt.Sprintf("/test%d", i)
		cv, _ := types.NewChangeValue(path, types.NewTypedValueUint64(uint(i)), false)
		changeValues = append(changeValues, cv)
	}

	newChange, _ := change.NewChange(changeValues, "Benchmarked Change")

	err := newChange.IsValid()
	assert.NilError(b, err, "Invalid change %s", err)
}
