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
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "has no target")
}

func Test_getWithPrefixNoOtherPathsNoTarget(t *testing.T) {
	server, _, mocks := setUp(t)
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found"))

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NoError(t, err)

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	_, err = server.Get(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "has no target")

}

// Test_getNoPathElemsProto tests for  Paths with no elements - should treat it like /
func Test_getNoPathElemsJSON(t *testing.T) {
	server, _, mocks := setUp(t)
	setUpChangesMock(mocks)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*cache.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(4)

	noPath1 := gnmi.Path{Target: "Device1"}

	request := gnmi.GetRequest{
		Path:     []*gnmi.Path{&noPath1},
		Encoding: gnmi.Encoding_JSON,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 1)
	assert.Equal(t, len(result.Notification[0].Update), 1)
	assert.Equal(t, utils.StrPathElem(result.Notification[0].Update[0].Path.Elem), "")
	assert.Equal(t, `{
  "cont1a": {
    "cont2a": {
      "leaf2a": 13,
      "leaf2b": "1.4567"
    },
    "list2a": [
      {
        "name": "first",
        "tx-power": 19
      }
    ],
    "list4": [
      {
        "id": "first",
        "leaf4b": "initial value",
        "list4a": [
          {
            "displayname": "this is a list",
            "fkey1": "abc",
            "fkey2": 8
          }
        ]
      }
    ],
    "list5": [
      {
        "key1": "abc",
        "key2": 8,
        "leaf5a": "Leaf 5a"
      }
    ]
  }
}`, utils.StrVal(result.Notification[0].Update[0].Val))

}

// Test_getNoPathElemsProto tests for  Paths with no elements - should treat it like /
func Test_getNoPathElemsProto(t *testing.T) {
	server, _, mocks := setUp(t)
	setUpChangesMock(mocks)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*cache.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(4)

	noPath1 := gnmi.Path{Target: "Device1"}

	request := gnmi.GetRequest{
		Path:     []*gnmi.Path{&noPath1},
		Encoding: gnmi.Encoding_PROTO,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 1)
	assert.Equal(t, len(result.Notification[0].Update), 12)
	for _, upd := range result.Notification[0].Update {
		switch path := utils.StrPathElem(upd.GetPath().GetElem()); path {
		case "/cont1a/cont2a/leaf2a":
			assert.Equal(t, utils.StrVal(upd.Val), "13")
		case "/cont1a/cont2a/leaf2b":
			assert.Equal(t, utils.StrVal(upd.Val), "1.4567")
		case "/cont1a/list2a[name=first]/name":
			assert.Equal(t, utils.StrVal(upd.Val), "first")
		case "/cont1a/list2a[name=first]/tx-power":
			assert.Equal(t, utils.StrVal(upd.Val), "19")
		case "/cont1a/list4[id=first]/id":
			assert.Equal(t, utils.StrVal(upd.Val), "first")
		case "/cont1a/list4[id=first]/leaf4b":
			assert.Equal(t, utils.StrVal(upd.Val), "initial value")
		case "/cont1a/list5[key1=abc][key2=8]/key1":
			assert.Equal(t, utils.StrVal(upd.Val), "abc")
		case "/cont1a/list5[key1=abc][key2=8]/key2":
			assert.Equal(t, utils.StrVal(upd.Val), "8")
		case "/cont1a/list5[key1=abc][key2=8]/leaf5a":
			assert.Equal(t, utils.StrVal(upd.Val), "Leaf 5a")
		case "/cont1a/list4[id=first]/list4a[fkey1=abc][fkey2=8]/fkey1":
			assert.Equal(t, utils.StrVal(upd.Val), "abc")
		case "/cont1a/list4[id=first]/list4a[fkey1=abc][fkey2=8]/fkey2":
			assert.Equal(t, utils.StrVal(upd.Val), "8")
		case "/cont1a/list4[id=first]/list4a[fkey1=abc][fkey2=8]/displayname":
			assert.Equal(t, utils.StrVal(upd.Val), "this is a list")
		default:
			t.Errorf("unexpected elem %s", path)
		}
	}
}

