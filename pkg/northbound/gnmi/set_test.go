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

package gnmi

import (
	"context"
	"github.com/golang/mock/gomock"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"regexp"
	"strconv"
	"testing"
)

const (
	networkChange1 = "NetworkChange1"
	device1        = "Device1"
	deviceVersion1 = "1.0.0"
)

var uuidRegex = regexp.MustCompile("[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}")

func setUpLocalhostDeviceCache(mocks *AllMocks) {
	const localhost1 = "localhost-1"
	const localhost2 = "localhost-2"
	// Setting up device cache
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("localhost-1")).Return([]*cache.Info{
		{
			DeviceID: localhost1,
			Version:  "1.0.0",
			Type:     "TestDevice",
		},
	}).AnyTimes()
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("localhost-2")).Return([]*cache.Info{
		{
			DeviceID: localhost2,
			Version:  "1.0.0",
			Type:     "TestDevice",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).AnyTimes()
}

func setUpDeviceWithMultipleVersions(mocks *AllMocks, deviceName string) {
	// Setting up mocks
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID(deviceName)).Return([]*cache.Info{
		{
			DeviceID: devicetype.ID(deviceName),
			Version:  "1.0.0",
			Type:     "TestDevice",
		},
		{
			DeviceID: devicetype.ID(deviceName),
			Version:  "2.0.0",
			Type:     "TestDevice",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(topodevice.ID(deviceName)).Return(nil, nil).AnyTimes()
}

// Test_doSingleSet shows how a value of 1 path can be set on a target
func Test_doSingleSet(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_UintVal{UintVal: 11}
	value := gnmi.TypedValue{Value: &typedValue}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "Device1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	ext100Name := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  GnmiExtensionNetwkChangeID,
			Msg: []byte("TestChange"),
		},
	}

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Extension: []*gnmi_ext.Extension{{
			Ext: &ext100Name,
		}},
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NoError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.NotNil(t, setResponse, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 1)

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_UPDATE.String())

	path := setResponse.Response[0].Path

	assert.Equal(t, path.Target, "Device1")

	assert.Equal(t, len(path.Elem), 3, "Expected 3 path elements")

	assert.Equal(t, path.Elem[0].Name, "cont1a")
	assert.Equal(t, path.Elem[1].Name, "cont2a")
	assert.Equal(t, path.Elem[2].Name, "leaf2a")

	// Check that an the network change ID is given in extension 100 and that devices show up as disconnected
	assert.Equal(t, len(setResponse.Extension), 1)

	extensionChgID := setResponse.Extension[0].GetRegisteredExt()
	assert.Equal(t, extensionChgID.Id.String(), strconv.Itoa(GnmiExtensionNetwkChangeID))
	assert.Equal(t, string(extensionChgID.Msg), "TestChange")
}

// Test_doSingleSet shows how a value of 1 list can be set on a target - using prefix
func Test_doSingleSetList(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	pathElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/list2a[name=aaa/bbb]/tx-power"))
	typedValue := gnmi.TypedValue_UintVal{UintVal: 16}
	value := gnmi.TypedValue{Value: &typedValue}
	prefix := &gnmi.Path{Elem: prefixElemsRefs.Elem, Target: "Device1"}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	ext100Name := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  GnmiExtensionNetwkChangeID,
			Msg: []byte("TestChange"),
		},
	}

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Extension: []*gnmi_ext.Extension{{
			Ext: &ext100Name,
		}},
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NoError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.NotNil(t, setResponse, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 1)

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_UPDATE.String())

	path := setResponse.Response[0].Path

	assert.Equal(t, path.Target, "Device1")
	assert.Equal(t, len(path.Elem), 3, "Expected 3 path elements")

	assert.Equal(t, path.Elem[0].Name, "cont1a")
	assert.Equal(t, path.Elem[1].Name, "list2a")
	assert.Equal(t, path.Elem[2].Name, "tx-power")

	// Check that an the network change ID is given in extension 100 and that the device is currently disconnected
	assert.Equal(t, len(setResponse.Extension), 1)

	extension := setResponse.Extension[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(GnmiExtensionNetwkChangeID))
	assert.Equal(t, string(extension.Msg), "TestChange")
}

// Test_doSingleSet shows how a value of 1 list can be set on a target - using prefix
func Test_doSingleSetListIndexInvalid(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	pathElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/list2a[name=aa/b]/name"))
	typedValue := gnmi.TypedValue_StringVal{StringVal: "aa/c"}
	value := gnmi.TypedValue{Value: &typedValue}
	prefix := &gnmi.Path{Elem: prefixElemsRefs.Elem, Target: "Device1"}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	ext100Name := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  GnmiExtensionNetwkChangeID,
			Msg: []byte("TestChange"),
		},
	}

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Extension: []*gnmi_ext.Extension{{
			Ext: &ext100Name,
		}},
	}

	_, setError := server.Set(context.Background(), &setRequest)
	assert.Error(t, setError, "expected error from gnmi Set because of invalid index")
}

