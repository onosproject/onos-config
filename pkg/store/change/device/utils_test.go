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

package device

import (
	"encoding/base64"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/store/change"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
	"os"
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

var B64 = base64.StdEncoding.EncodeToString


func setUp(t *testing.T) (*types.DeviceChange, *types.DeviceChange, Store) {
	log.SetOutput(os.Stdout)
	ctrl := gomock.NewController(t)
	mockChangeStore := mockstore.NewMockDeviceChangesStore(ctrl)
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

	device1 := &devicepb.Device{
		ID:          "Device1",
		Revision:    0,
		Address:     "",
		Target:      "",
		Version:     "1.0.0",
		Timeout:     nil,
		Credentials: devicepb.Credentials{},
		TLS:         devicepb.TlsConfig{},
		Type:        "TestDevice",
		Role:        "",
		Attributes:  nil,
		Protocols:   nil,
	}
	change1 := types.Change{
		Values: []*types.ChangeValue{
			config1Value01, config1Value02, config1Value03,
			config1Value04, config1Value05, config1Value06,
			config1Value07, config1Value08, config1Value09,
			config1Value10, config1Value11,
		},
		DeviceID: device1.ID,
	}
	deviceChange1 := &devicechange.DeviceChange{
		Change: &change1,
		ID:     "Change1",
	}

	config2Value01, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2B, types.NewTypedValueFloat(ValueLeaf2B314), false)
	config2Value02, _ := types.NewChangeValue(Test1Cont1AList2ATxout3, types.NewTypedValueEmpty(), false)
	config2Value03, _ := types.NewChangeValue(Test1Cont1AList2ATxout3Txpwr, types.NewTypedValueUint64(ValueTxout3Txpwr16), false)

	change2 := types.Change{
		Values: []*types.ChangeValue{
			config2Value01, config2Value02, config2Value03,
		},
		DeviceID: device1.ID,
	}
	deviceChange2 := &devicechange.DeviceChange{
		Change: &change2,
		ID:     "Change2",
	}

	config3Value01, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2C, types.NewTypedValueString(ValueLeaf2CDef), false)
	config3Value02, _ := types.NewChangeValue(Test1Cont1AList2ATxout2, types.NewTypedValueEmpty(), true)

	change3 := types.Change{
		Values: []*types.ChangeValue{
			config3Value01, config3Value02,
		},
		DeviceID: device1.ID,
	}
	deviceChange3 := &devicechange.DeviceChange{
		Change: &change3,
		ID:     "Change3",
	}

	config4Value01, _ := types.NewChangeValue(Test1Cont1ACont2ALeaf2C, types.NewTypedValueString(ValueLeaf2CGhi), false)
	config4Value02, _ := types.NewChangeValue(Test1Cont1AList2ATxout1, types.NewTypedValueEmpty(), true)
	change4 := types.Change{
		Values: []*types.ChangeValue{
			config4Value01, config4Value02,
		},
		DeviceID: device1.ID,
	}
	deviceChange4 := &devicechange.DeviceChange{
		Change: &change4,
		ID:     "Change4",
	}

	//c1ID = change1.ID
	//c2ID = change2.ID
	//c3ID = change2.ID
	//
	//device1V, err = NewConfiguration("Device1", "1.0.0", "TestDevice",
	//	[]change.ID{change1.ID, change2.ID, change3.ID})
	//if err != nil {
	//	log.Error(err)
	//	os.Exit(-1)
	//}
	//
	//device2V, err = NewConfiguration("Device2", "10.0.100", "TestDevice",
	//	[]change.ID{change1.ID, change2.ID, change4.ID})
	//if err != nil {
	//	log.Error(err)
	//	os.Exit(-1)
	//}

	mockChangeStore.EXPECT().Get(deviceChange1.ID).Return(deviceChange1, nil).AnyTimes()
	mockChangeStore.EXPECT().Get(deviceChange2.ID).Return(deviceChange2, nil).AnyTimes()
	mockChangeStore.EXPECT().Get(deviceChange3.ID).Return(deviceChange3, nil).AnyTimes()
	mockChangeStore.EXPECT().Get(deviceChange4.ID).Return(deviceChange4, nil).AnyTimes()

	return deviceChange1, deviceChange4, mockChangeStore
}

func checkPathvalue(t *testing.T, config []*types.PathValue, index int,
	expPaths []string, expValues [][]byte, expTypes []types.ValueType) {

	// Check that they are kept in a consistent order
	if config[index].Path != expPaths[index] {
		t.Errorf("Unexpected change %d Exp: %s, Got %s", index,
			expPaths[index], config[index].Path)
	}

	if B64(config[index].GetValue().GetBytes()) != B64(expValues[index]) {
		t.Errorf("Unexpected change value %d Exp: %v, Got %v", index,
			expValues[index], config[index].GetValue().GetBytes())
	}

	if config[index].GetValue().GetType() != expTypes[index] {
		t.Errorf("Unexpected change type %d Exp: %d, Got %d", index,
			expTypes[index], config[index].GetValue().GetType())
	}
}

func Test_device1_first_version(t *testing.T) {
	device1V, _, _ := setUp(t)

	log.Info("Device ", device1V.Change.DeviceID, " (latest) Changes:")
	for idx, cid := range device1V.Change.Values {
		log.Infof("%d: %s\n", idx, cid)
	}

	//assert.Equal(t, device1V.Name, ConfigName("Device1-1.0.0"))
	//
	////Check the value of leaf2c before
	//change1, ok := changeStore[B64(c1ID)]
	//assert.Assert(t, ok)
	//assert.Equal(t, len(change1.Config), 11)
	//leaf2c := change1.Config[4]
	//assert.Equal(t, leaf2c.GetValue().ValueToString(), "abc")
	//
	//config := device1V.ExtractFullConfig(change1, changeStore, 0)
	//for _, c := range config {
	//	log.Infof("Path %s = %s\n", c.Path, c.GetValue().ValueToString())
	//}
	//
	//// Check the value of leaf2c after - the value from the early change should be the same
	//// This is here because ExtractFullConfig had been inadvertently changing the value
	//change1, ok = changeStore[B64(c1ID)]
	//assert.Assert(t, ok)
	//assert.Equal(t, len(change1.Config), 11)
	//leaf2c = change1.Config[4]
	//assert.Equal(t, leaf2c.GetValue().ValueToString(), "abc")
	//
	//for i := 0; i < len(Config1Paths); i++ {
	//	checkPathvalue(t, config, i,
	//		Config1Paths[0:11], Config1Values[0:11], Config1Types[0:11])
	//}
}