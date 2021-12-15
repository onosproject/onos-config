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
	"strings"
	"testing"
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
	server, mocks := setUpForGetSetTests(t)
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

// Test_doSingleSetEmptyString deals with setting an empty value on a string
func Test_doSingleSetEmptyString(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "leaf1a"})
	typedValue := gnmi.TypedValue_StringVal{StringVal: ""}
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
	assert.EqualError(t, setError,
		`rpc error: code = InvalidArgument desc = rpc error: code = InvalidArgument desc = Empty string not allowed. Delete attribute instead. /cont1a/leaf1a`)
}

// Test_doSingleSet shows list within a list with leafref keys and double key
func Test_doSingleSetList(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	prefix := &gnmi.Path{Elem: prefixElemsRefs.Elem, Target: "Device1"}

	list5K1Path, _ := utils.ParseGNMIElements(utils.SplitPath("/list5[key1=5][key2=2]/key1"))
	// Should be able to use a number as string index.
	list5K1Value := gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "5"}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list5K1Path.Elem}, Val: &list5K1Value})

	// Can use a numeric key in an index. When this is done we have to explicitly specify the key
	// otherwise it will be assumed as string
	list5K2Path, _ := utils.ParseGNMIElements(utils.SplitPath("/list5[key1=5][key2=2]/key2"))
	list5K2Value := gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: 2}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list5K2Path.Elem}, Val: &list5K2Value})
	list4list4aKey1Path, _ := utils.ParseGNMIElements(utils.SplitPath("/list4[id=first]/list4a[fkey1=5][fkey2=2]/fkey1"))
	list4list4aKey1Value := gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "5"}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list4list4aKey1Path.Elem}, Val: &list4list4aKey1Value})
	list4list4aKey2Path, _ := utils.ParseGNMIElements(utils.SplitPath("/list4[id=first]/list4a[fkey1=5][fkey2=2]/fkey2"))
	list4list4aKey2Value := gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: 2}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list4list4aKey2Path.Elem}, Val: &list4list4aKey2Value})
	list4list4aDispnamePath, _ := utils.ParseGNMIElements(utils.SplitPath("/list4[id=first]/list4a[fkey1=5][fkey2=2]/displayname"))
	list4list4aDispnameValue := gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "no longer than 20"}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list4list4aDispnamePath.Elem}, Val: &list4list4aDispnameValue})

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.NoError(t, setError)
	assert.Equal(t, 5, len(setResponse.Response))
	for _, resp := range setResponse.Response {
		switch path := strings.ReplaceAll(resp.Path.String(), "  ", " "); path {
		case
			`elem:{name:"cont1a"} elem:{name:"list5" key:{key:"key1" value:"5"} key:{key:"key2" value:"2"}} elem:{name:"key1"} target:"Device1"`,
			`elem:{name:"cont1a"} elem:{name:"list5" key:{key:"key1" value:"5"} key:{key:"key2" value:"2"}} elem:{name:"key2"} target:"Device1"`,
			`elem:{name:"cont1a"} elem:{name:"list4" key:{key:"id" value:"first"}} elem:{name:"list4a" key:{key:"fkey1" value:"5"} key:{key:"fkey2" value:"2"}} elem:{name:"displayname"} target:"Device1"`,
			`elem:{name:"cont1a"} elem:{name:"list4" key:{key:"id" value:"first"}} elem:{name:"list4a" key:{key:"fkey1" value:"5"} key:{key:"fkey2" value:"2"}} elem:{name:"fkey1"} target:"Device1"`,
			`elem:{name:"cont1a"} elem:{name:"list4" key:{key:"id" value:"first"}} elem:{name:"list4a" key:{key:"fkey1" value:"5"} key:{key:"fkey2" value:"2"}} elem:{name:"fkey2"} target:"Device1"`:
			assert.Equal(t, resp.GetOp().String(), gnmi.UpdateResult_UPDATE.String())
		default:
			t.Errorf("unexpected path %s", path)
		}
	}
}