// Test_do2SetsOnSameTarget shows how 2 paths can be changed on a target
func Test_do2SetsOnSameTarget(t *testing.T) {
	server, mocks, mgr := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()
	setUpLocalhostDeviceCache(mocks)

	pathElemsRefs1, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	value1Str := gnmi.TypedValue_DecimalVal{DecimalVal: &gnmi.Decimal64{Digits: 123456, Precision: 5}}
	value := gnmi.TypedValue{Value: &value1Str}

	updatePath1 := gnmi.Path{Elem: pathElemsRefs1.Elem, Target: "localhost-1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath1, Val: &value})

	pathElemsRefs2, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2g"})
	value2Bool := gnmi.TypedValue_BoolVal{BoolVal: true}
	value2 := gnmi.TypedValue{Value: &value2Bool}

	updatePath2 := gnmi.Path{Elem: pathElemsRefs2.Elem, Target: "localhost-1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath2, Val: &value2})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NoError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.NotNil(t, setResponse, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 2)

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_UPDATE.String())
	assert.Equal(t, setResponse.Response[1].Op.String(), gnmi.UpdateResult_UPDATE.String())

	// Response parts might not be in the same order because changes are stored in a map
	for _, res := range setResponse.Response {
		assert.Equal(t, res.Path.Target, "localhost-1")
	}

	assert.Equal(t, 1, len(setResponse.Extension))
	assert.Equal(t, 100, int(setResponse.Extension[0].GetRegisteredExt().Id))
	changeUUID := string(setResponse.Extension[0].GetRegisteredExt().GetMsg())
	assert.True(t, uuidRegex.MatchString(changeUUID), "ID does not match %s", uuidRegex.String())

	// Check the network change was made
	nwChange, err := mgr.NetworkChangesStore.Get(networkchange.ID(changeUUID))
	assert.NoError(t, err)
	assert.Equal(t, 1, len(nwChange.Changes))
	changes := nwChange.Changes[0]
	assert.Equal(t, "localhost-1", string(changes.DeviceID))
	assert.Equal(t, "TestDevice", string(changes.DeviceType))
	assert.Equal(t, "1.0.0", string(changes.DeviceVersion))
	assert.Equal(t, 2, len(changes.Values))
	for _, ch := range changes.Values {
		switch ch.String() {
		case
			`path:"/cont1a/cont2a/leaf2b" value:<bytes:"\001\342@" type:DECIMAL type_opts:5 type_opts:0 > `,
			`path:"/cont1a/cont2a/leaf2g" value:<bytes:"\001" type:BOOL > `:
		default:
			t.Errorf("unexpected network-change value %s", ch.String())
		}
	}
}

// Test_do2SetsOnDiffTargets shows how paths on multiple targets can be Set at
// same time
func Test_do2SetsOnDiffTargets(t *testing.T) {
	server, mocks, mgr := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()
	setUpLocalhostDeviceCache(mocks)

	// Make the same change to 2 targets
	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_UintVal{UintVal: 2}
	value := gnmi.TypedValue{Value: &typedValue}

	updatePathTgt1 := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "localhost-1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePathTgt1, Val: &value})

	updatePathTgt2 := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "localhost-2"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePathTgt2, Val: &value})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NoError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.NotNil(t, setResponse, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 2)

	// It's a map, so the order of element's is not guaranteed
	for _, result := range setResponse.Response {
		assert.Equal(t, result.Op.String(), gnmi.UpdateResult_UPDATE.String())
		assert.Equal(t, result.Path.Elem[2].Name, "leaf2a")
	}

	assert.Equal(t, 1, len(setResponse.Extension))
	assert.Equal(t, 100, int(setResponse.Extension[0].GetRegisteredExt().Id))
	changeUUID := string(setResponse.Extension[0].GetRegisteredExt().GetMsg())
	assert.True(t, uuidRegex.MatchString(changeUUID), "ID does not match %s", uuidRegex.String())

	// Check the network change was made
	nwChange, err := mgr.NetworkChangesStore.Get(networkchange.ID(changeUUID))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(nwChange.Changes))
	for _, ch := range nwChange.Changes {
		assert.Equal(t, "TestDevice", string(ch.DeviceType))
		assert.Equal(t, "1.0.0", string(ch.DeviceVersion))
		assert.Equal(t, 1, len(ch.Values))
		switch ch.DeviceID {
		case "localhost-1", "localhost-2":
			assert.Equal(t, 1, len(ch.Values))
			assert.Equal(t, `path:"/cont1a/cont2a/leaf2a" value:<bytes:"\002" type:UINT type_opts:8 > `,
				ch.Values[0].String())
		default:
			t.Errorf("unexpected DeviceID %s", ch.DeviceID)
		}
	}
}