// Test_getAllDevices is where a wildcard is used for target - path is ignored
func Test_getAllDevices(t *testing.T) {
	server, _, _ := setUpForGetSetTests(t)

	allDevicesPath := gnmi.Path{Elem: make([]*gnmi.PathElem, 0), Target: "*"}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&allDevicesPath},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 1)
	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, result.Notification[0].Update[0].Path.Target, "*")

	deviceListStr := utils.StrVal(result.Notification[0].Update[0].Val)
	assert.Contains(t, deviceListStr, `{
  "targets": [
    "Device1",
    "Device2",
    "Device3"
  ]
}`)
}

// Test_getalldevices is where a wildcard is used for target - path is ignored
func Test_getAllDevicesInPrefixJSON(t *testing.T) {
	server, _, _ := setUpForGetSetTests(t)

	request := gnmi.GetRequest{
		Prefix: &gnmi.Path{Target: "*"},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 1)
	assert.Equal(t, len(result.Notification[0].Update), 1)

	deviceListStr := utils.StrVal(result.Notification[0].Update[0].Val)

	assert.Contains(t, deviceListStr, `{
  "targets": [
    "Device1",
    "Device2",
    "Device3"
  ]
}`)
}

// Test_getalldevices is where a wildcard is used for target - path is ignored
func Test_getAllDevicesInPrefixPROTO(t *testing.T) {
	server, _, _ := setUpForGetSetTests(t)

	request := gnmi.GetRequest{
		Prefix:   &gnmi.Path{Target: "*"},
		Encoding: gnmi.Encoding_PROTO,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 1)
	assert.Equal(t, len(result.Notification[0].Update), 1)

	deviceListStr := utils.StrVal(result.Notification[0].Update[0].Val)

	assert.Contains(t, deviceListStr, "Device1, Device2, Device3")
}

func Test_get2PathsWithPrefixJSON(t *testing.T) {
	server, _, mocks := setUp(t)
	setUpChangesMock(mocks)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*cache.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(4)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NoError(t, err)

	leafAPath, err := utils.ParseGNMIElements([]string{"leaf2a"})
	assert.NoError(t, err)
	leafAPath.Target = "Device1"

	leafBPath, err := utils.ParseGNMIElements([]string{"leaf2b"})
	assert.NoError(t, err)
	leafBPath.Target = "Device1"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
		Path:   []*gnmi.Path{leafAPath, leafBPath},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 2) // TODO - since these share the same device they could be 1 notification

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/cont1a/cont2a")
	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/leaf2a")
	assert.Equal(t, utils.StrVal(result.Notification[0].Update[0].GetVal()),
		`{
  "cont1a": {
    "cont2a": {
      "leaf2a": 13
    }
  }
}`)

	assert.Equal(t, utils.StrPath(result.Notification[1].Update[0].Path),
		"/leaf2b")

	assert.Equal(t, utils.StrVal(result.Notification[1].Update[0].GetVal()),
		`{
  "cont1a": {
    "cont2a": {
      "leaf2b": "1.4567"
    }
  }
}`)
}

func Test_get2PathsWithPrefixProto(t *testing.T) {
	server, _, mocks := setUp(t)
	setUpChangesMock(mocks)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*cache.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(4)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NoError(t, err)

	leafAPath, err := utils.ParseGNMIElements([]string{"leaf2a"})
	assert.NoError(t, err)
	leafAPath.Target = "Device1"

	leafBPath, err := utils.ParseGNMIElements([]string{"leaf2b"})
	assert.NoError(t, err)
	leafBPath.Target = "Device1"

	request := gnmi.GetRequest{
		Prefix:   prefixPath,
		Path:     []*gnmi.Path{leafAPath, leafBPath},
		Encoding: gnmi.Encoding_PROTO,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 2) // TODO - since these share the same device they could be 1 notification

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/cont1a/cont2a")
	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path), "/leaf2a")
	assert.Equal(t, utils.StrVal(result.Notification[0].Update[0].GetVal()), "13")
	assert.Equal(t, utils.StrPath(result.Notification[1].Update[0].Path), "/leaf2b")
	assert.Equal(t, utils.StrVal(result.Notification[1].Update[0].GetVal()), "1.4567")
}

