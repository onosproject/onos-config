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
	"encoding/base64"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	devicetype "github.com/onosproject/onos-config/pkg/types/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"gotest.tools/assert"
	log "k8s.io/klog"
	"os"
	"strings"
	"testing"
)

const (
	Test1Cont1A                  = "/cont1a"
	Test1Cont1ACont2A            = "/cont1a/cont2a"
	Test1Cont1ACont2ALeaf2A      = "/cont1a/cont2a/leaf2a"
	Test1Cont1ACont2ALeaf2B      = "/cont1a/cont2a/leaf2b"
	Test1Cont1ACont2ALeaf2C      = "/cont1a/cont2a/leaf2c"
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

var Config1Types = [11]devicechangetypes.ValueType{
	devicechangetypes.ValueType_EMPTY, // 0
	devicechangetypes.ValueType_EMPTY, // 0
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_FLOAT, // 3
	devicechangetypes.ValueType_STRING,
	devicechangetypes.ValueType_STRING, // 5
	devicechangetypes.ValueType_EMPTY,
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_EMPTY,
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_STRING, // 10
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

var Config1PreviousTypes = [13]devicechangetypes.ValueType{
	devicechangetypes.ValueType_EMPTY, // 0
	devicechangetypes.ValueType_EMPTY, // 0
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_FLOAT, // 3
	devicechangetypes.ValueType_STRING,
	devicechangetypes.ValueType_STRING, // 5
	devicechangetypes.ValueType_EMPTY,
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_EMPTY,
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_EMPTY, // 10
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_STRING,
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

var Config1FirstTypes = [11]devicechangetypes.ValueType{
	devicechangetypes.ValueType_EMPTY, // 0
	devicechangetypes.ValueType_EMPTY, // 0
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_FLOAT, // 3
	devicechangetypes.ValueType_STRING,
	devicechangetypes.ValueType_STRING, // 5
	devicechangetypes.ValueType_EMPTY,
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_EMPTY,
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_STRING, // 10
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

var Config2Types = [11]devicechangetypes.ValueType{
	devicechangetypes.ValueType_EMPTY, // 0
	devicechangetypes.ValueType_EMPTY, // 0
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_FLOAT, // 3
	devicechangetypes.ValueType_STRING,
	devicechangetypes.ValueType_STRING, // 5
	devicechangetypes.ValueType_EMPTY,
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_EMPTY,
	devicechangetypes.ValueType_UINT,
	devicechangetypes.ValueType_STRING, // 10
}

const (
	Device1ID = devicetype.ID("Device1-1.0.0")
	Device2ID = devicetype.ID("Device2-1.0.0")
)

var B64 = base64.StdEncoding.EncodeToString

func makeDevice(ID devicetopo.ID) *devicetopo.Device {
	return &devicetopo.Device{
		ID:          ID,
		Revision:    0,
		Address:     "",
		Target:      "",
		Version:     "1.0.0",
		Timeout:     nil,
		Credentials: devicetopo.Credentials{},
		TLS:         devicetopo.TlsConfig{},
		Type:        "TestDevice",
		Role:        "",
		Attributes:  nil,
		Protocols:   nil,
	}
}

func setUp(t *testing.T) (*devicechangetypes.DeviceChange, *devicechangetypes.DeviceChange, device.Store) {
	log.SetOutput(os.Stdout)
	ctrl := gomock.NewController(t)
	mockChangeStore := mockstore.NewMockDeviceChangesStore(ctrl)
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

	device1 := makeDevice(devicetopo.ID(Device1ID))
	device2 := makeDevice(devicetopo.ID(Device2ID))

	change1 := devicechangetypes.Change{
		Values: []*devicechangetypes.ChangeValue{
			config1Value01, config1Value02, config1Value03,
			config1Value04, config1Value05, config1Value06,
			config1Value07, config1Value08, config1Value09,
			config1Value10, config1Value11,
		},
		DeviceID:      devicetype.ID(device1.ID),
		DeviceVersion: "1.0.0",
	}
	deviceChange1 := &devicechangetypes.DeviceChange{
		Change: &change1,
		ID:     "Change1",
		Status: changetypes.Status{
			Phase:   changetypes.Phase_CHANGE,
			State:   changetypes.State_COMPLETE,
			Reason:  0,
			Message: "",
		},
	}

	config2Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2B, devicechangetypes.NewTypedValueFloat(ValueLeaf2B314), false)
	config2Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout3, devicechangetypes.NewTypedValueEmpty(), false)
	config2Value03, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout3Txpwr, devicechangetypes.NewTypedValueUint64(ValueTxout3Txpwr16), false)

	change2 := devicechangetypes.Change{
		Values: []*devicechangetypes.ChangeValue{
			config2Value01, config2Value02, config2Value03,
		},
		DeviceID:      devicetype.ID(device1.ID),
		DeviceVersion: "1.0.0",
	}
	deviceChange2 := &devicechangetypes.DeviceChange{
		Change: &change2,
		ID:     "Change2",
		Status: changetypes.Status{
			Phase:   changetypes.Phase_CHANGE,
			State:   changetypes.State_COMPLETE,
			Reason:  0,
			Message: "",
		},
	}

	config3Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2C, devicechangetypes.NewTypedValueString(ValueLeaf2CDef), false)
	config3Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout2, devicechangetypes.NewTypedValueEmpty(), true)

	change3 := devicechangetypes.Change{
		Values: []*devicechangetypes.ChangeValue{
			config3Value01, config3Value02,
		},
		DeviceID:      devicetype.ID(device1.ID),
		DeviceVersion: "1.0.0",
	}
	deviceChange3 := &devicechangetypes.DeviceChange{
		Change: &change3,
		ID:     "Change3",
		Status: changetypes.Status{
			Phase:   changetypes.Phase_CHANGE,
			State:   changetypes.State_COMPLETE,
			Reason:  0,
			Message: "",
		},
	}

	config4Value01, _ := devicechangetypes.NewChangeValue(Test1Cont1ACont2ALeaf2C, devicechangetypes.NewTypedValueString(ValueLeaf2CGhi), false)
	config4Value02, _ := devicechangetypes.NewChangeValue(Test1Cont1AList2ATxout1, devicechangetypes.NewTypedValueEmpty(), true)
	change4 := devicechangetypes.Change{
		Values: []*devicechangetypes.ChangeValue{
			config4Value01, config4Value02,
		},
		DeviceID:      devicetype.ID(device2.ID),
		DeviceVersion: "1.0.0",
	}
	deviceChange4 := &devicechangetypes.DeviceChange{
		Change: &change4,
		ID:     "Change4",
		Status: changetypes.Status{
			Phase:   changetypes.Phase_CHANGE,
			State:   changetypes.State_COMPLETE,
			Reason:  0,
			Message: "",
		},
	}

	mockChangeStore.EXPECT().Get(deviceChange1.ID).Return(deviceChange1, nil).AnyTimes()
	mockChangeStore.EXPECT().Get(deviceChange2.ID).Return(deviceChange2, nil).AnyTimes()
	mockChangeStore.EXPECT().Get(deviceChange3.ID).Return(deviceChange3, nil).AnyTimes()
	mockChangeStore.EXPECT().Get(deviceChange4.ID).Return(deviceChange4, nil).AnyTimes()

	mockChangeStore.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(device devicetype.VersionedID, c chan<- *devicechangetypes.DeviceChange) (stream.Context, error) {
			go func() {
				c <- deviceChange1
				c <- deviceChange2
				if strings.Contains(string(device), "Device1") {
					c <- deviceChange3
				} else {
					c <- deviceChange4
				}
				close(c)
			}()
			return stream.NewContext(func() {

			}), nil
		}).AnyTimes()

	return deviceChange3, deviceChange4, mockChangeStore
}