// Test_doSingleSet shows how a value of 1 list can be set on a target - using prefix
func Test_doSingleSetListIndexInvalid(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	prefix := &gnmi.Path{Elem: prefixElemsRefs.Elem, Target: "Device1"}
	pathElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/list5[key1=abc][key2=7]/key2"))
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem}
	typedValue := gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: 6}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &typedValue})

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	_, setError := server.Set(context.Background(), &setRequest)
	assert.EqualError(t, setError, "rpc error: code = InvalidArgument desc = index attribute key2=6 does not match /cont1a/list5[key1=abc][key2=7]/key2")
}

// Test_do2SetsOnSameTarget shows how 2 paths can be changed on a target
func Test_do2SetsOnSameTarget(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
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
	nwChange, err := server.networkChangesStore.Get(networkchange.ID(changeUUID))
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
	server, mocks := setUpForGetSetTests(t)
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
	nwChange, err := server.networkChangesStore.Get(networkchange.ID(changeUUID))
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
	server, mocks := setUpForGetSetTests(t)
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

// Test_doDoubleDelete shows how a value of 2 paths can be deleted on a target
// One of them is a list item with a non-index key (tx-power) - if an index key is specified
// then the whole list entry should be deleted
func Test_doDoubleDelete(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	assert.NoError(t, err)
	deletePath := &gnmi.Path{Elem: pathElemsRefs.Elem, Target: "Device1"}
	deletePaths = append(deletePaths, deletePath)
	pathElemsRefs2, err := utils.ParseGNMIElements([]string{"cont1a", "list2a[name=first]", "tx-power"})
	assert.NoError(t, err)
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

		switch strings.ReplaceAll(path.String(), "  ", " ") {
		case `elem:{name:"cont1a"} elem:{name:"cont2a"} elem:{name:"leaf2a"} target:"Device1"`,
			`elem:{name:"cont1a"} elem:{name:"list2a" key:{key:"name" value:"first"}} elem:{name:"tx-power"} target:"Device1"`:
		default:
			t.Errorf("unexpected response in delete. %s", path.String())
		}
	}

	// Check that an the network change ID is given in extension 100 and that the device is currently disconnected
	assert.Equal(t, len(setResponse.Extension), 1)

	extension := setResponse.Extension[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(GnmiExtensionNetwkChangeID))
	assert.Equal(t, string(extension.Msg), "TestChange")
}

// Test_doDeleteByIndex shows how a delete can be on just the index
// There are 3 ways to delete an item in a list
// 1) Like this test - just the list item element is given (not a child attribute)
// 2) Like above - an non-index attribute is given - in this case the list entry stays, only the non-idx attr is removed
// 3) Like next below - an index attrib is given for delete - the result should be the same as 1)
func Test_doDeleteByIndexNoAttr(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	list4aFirstPath, err := utils.ParseGNMIElements([]string{"cont1a", "list4[id=first]", "list4a[fkey1=abc][fkey2=8]"})
	assert.NoError(t, err)
	list4aFirstValue := &gnmi.Path{Elem: list4aFirstPath.Elem, Target: "Device1"}
	deletePaths = append(deletePaths, list4aFirstValue)

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.NoError(t, setError, "Unexpected error from gnmi Set")
	assert.NotNil(t, setResponse, "Expected setResponse to exist")
	for _, r := range setResponse.Response {
		path := r.Path
		assert.Equal(t, path.Target, "Device1")
		switch strings.ReplaceAll(path.String(), "  ", " ") {
		case `elem:{name:"cont1a"} elem:{name:"list4" key:{key:"id" value:"first"}} elem:{name:"list4a" key:{key:"fkey1" value:"abc"} key:{key:"fkey2" value:"8"}} target:"Device1"`:
			assert.Equal(t, r.Op.String(), gnmi.UpdateResult_DELETE.String())
		default:
			t.Errorf("unexpected response in delete. %s", path.String())
		}
	}
}

