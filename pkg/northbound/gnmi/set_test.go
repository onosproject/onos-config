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
	td1 "github.com/onosproject/config-models/modelplugin/testdevice-1.0.0/testdevice_1_0_0"
	td2 "github.com/onosproject/config-models/modelplugin/testdevice-2.0.0/testdevice_2_0_0"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/utils"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/openconfig/goyang/pkg/yang"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/assert"
	"strconv"
	"testing"
)

const (
	cont1aCont2aLeaf2a = "/cont1a/cont2a/leaf2a"
	cont1aCont2aLeaf2b = "/cont1a/cont2a/leaf2b"
	cont1aCont2aLeaf2c = "/cont1a/cont2a/leaf2c" // State
	networkChange1     = "NetworkChange1"
	device1            = "Device1"
	deviceVersion1     = "1.0.0"
)

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
	server, _ := setUpForGetSetTests(t)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_UintVal{UintVal: 16}
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

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

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
	server, _ := setUpForGetSetTests(t)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	prefixElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/cont1a"))
	pathElemsRefs, _ := utils.ParseGNMIElements(utils.SplitPath("/list2a[name=a/b]/tx-power"))
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

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

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

// Test_do2SetsOnSameTarget shows how 2 paths can be changed on a target
func Test_do2SetsOnSameTarget(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()
	setUpLocalhostDeviceCache(mocks)

	pathElemsRefs1, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	value1Str := gnmi.TypedValue_StringVal{StringVal: "newValue2b"}
	value := gnmi.TypedValue{Value: &value1Str}

	updatePath1 := gnmi.Path{Elem: pathElemsRefs1.Elem, Target: "localhost-1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath1, Val: &value})

	pathElemsRefs2, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2c"})
	value2Str := gnmi.TypedValue_StringVal{StringVal: "newValue2c"}
	value2 := gnmi.TypedValue{Value: &value2Str}

	updatePath2 := gnmi.Path{Elem: pathElemsRefs2.Elem, Target: "localhost-1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath2, Val: &value2})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 2)

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_UPDATE.String())
	assert.Equal(t, setResponse.Response[1].Op.String(), gnmi.UpdateResult_UPDATE.String())

	// Response parts might not be in the same order because changes are stored in a map
	for _, res := range setResponse.Response {
		assert.Equal(t, res.Path.Target, "localhost-1")
	}
}

// Test_do2SetsOnDiffTargets shows how paths on multiple targets can be Set at
// same time
func Test_do2SetsOnDiffTargets(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()
	setUpLocalhostDeviceCache(mocks)

	// Make the same change to 2 targets
	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_StringVal{StringVal: "2ndValue2a"}
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

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

	assert.Equal(t, len(setResponse.Response), 2)

	// It's a map, so the order of element's is not guaranteed
	for _, result := range setResponse.Response {
		assert.Equal(t, result.Op.String(), gnmi.UpdateResult_UPDATE.String())
		assert.Equal(t, result.Path.Elem[2].Name, "leaf2a")
	}
}

// Test_do2SetsOnOneTargetOneOnDiffTarget shows how multiple paths on multiple
// targets can be Set at same time
func Test_do2SetsOnOneTargetOneOnDiffTarget(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()
	setUpLocalhostDeviceCache(mocks)

	// Make the same change to 2 targets
	pathElemsRefs2a, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	valueStr2a := gnmi.TypedValue_StringVal{StringVal: "3rdValue2a"}

	pathElemsRefs2b, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2b"})
	valueStr2b := gnmi.TypedValue_StringVal{StringVal: "3rdValue2b"}

	pathElemsRefs2c, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2c"})

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
		&gnmi.Path{Elem: pathElemsRefs2c.Elem, Target: "localhost-2"})

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	setResponse, setError := server.Set(context.Background(), &setRequest)

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

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
	server, _ := setUpForGetSetTests(t)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	deletePath := &gnmi.Path{Elem: pathElemsRefs.Elem, Target: "Device1"}
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

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to exist")

	assert.Equal(t, len(setResponse.Response), 1)

	assert.Equal(t, setResponse.Response[0].Op.String(), gnmi.UpdateResult_DELETE.String())

	path := setResponse.Response[0].Path

	assert.Equal(t, path.Target, "Device1")

	assert.Equal(t, len(path.Elem), 3, "Expected 3 path elements")

	assert.Equal(t, path.Elem[0].Name, "cont1a")
	assert.Equal(t, path.Elem[1].Name, "cont2a")
	assert.Equal(t, path.Elem[2].Name, "leaf2a")

	// Check that an the network change ID is given in extension 100 and that the device is currently disconnected
	assert.Equal(t, len(setResponse.Extension), 1)

	extension := setResponse.Extension[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(GnmiExtensionNetwkChangeID))
	assert.Equal(t, string(extension.Msg), "TestChange")
}