func checkPathValue(t *testing.T, config []*devicechangetypes.PathValue, index int,
	expPaths []string, expValues [][]byte, expTypes []devicechangetypes.ValueType) {

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

func Test_device1_version(t *testing.T) {
	device1V, _, changeStore := setUp(t)

	log.Info("Device ", device1V.Change.DeviceID, " (latest) Changes:")
	for idx, cid := range device1V.Change.Values {
		log.Infof("%d: %s\n", idx, cid)
	}

	assert.Equal(t, device1V.Change.DeviceID, Device1ID)

	// Check the value of leaf2c before
	change1, ok := changeStore.Get("Change1")
	assert.Assert(t, ok)
	assert.Equal(t, len(change1.Change.Values), 11)
	leaf2c := change1.Change.Values[4]
	assert.Equal(t, leaf2c.GetValue().ValueToString(), "abc")

	pathValues, ok := ExtractFullConfig(device1V.Change.GetVersionedDeviceID(), change1.Change, changeStore, 0)
	assert.Assert(t, ok)
	for _, c := range pathValues {
		log.Infof("Path %s = %s\n", c.Path, c.GetValue().ValueToString())
	}
	//
	// Check the value of leaf2c after - the value from the early change should be the same
	// This is here because ExtractFullConfig had been inadvertently changing the value
	change1, ok = changeStore.Get("Change1")
	assert.Assert(t, ok)
	assert.Equal(t, len(change1.Change.Values), 11)
	leaf2c = change1.Change.Values[4]
	assert.Equal(t, leaf2c.GetValue().ValueToString(), "abc")

	for i := 0; i < len(Config1Paths); i++ {
		checkPathValue(t, pathValues, i,
			Config1Paths[0:11], Config1Values[0:11], Config1Types[0:11])
	}
}

func Test_device1_prev_version(t *testing.T) {
	device1V, _, changeStore := setUp(t)

	const changePrevious = 1
	log.Info("Configuration ", device1V.Change.DeviceID, " (n-1) Changes:")
	for idx, cid := range device1V.Change.Values[0 : len(device1V.Change.Values)-changePrevious] {
		log.Infof("%d: %s\n", idx, cid)
	}

	assert.Equal(t, device1V.Change.DeviceID, Device1ID)

	config, _ := ExtractFullConfig(device1V.Change.GetVersionedDeviceID(), nil, changeStore, changePrevious)
	for _, c := range config {
		log.Infof("Path %s = %s\n", c.Path, c.GetValue().ValueToString())
	}

	for i := 0; i < len(Config1PreviousPaths); i++ {
		checkPathValue(t, config, i,
			Config1PreviousPaths[0:13], Config1PreviousValues[0:13], Config1PreviousTypes[0:13])
	}
}

func Test_device1_first_version(t *testing.T) {
	device1V, _, changeStore := setUp(t)
	const changePrevious = 2
	log.Info("Configuration ", device1V.Change.DeviceID, " (n-2) Changes:")
	for idx, cid := range device1V.Change.Values[0 : len(device1V.Change.Values)-changePrevious] {
		log.Infof("%d: %s\n", idx, cid)
	}

	assert.Equal(t, device1V.Change.DeviceID, Device1ID)

	config, _ := ExtractFullConfig(device1V.Change.GetVersionedDeviceID(), nil, changeStore, changePrevious)
	for _, c := range config {
		log.Infof("Path %s = %s\n", c.Path, c.GetValue().ValueToString())
	}

	for i := 0; i < len(Config1FirstPaths); i++ {
		checkPathValue(t, config, i,
			Config1FirstPaths[0:11], Config1FirstValues[0:11], Config1FirstTypes[0:11])
	}
}

func Test_device1_invalid_version(t *testing.T) {
	device1V, _, changeStore := setUp(t)
	const changePrevious = 3

	assert.Equal(t, device1V.Change.DeviceID, Device1ID)

	config, _ := ExtractFullConfig(device1V.Change.GetVersionedDeviceID(), nil, changeStore, changePrevious)
	if len(config) > 0 {
		t.Errorf("Not expecting any values for change (n-3). Got %d", len(config))
	}

}

func Test_device2_version(t *testing.T) {
	_, device2V, changeStore := setUp(t)
	log.Info("Configuration ", device2V.Change.DeviceID, " (latest) Changes:")
	for idx, cid := range device2V.Change.Values {
		log.Infof("%d: %s\n", idx, cid)
	}

	assert.Equal(t, device2V.Change.DeviceID, Device2ID)

	config, _ := ExtractFullConfig(device2V.Change.GetVersionedDeviceID(), nil, changeStore, 0)
	for _, c := range config {
		log.Infof("Path %s = %s\n", c.Path, c.GetValue().ValueToString())
	}

	for i := 0; i < len(Config2Paths); i++ {
		checkPathValue(t, config, i,
			Config2Paths[0:11], Config2Values[0:11], Config2Types[0:11])
	}
}