func Test_doDeleteByIndexWithIndexAttr(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	list4aFirstPath, err := utils.ParseGNMIElements([]string{"cont1a", "list4[id=first]", "list4a[fkey1=abc][fkey2=8]", "fkey1"})
	assert.NoError(t, err)
	list4aFirstValue := &gnmi.Path{Elem: list4aFirstPath.Elem, Target: "Device1"}
	deletePaths = append(deletePaths, list4aFirstValue)

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.NoError(t, setError, "Unexpected error from gnmi Set")
	assert.NotNil(t, setResponse, "Expected setResponse to exist")
	for _, r := range setResponse.Response {
		path := r.Path
		assert.Equal(t, path.Target, "Device1")
		switch strings.ReplaceAll(path.String(), "  ", " ") {
		case `elem:{name:"cont1a"} elem:{name:"list4" key:{key:"id" value:"first"}} elem:{name:"list4a" key:{key:"fkey1" value:"abc"} key:{key:"fkey2" value:"8"}} target:"Device1"`:
			assert.Equal(t, r.Op.String(), gnmi.UpdateResult_DELETE.String())
		default:
			t.Errorf("unexpected response in delete. %s", path.String())
		}
	}
}

func Test_doDeleteTopLevelObject(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	cont1aPath, err := utils.ParseGNMIElements([]string{"cont1a"})
	assert.NoError(t, err)
	cont1aTarget := &gnmi.Path{Elem: cont1aPath.Elem, Target: "Device1"}
	deletePaths = append(deletePaths, cont1aTarget)

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.NoError(t, setError, "Unexpected error from gnmi Set")
	assert.NotNil(t, setResponse, "Expected setResponse to exist")
	for _, r := range setResponse.Response {
		path := r.Path
		assert.Equal(t, path.Target, "Device1")
		switch strings.ReplaceAll(path.String(), "  ", " ") {
		case `elem:{name:"cont1a"} target:"Device1"`:
			assert.Equal(t, r.Op.String(), gnmi.UpdateResult_DELETE.String())
		default:
			t.Errorf("unexpected response in delete. %s", path.String())
		}
	}
}

// Test_doUpdateDeleteSet shows how a request with a delete and an update can be applied on a target
func Test_doUpdateDeleteSet(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
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
	server, mocks := setUpForGetSetTests(t)
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
	server, mocks := setUpForGetSetTests(t)
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
	server, mocks := setUpForGetSetTests(t)
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
	server, mocks := setUpForGetSetTests(t)
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

// Test_doSingleSetListLeafRef make a list entry whose key is a leaf ref to another list. Adding that reference entry at same time
func Test_doSingleSetListLeafRef(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	prefix := &gnmi.Path{Elem: prefixElemsRefs.Elem, Target: "Device1"}

	// The value we are referring to
	list2aSecondPath, _ := utils.ParseGNMIElements(utils.SplitPath("/list2a[name=second]/name")) // Is the index
	list2aSecondValue := gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "second"}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list2aSecondPath.Elem}, Val: &list2aSecondValue})

	// The path includes a reference 'first' to an existing index
	list4FirstLeaf4bPath, _ := utils.ParseGNMIElements(utils.SplitPath("/list4[id=first]/leaf4b"))
	list4FirstLeaf4bValue := gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "changed value"}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list4FirstLeaf4bPath.Elem}, Val: &list4FirstLeaf4bValue})

	// The path includes a reference 'second' to the newly created index above
	list4SecondLeaf4bPath, _ := utils.ParseGNMIElements(utils.SplitPath("/list4[id=second]/leaf4b"))
	list4SecondLeaf4bValue := gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "4b second value"}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list4SecondLeaf4bPath.Elem}, Val: &list4SecondLeaf4bValue})

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.NoError(t, setError)
	assert.NotNil(t, setResponse)
	assert.Equal(t, len(setResponse.Response), 3)
	for _, resp := range setResponse.Response {
		switch path := strings.ReplaceAll(resp.Path.String(), "  ", " "); path {
		case
			`elem:{name:"cont1a"} elem:{name:"list2a" key:{key:"name" value:"second"}} elem:{name:"name"} target:"Device1"`,
			`elem:{name:"cont1a"} elem:{name:"list4" key:{key:"id" value:"first"}} elem:{name:"leaf4b"} target:"Device1"`,
			`elem:{name:"cont1a"} elem:{name:"list4" key:{key:"id" value:"second"}} elem:{name:"leaf4b"} target:"Device1"`:
			assert.Equal(t, resp.GetOp().String(), gnmi.UpdateResult_UPDATE.String())
		default:
			t.Errorf("unexpected response path %v", path)
		}
	}
}