// Test_do2SetsOnOneTargetOneOnDiffTarget shows how multiple paths on multiple
// targets can be Set at same time
func Test_do2SetsOnOneTargetOneOnDiffTarget(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()
	setUpLocalhostDeviceCache(mocks)

	// Make the same change to 2 targets
	pathElemsRefs2a, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	valueStr2a := gnmi.TypedValue_UintVal{UintVal: 3}

	pathElemsRefs2b, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	valueStr2b := gnmi.TypedValue_DecimalVal{DecimalVal: &gnmi.Decimal64{Digits: 123456, Precision: 5}}

	pathElemsRefs2g, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2g"})

	// 2 changes on Target 1
	updatedPaths = append(updatedPaths, &gnmi.Update{
		Path: &gnmi.Path{Elem: pathElemsRefs2a.Elem, Target: "localhost-1"},
		Val:  &gnmi.TypedValue{Value: &valueStr2a},
	})
	updatedPaths = append(updatedPaths, &gnmi.Update{
		Path: &gnmi.Path{Elem: pathElemsRefs2b.Elem, Target: "localhost-1"},
		Val:  &gnmi.TypedValue{Value: &valueStr2b},
	})

	// 2 changes on Target 2 - one of them is a delete
	updatedPaths = append(updatedPaths, &gnmi.Update{
		Path: &gnmi.Path{Elem: pathElemsRefs2a.Elem, Target: "localhost-2"},
		Val:  &gnmi.TypedValue{Value: &valueStr2a},
	})
	deletePaths = append(deletePaths,
		&gnmi.Path{Elem: pathElemsRefs2g.Elem, Target: "localhost-2"})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NoError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.NotNil(t, setResponse, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 4)

	// The order of response items is not ordered because it is held in a map
	for _, res := range setResponse.Response {
		if res.Op.String() == gnmi.UpdateResult_DELETE.String() {
			assert.Equal(t, res.Path.Target, "localhost-2")
		}
	}
}

// Test_doSingleDelete shows how a value of 1 path can be deleted on a target
func Test_doSingleDelete(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	deletePath := &gnmi.Path{Elem: pathElemsRefs.Elem, Target: "Device1"}
	deletePaths = append(deletePaths, deletePath)
	pathElemsRefs2, _ := utils.ParseGNMIElements([]string{"cont1a", "list2a[name=n1]"})
	deletePath2 := &gnmi.Path{Elem: pathElemsRefs2.Elem, Target: "Device1"}
	deletePaths = append(deletePaths, deletePath2)

	ext100Name := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  GnmiExtensionNetwkChangeID,
			Msg: []byte("TestChange"),
		},
	}

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Extension: []*gnmi_ext.Extension{{
			Ext: &ext100Name,
		}},
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NoError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.NotNil(t, setResponse, "Expected setResponse to exist")

	assert.Equal(t, len(setResponse.Response), 2)

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_DELETE.String())

	for _, r := range setResponse.Response {
		path := r.Path
		assert.Equal(t, path.Target, "Device1")

		switch len(path.Elem) {
		case 2: // The list item
			assert.Equal(t, path.Elem[0].Name, "cont1a")
			assert.Equal(t, path.Elem[1].Name, "list2a")
			list2aKey, ok := path.Elem[1].Key["name"]
			assert.True(t, ok, "expecting Key to have index 'name'")
			assert.Equal(t, list2aKey, "n1")

		case 3: // the leaf 2a
			assert.Equal(t, path.Elem[0].Name, "cont1a")
			assert.Equal(t, path.Elem[1].Name, "cont2a")
			assert.Equal(t, path.Elem[2].Name, "leaf2a")
		default:
			t.Errorf("unexpected response length in Single delete. %d", len(path.Elem))
		}
	}

	// Check that an the network change ID is given in extension 100 and that the device is currently disconnected
	assert.Equal(t, len(setResponse.Extension), 1)

	extension := setResponse.Extension[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(GnmiExtensionNetwkChangeID))
	assert.Equal(t, string(extension.Msg), "TestChange")
}