func Test_getWithPrefixNoOtherPaths(t *testing.T) {
	server, _, mocks := setUp(t)
	setUpChangesMock(mocks)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*cache.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(2)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NoError(t, err)
	prefixPath.Target = "Device1"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 1)

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/cont1a/cont2a")

	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/")
	val := utils.StrVal(result.Notification[0].Update[0].GetVal())
	assert.Equal(t, val, `{
  "cont1a": {
    "cont2a": {
      "leaf2a": 13,
      "leaf2b": "1.4567"
    }
  }
}`, "Got JSON value")

	requestProto := gnmi.GetRequest{Prefix: prefixPath, Encoding: gnmi.Encoding_PROTO}
	resultProto, err := server.Get(context.TODO(), &requestProto)

	assert.NoError(t, err)

	assert.Equal(t, len(resultProto.Notification), 1)

	assert.Equal(t, len(resultProto.Notification[0].Update), 2)

	assert.Equal(t, utils.StrPath(resultProto.Notification[0].Prefix),
		"/cont1a/cont2a")

	assert.Equal(t, utils.StrPath(resultProto.Notification[0].Update[0].Path), "/leaf2a")
	valProto := resultProto.Notification[0].Update[0].GetVal().GetUintVal()
	assert.Equal(t, valProto, uint64(13), "Got PROTO value")
	assert.Equal(t, utils.StrPath(resultProto.Notification[0].Update[1].Path), "/leaf2b")
	valProto2 := utils.StrVal(resultProto.Notification[0].Update[1].GetVal())
	assert.Equal(t, valProto2, "1.4567", "Got PROTO value")
}

func Test_targetDoesNotExist(t *testing.T) {
	server, _, mocks := setUp(t)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device3")).Return([]*cache.Info{
		{
			DeviceID: "Device3",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).AnyTimes()
	mocks.MockStores.DeviceStateStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]*devicechange.PathValue{}, nil).AnyTimes()
	setUpListMock(mocks)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NoError(t, err)
	prefixPath.Target = "Device3"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	md := make(map[string]string) // Try passing in some metadata with context - usually done by AuthInterceptor
	md["name"] = "test user"
	md["email"] = "test email"
	result, err := server.Get(metadata.NewIncomingContext(context.TODO(), metadata.New(md)), &request)
	assert.NoError(t, err, "get should not return an error")
	assert.NotNil(t, result)
	assert.Nil(t, result.Notification[0].Update[0].Val)
}

// Target does exist, but specified path does not
// No error - just an empty value
func Test_pathDoesNotExist(t *testing.T) {
	server, _, mocks := setUp(t)
	mocks.MockDeviceCache.EXPECT().GetDevicesByID(devicetype.ID("Device1")).Return([]*cache.Info{
		{
			DeviceID: "Device1",
			Type:     "Devicesim",
			Version:  "1.0.0",
		},
	}).AnyTimes()
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).Times(2)
	mocks.MockStores.DeviceStateStore.EXPECT().Get(gomock.Any(), gomock.Any()).Return([]*devicechange.PathValue{}, nil).AnyTimes()
	setUpListMock(mocks)

	prefixPath, err := utils.ParseGNMIElements([]string{"cont1a", "cont2a"})
	assert.NoError(t, err)
	prefixPath.Target = "Device1"
	path, err := utils.ParseGNMIElements([]string{"leaf2w"})
	assert.NoError(t, err)

	request := gnmi.GetRequest{
		Prefix: prefixPath,
		Path:   []*gnmi.Path{path},
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)

	assert.Equal(t, len(result.Notification), 1)

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/cont1a/cont2a")

	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/leaf2w")
	assert.Nil(t, result.Notification[0].Update[0].Val)
}