// Test_doSingleSetListInvalidLeafRef make a list entry whose key is a leaf ref to another list, but that ref does not exist
func Test_doSingleSetListInvalidLeafRef(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	prefix := &gnmi.Path{Elem: prefixElemsRefs.Elem, Target: "Device1"}

	// The path includes a reference 'second' to the newly created index above
	list4SecondLeaf4bPath, _ := utils.ParseGNMIElements(utils.SplitPath("/list4[id=second]/leaf4b"))
	list4SecondLeaf4bValue := gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "4b second value"}}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &gnmi.Path{Elem: list4SecondLeaf4bPath.Elem}, Val: &list4SecondLeaf4bValue})

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.Contains(t, setError.Error(), `rpc error: code = InvalidArgument desc = validation error field name Id value second (string ptr) schema path /device/cont1a/list4/id has leafref path /cont1a/list2a/name not equal to any target nodes`)
	assert.Nil(t, setResponse)
}

// Test_doDeleteOfReferencedEntryFromList try to remove a list entry and its reference in another list
func Test_doDeleteOfReferencedEntryFromList(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	prefix := &gnmi.Path{Elem: prefixElemsRefs.Elem, Target: "Device1"}

	// The path "first" is already referenced by list4
	list2aNamePath, _ := utils.ParseGNMIElements(utils.SplitPath("/list2a[name=first]/name"))
	deletePaths = append(deletePaths, list2aNamePath)
	// The path "first" is to be deleted in list4
	list4IdPath, _ := utils.ParseGNMIElements(utils.SplitPath("/list4[id=first]/id"))
	deletePaths = append(deletePaths, list4IdPath)

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.NoError(t, setError)
	assert.NotNil(t, setResponse)
	assert.Equal(t, 2, len(setResponse.Response))
	for _, resp := range setResponse.Response {
		switch path := strings.ReplaceAll(resp.Path.String(), "  ", " "); path {
		case
			`elem:{name:"cont1a"} elem:{name:"list2a" key:{key:"name" value:"first"}} target:"Device1"`,
			`elem:{name:"cont1a"} elem:{name:"list4" key:{key:"id" value:"first"}} target:"Device1"`:
			assert.Equal(t, resp.GetOp().String(), gnmi.UpdateResult_DELETE.String())
		default:
			t.Errorf("unexpected response path %v", path)
		}
	}
}

// Test_doDeleteOfReferencedEntryFromListInvalid try to remove a list entry when it is already referenced by another list
func Test_doDeleteOfReferencedEntryFromListInvalid(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	prefix := &gnmi.Path{Elem: prefixElemsRefs.Elem, Target: "Device1"}

	// The path "first" is already referenced by list4 - delete should not be allowed
	list2aNamePath, _ := utils.ParseGNMIElements(utils.SplitPath("/list2a[name=first]/name"))
	deletePaths = append(deletePaths, list2aNamePath)

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.Contains(t, setError.Error(), "rpc error: code = InvalidArgument desc = validation error pointed-to value with path /cont1a/list2a/name from field Id value first (string ptr) schema /device/cont1a/list4/id is empty set")
	assert.Nil(t, setResponse)
}