// Test_doUpdateDeleteSet shows how a request with a delete and an update can be applied on a target
func Test_doUpdateDeleteSet(t *testing.T) {
	server, _ := setUpForGetSetTests(t)
	deletePaths, replacedPaths, updatedPaths := setUpPathsForGetSetTests()

	pathElemsRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2a"})
	typedValue := gnmi.TypedValue_StringVal{StringVal: "newValue2a"}
	value := gnmi.TypedValue{Value: &typedValue}
	updatePath := gnmi.Path{Elem: pathElemsRefs.Elem, Target: "Device1"}
	updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: &value})

	pathElemsDeleteRefs, _ := utils.ParseGNMIElements([]string{"cont1a", "cont2a", "leaf2c"})
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

	assert.NilError(t, setError, "Unexpected error from gnmi Set")

	// Check that Response is correct
	assert.Assert(t, setResponse != nil, "Expected setResponse to have a value")

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
	assert.Equal(t, pathDelete.Elem[2].Name, "leaf2c")

	// Check that an the network change ID is given in extension 100 and that the device is currently disconnected
	assert.Equal(t, len(setResponse.Extension), 1)

	extension := setResponse.Extension[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(GnmiExtensionNetwkChangeID))
	assert.Equal(t, string(extension.Msg), "TestChange")
}

//TODO test with update and delete at the same time.

func TestSet_checkForReadOnly(t *testing.T) {
	server, mgr, mocks := setUp(t)

	modelPluginTestDevice1 := MockModelPlugin{
		td1.UnzipSchema,
	}
	modelPluginTestDevice2 := MockModelPlugin{
		td2.UnzipSchema,
	}
	mgr.ModelRegistry.ModelPlugins["TestDevice-1.0.0"] = modelPluginTestDevice1
	ds1Schema, _ := modelPluginTestDevice1.Schema()
	readOnlyPathsTd1, _ := modelregistry.ExtractPaths(ds1Schema["Device"], yang.TSUnset, "", "")
	mgr.ModelRegistry.ModelReadOnlyPaths["TestDevice-1.0.0"] = readOnlyPathsTd1

	mgr.ModelRegistry.ModelPlugins["TestDevice-2.0.0"] = modelPluginTestDevice2
	td2Schema, _ := modelPluginTestDevice2.Schema()
	readOnlyPathsTd2, _ := modelregistry.ExtractPaths(td2Schema["Device"], yang.TSUnset, "", "")
	mgr.ModelRegistry.ModelReadOnlyPaths["TestDevice-2.0.0"] = readOnlyPathsTd2

	cacheInfo1v1 := cache.Info{
		DeviceID: "device-1",
		Type:     "TestDevice",
		Version:  "1.0.0",
	}

	cacheInfo1v2 := cache.Info{
		DeviceID: "device-1",
		Type:     "TestDevice",
		Version:  "2.0.0",
	}

	cacheInfo2v1 := cache.Info{
		DeviceID: "device-2",
		Type:     "TestDevice",
		Version:  "1.0.0",
	}

	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("device-1")).Return([]*cache.Info{&cacheInfo1v1, &cacheInfo1v2})
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("device-2")).Return([]*cache.Info{&cacheInfo2v1})

	updateT1 := make(map[string]*devicechange.TypedValue)
	updateT1[cont1aCont2aLeaf2a] = devicechange.NewTypedValueUint64(10)
	updateT1[cont1aCont2aLeaf2b] = devicechange.NewTypedValueString("2b on t1")

	updateT2 := make(map[string]*devicechange.TypedValue)
	updateT2[cont1aCont2aLeaf2a] = devicechange.NewTypedValueUint64(11)
	updateT2[cont1aCont2aLeaf2b] = devicechange.NewTypedValueString("2b on t2")
	updateT2[cont1aCont2aLeaf2c] = devicechange.NewTypedValueString("2c on t2 - ro attribute")

	targetUpdates := make(map[string]devicechange.TypedValueMap)
	targetUpdates["device-1"] = updateT1
	targetUpdates["device-2"] = updateT2

	err := server.checkForReadOnly("device-1", "TestDevice", "1.0.0", updateT1, make([]string, 0))
	assert.NilError(t, err, "unexpected error on T1")
	err = server.checkForReadOnly("device-2", "TestDevice", "1.0.0", updateT2, make([]string, 0))
	assert.Error(t, err, `update contains a change to a read only path /cont1a/cont2a/leaf2c. Rejected`)
}

// Tests giving a new device without specifying a type
func TestSet_MissingDeviceType(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
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

	assert.ErrorContains(t, setError, "Need to supply a type and version")
	assert.Assert(t, setResponse == nil)
}

// Tests giving a device with multiple versions where the request doesn't specify which one
func TestSet_ConflictingDeviceType(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
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

	assert.ErrorContains(t, setError, "Specify 1 version with extension 102")
	assert.Assert(t, setResponse == nil)
}

// Test giving a device with a type that doesn't match the type in the store
func TestSet_BadDeviceType(t *testing.T) {
	server, mocks := setUpForGetSetTests(t)
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

	assert.ErrorContains(t, setError, "type given NotTheSameType does not match expected TestDevice")
	assert.Assert(t, setResponse == nil)
}
