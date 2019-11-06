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
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/assert"
	"strconv"
	"testing"
)

// See also the Test_getWithPrefixNoOtherPathsNoTarget below where the Target
// is in the Prefix
func Test_getNoTarget(t *testing.T) {
	server, _, mocks := setUp(t)
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found"))

	noTargetPath1 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}
	noTargetPath2 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&noTargetPath1, &noTargetPath2},
	}

	_, err := server.Get(context.TODO(), &request)
	assert.ErrorContains(t, err, "has no target")
}

func Test_getWithPrefixNoOtherPathsNoTarget(t *testing.T) {
	server, _, mocks := setUp(t)
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found"))

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NilError(t, err)

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	_, err = server.Get(context.TODO(), &request)
	assert.ErrorContains(t, err, "has no target")

}

// Test_getNoPathElems tests for  Paths with no elements - should treat it like /
func Test_getNoPathElems(t *testing.T) {
	server, _, mocks := setUp(t)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(gomock.Any()).Return([]*device.Info{
		{
			DeviceID: "Device1",
			Version:  "1.0.0",
			Type:     "TestDevice",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).AnyTimes()
	setUpListMock(mocks)

	noPath1 := gnmi.Path{Target: "Device1"}
	noPath2 := gnmi.Path{Target: "Device2"}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&noPath1, &noPath2},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NilError(t, err)

	assert.Equal(t, len(result.Notification), 2)

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, len(result.Notification[1].Update), 1)
}

// Test_getAllDevices is where a wildcard is used for target - path is ignored
func Test_getAllDevices(t *testing.T) {
	server, _, _, _, mocks := setUpForGetSetTests(t)
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).AnyTimes()

	allDevicesPath := gnmi.Path{Elem: make([]*gnmi.PathElem, 0), Target: "*"}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&allDevicesPath},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NilError(t, err)

	assert.Equal(t, len(result.Notification), 1)
	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, result.Notification[0].Update[0].Path.Target, "*")

	deviceListStr := utils.StrVal(result.Notification[0].Update[0].Val)

	assert.Equal(t, deviceListStr, "[Device1 (1.0.0), Device2 (2.0.0), Device3]")
}

// Test_getalldevices is where a wildcard is used for target - path is ignored
func Test_getAllDevicesInPrefix(t *testing.T) {
	server, _, _, _, mocks := setUpForGetSetTests(t)
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).AnyTimes()

	request := gnmi.GetRequest{
		Prefix: &gnmi.Path{Target: "*"},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NilError(t, err)

	assert.Equal(t, len(result.Notification), 1)
	assert.Equal(t, len(result.Notification[0].Update), 1)

	deviceListStr := utils.StrVal(result.Notification[0].Update[0].Val)

	assert.Equal(t, deviceListStr, "[Device1 (1.0.0), Device2 (2.0.0), Device3]", "Expected value")
}

func Test_get2PathsWithPrefix(t *testing.T) {
	server, _, mocks := setUp(t)
	setUpChangesMock(mocks)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*device.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(4)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NilError(t, err)

	leafAPath, err := utils.ParseGNMIElements([]string{"leaf2a"})
	assert.NilError(t, err)
	leafAPath.Target = "Device1"

	leafBPath, err := utils.ParseGNMIElements([]string{"leaf2b"})
	assert.NilError(t, err)
	leafBPath.Target = "Device1"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
		Path:   []*gnmi.Path{leafAPath, leafBPath},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NilError(t, err)

	assert.Equal(t, len(result.Notification), 2)

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/cont1a/cont2a")
	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/leaf2a")
	assert.Equal(t, result.Notification[0].Update[0].GetVal().GetUintVal(), uint64(13))

	assert.Equal(t, utils.StrPath(result.Notification[1].Update[0].Path),
		"/leaf2b")

	assert.Equal(t, result.Extension[0].GetRegisteredExt().Id.String(), strconv.Itoa(GnmiExtensionDevicesNotConnected))
	assert.Equal(t, string(result.Extension[0].GetRegisteredExt().Msg), "Device1")
}

func Test_getWithPrefixNoOtherPaths(t *testing.T) {
	server, _, mocks := setUp(t)
	setUpChangesMock(mocks)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*device.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(2)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NilError(t, err)
	prefixPath.Target = "Device1"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NilError(t, err)

	assert.Equal(t, len(result.Notification), 1)

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/cont1a/cont2a")

	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/")
	val := result.Notification[0].Update[0].GetVal().GetUintVal()
	assert.Equal(t, val, uint64(13), "Got")
}

func Test_targetDoesNotExist(t *testing.T) {
	server, _, mocks := setUp(t)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device3")).Return([]*device.Info{
		{
			DeviceID: "Device3",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).AnyTimes()
	setUpListMock(mocks)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NilError(t, err)
	prefixPath.Target = "Device3"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NilError(t, err, "get should not return an error")
	assert.Assert(t, result != nil)
	assert.Assert(t, result.Notification[0].Update[0].Val == nil)
	extension := result.Extension[0].GetRegisteredExt()
	assert.Equal(t, extension.Id.String(), strconv.Itoa(GnmiExtensionDevicesNotConnected), "extension 103 is not specified")
}

// Target does exist, but specified path does not
// No error - just an empty value
func Test_pathDoesNotExist(t *testing.T) {
	server, _, mocks := setUp(t)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*device.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(2)
	setUpListMock(mocks)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NilError(t, err)
	prefixPath.Target = "Device1"
	path, err := utils.ParseGNMIElements([]string{"leaf2w"})
	assert.NilError(t, err)

	request := gnmi.GetRequest{
		Prefix: prefixPath,
		Path:   []*gnmi.Path{path},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NilError(t, err)

	assert.Equal(t, len(result.Notification), 1)

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/cont1a/cont2a")

	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/leaf2w")
	assert.Assert(t, result.Notification[0].Update[0].Val == nil)
}