func Test_findPathFromModel(t *testing.T) {
	server, _ := setUpForGetSetTests(t)
	td1Mp, err := server.modelRegistry.GetPlugin("TestDevice-1.0.0")
	assert.NoError(t, err)

	// Regular leaf
	isExactMatch, rwPath, err := findPathFromModel("/cont1a/leaf1a", td1Mp.ReadWritePaths, true)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "leaf1a", rwPath.AttrName)

	// Container
	isExactMatch, rwPath, err = findPathFromModel("/cont1a", td1Mp.ReadWritePaths, false)
	assert.NoError(t, err)
	assert.False(t, isExactMatch)
	assert.True(t, len(rwPath.AttrName) > 0)

	_, _, err = findPathFromModel("/cont1a", td1Mp.ReadWritePaths, true)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = unable to find exact match for RW model path /cont1a. 21 paths inspected")

	// Another leaf
	isExactMatch, rwPath, err = findPathFromModel("/cont1a/cont2a/leaf2a", td1Mp.ReadWritePaths, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "leaf2a", rwPath.AttrName)

	// List Anon Index entry
	isExactMatch, rwPath, err = findPathFromModel("/cont1a/list2a[name=test]", td1Mp.ReadWritePaths, false)
	assert.NoError(t, err)
	assert.False(t, isExactMatch)
	assert.Equal(t, "name", rwPath.AttrName)

	// List key exact
	isExactMatch, rwPath, err = findPathFromModel("/cont1a/list2a[name=test]/name", td1Mp.ReadWritePaths, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "name", rwPath.AttrName)

	// List non key
	isExactMatch, rwPath, err = findPathFromModel("/cont1a/list2a[name=test]/tx-power", td1Mp.ReadWritePaths, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "tx-power", rwPath.AttrName)

	// Double keyed List key attr
	isExactMatch, rwPath, err = findPathFromModel("/cont1a/list5[key1=test][key2=10]/key1", td1Mp.ReadWritePaths, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "key1", rwPath.AttrName)

	// Double keyed List - just list
	isExactMatch, rwPath, err = findPathFromModel("/cont1a/list5[key1=test][key2=10]", td1Mp.ReadWritePaths, false)
	assert.NoError(t, err)
	assert.False(t, isExactMatch)
	switch rwPath.AttrName {
	case "key1", "key2":
	default:
		t.Errorf("unexpected attr name %s", rwPath.AttrName)
	}

	// Double keyed List non-key attr
	isExactMatch, rwPath, err = findPathFromModel("/cont1a/list5[key1=test][key2=10]/leaf5a", td1Mp.ReadWritePaths, false)
	assert.NoError(t, err)
	assert.True(t, isExactMatch)
	assert.Equal(t, "leaf5a", rwPath.AttrName)

	// Double keyed List invalid attr
	_, _, err = findPathFromModel("/cont1a/list5[key1=test][key2=10]/invalid", td1Mp.ReadWritePaths, false)
	assert.EqualError(t, err, "rpc error: code = InvalidArgument desc = unable to find RW model path /cont1a/list5[key1=test][key2=10]/invalid ( without index /cont1a/list5/invalid). 21 paths inspected")
}

func Test_deleteReferencedContainerList(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	setUpChangesMock(mocks)

	setUpPathsForGetSetTests()

	// First add 2 instances of cont10/list10
	prefix := &gnmi.Path{Target: "Device1"}

	// Now delete cont10 - should not be possible because item is referenced by /cont20/list20[list20id=11]
	var updatedPaths []*gnmi.Update
	var replacedPaths []*gnmi.Update
	var deletePaths []*gnmi.Path
	cont10DeletePath, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a/cont2"))
	deletePaths = append(deletePaths, &gnmi.Path{Elem: cont10DeletePath.Elem})

	var setRequest = gnmi.SetRequest{
		Prefix:  prefix,
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)
	assert.Errorf(t, setError, "Expecting error as /cont1a/cont2 is used as a leafref")
	assert.Nil(t, setResponse)
}