// Test_doUpdateDeleteSet shows how a request with a delete and an update can be applied on a target
func Test_doUpdateDeleteSet(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_UintVal{UintVal: 2}
	value := gnmi.TypedValue{Value: &typedValue}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "Device1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	pathElemsDeleteRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2g"})
	deletePath := &gnmi.Path{Elem: pathElemsDeleteRefs.Elem, Target: "Device1"}
	deletePaths = append(deletePaths, deletePath)

	ext100Name := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  GnmiExtensionNetwkChangeID,
			Msg: []byte("TestChange"),
		},
	}

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Extension: []*gnmi_ext.Extension{{
			Ext: &ext100Name,
		}},
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NoError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.NotNil(t, setResponse, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 2)

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_UPDATE.String())

	path := setResponse.Response[0].Path

	assert.Equal(t, path.Target, "Device1")

	assert.Equal(t, len(path.Elem), 3, "Expected 3 path elements")

	assert.Equal(t, path.Elem[0].Name, "cont1a")
	assert.Equal(t, path.Elem[1].Name, "cont2a")
	assert.Equal(t, path.Elem[2].Name, "leaf2a")

	assert.Equal(t, setResponse.Response[1].Op.String(), gnmi.UpdateResult_DELETE.String())

	pathDelete := setResponse.Response[1].Path

	assert.Equal(t, pathDelete.Target, "Device1")

	assert.Equal(t, len(pathDelete.Elem), 3, "Expected 3 path elements")

	assert.Equal(t, pathDelete.Elem[0].Name, "cont1a")
	assert.Equal(t, pathDelete.Elem[1].Name, "cont2a")
	assert.Equal(t, pathDelete.Elem[2].Name, "leaf2g")

	// Check that an the network change ID is given in extension 100 and that the device is currently disconnected
	assert.Equal(t, len(setResponse.Extension), 1)

	extension := setResponse.Extension[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(GnmiExtensionNetwkChangeID))
	assert.Equal(t, string(extension.Msg), "TestChange")
}

//TODO test with update and delete at the same time.

func TestSet_checkForReadOnly(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2c"})
	typedValue := gnmi.TypedValue_StringVal{
		StringVal: "test1",
	}
	value := gnmi.TypedValue{Value: &typedValue}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "Device1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	ext100Name := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  GnmiExtensionNetwkChangeID,
			Msg: []byte("TestChange"),
		},
	}

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Extension: []*gnmi_ext.Extension{{
			Ext: &ext100Name,
		}},
	}

	_, setError := server.Set(context.Background(), &setRequest)
	assert.Error(t, setError, "unable to find RW model path /cont1a/cont2a/leaf2c")
}

// Tests giving a new device without specifying a type
func TestSet_MissingDeviceType(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	// Setting up mocks
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(gomock.Any()).Return(make([]*cache.Info, 0))
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).AnyTimes()

	// Making change
	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_StringVal{StringVal: "newValue2a"}
	value := gnmi.TypedValue{Value: &typedValue}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "NoSuchDevice"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.Error(t, setError)
	assert.Contains(t, setError.Error(), "NoSuchDevice is not known")
	assert.Nil(t, setResponse)
}

// Tests giving a device with multiple versions where the request doesn't specify which one
func TestSet_ConflictingDeviceType(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	// Setting up mocks
	const deviceName = "DeviceWithMultipleVersions"
	setUpDeviceWithMultipleVersions(mocks, deviceName)

	// Making change
	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_StringVal{StringVal: "newValue2a"}
	value := gnmi.TypedValue{Value: &typedValue}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem, Target: deviceName}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.Error(t, setError)
	assert.Contains(t, setError.Error(), "Specify 1 version with extension 102")
	assert.Nil(t, setResponse)
}

// Test giving a device with a type that doesn't match the type in the store
func TestSet_BadDeviceType(t *testing.T) {
	server, mocks, _ := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	// Setting up mocks
	const deviceName = "DeviceWithMultipleVersions"
	setUpDeviceWithMultipleVersions(mocks, deviceName)

	// Making change
	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_StringVal{StringVal: "newValue2a"}
	value := gnmi.TypedValue{Value: &typedValue}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem, Target: deviceName}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	extName := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  GnmiExtensionDeviceType,
			Msg: []byte("NotTheSameType"),
		},
	}

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Extension: []*gnmi_ext.Extension{{
			Ext: &extName,
		}},
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.Error(t, setError)
	assert.Contains(t, setError.Error(), "target DeviceWithMultipleVersions type given NotTheSameType does not match expected TestDevice")
	assert.Nil(t, setResponse)
}

/*
func Test_findPathFromModel(t *testing.T) {
	_, _, mgr := setUpForGetSetTests(t)

	rwPath, err := findPathFromModel("/cont1a", mgr.ModelRegistry.ModelReadWritePaths["TestDevice-1.0.0"], false)
	assert.NoError(t, err)
	assert.NotNil(t, rwPath)

	rwPath2, err := findPathFromModel("/cont1a/list2a", mgr.ModelRegistry.ModelReadWritePaths["TestDevice-1.0.0"], false)
	assert.NoError(t, err)
	assert.NotNil(t, rwPath2)

	rwPath3, err := findPathFromModel("/cont1a/list2a[name=123]/name", mgr.ModelRegistry.ModelReadWritePaths["TestDevice-1.0.0"], true)
	assert.NoError(t, err)
	assert.NotNil(t, rwPath3)
	assert.Equal(t, []string{"4..8"}, rwPath3.Length)
}
*/
