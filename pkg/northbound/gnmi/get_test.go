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
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"strings"
	"testing"
)

// See also the Test_getWithPrefixNoOtherPathsNoTarget below where the Target
// is in the Prefix
func Test_getNoTarget(t *testing.T) {
	server := setUp(false)

	noTargetPath1 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}
	noTargetPath2 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&noTargetPath1, &noTargetPath2},
	}

	_, err := server.Get(nil, &request)
	assert.ErrorContains(t, err, "has no target")
}

func Test_getWithPrefixNoOtherPathsNoTarget(t *testing.T) {
	server := setUp(false)

	prefixPath, err := utils.ParseGNMIElements([]string{"test1:cont1a", "cont2a"})

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	_, err = server.Get(nil, &request)
	assert.ErrorContains(t, err, "has no target")

}

// Test_getNoPathElems tests for  Paths with no elements - should treat it like /
func Test_getNoPathElems(t *testing.T) {
	server := setUp(false)

	noPath1 := gnmi.Path{Target: "Device1"}
	noPath2 := gnmi.Path{Target: "Device2"}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&noPath1, &noPath2},
	}

	result, err := server.Get(nil, &request)
	assert.NilError(t, err, "Unexpected error")

	assert.Equal(t, len(result.Notification), 1)

	assert.Equal(t, len(result.Notification[0].Update), 2)
}

// Test_getAllDevices is where a wildcard is used for target - path is ignored
func Test_getAllDevices(t *testing.T) {

	server := setUp(false)

	allDevicesPath := gnmi.Path{Elem: make([]*gnmi.PathElem, 0), Target: "*"}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&allDevicesPath},
	}

	result, err := server.Get(nil, &request)
	assert.NilError(t, err, "Unexpected error calling gNMI Get")

	assert.Equal(t, len(result.Notification), 1)
	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, result.Notification[0].Update[0].Path.Target, "*")

	deviceListStr := utils.StrVal(result.Notification[0].Update[0].Val)

	assert.Equal(t, deviceListStr,
		"[Device1, Device2, localhost:10161, localhost:10162, localhost:10163]")
}

// Test_getalldevices is where a wildcard is used for target - path is ignored
func Test_getAllDevicesInPrefix(t *testing.T) {

	server := setUp(false)

	request := gnmi.GetRequest{
		Prefix: &gnmi.Path{Target: "*"},
	}

	result, err := server.Get(nil, &request)
	assert.NilError(t, err, "Unexpected error calling gNMI Get")

	assert.Equal(t, len(result.Notification), 1, "Expected 1 notification")
	assert.Equal(t, len(result.Notification[0].Update), 1, "Expected 1 update")

	deviceListStr := utils.StrVal(result.Notification[0].Update[0].Val)

	assert.Equal(t, deviceListStr,
		"[Device1, Device2, localhost:10161, localhost:10162, localhost:10163]",
		"Expected value")
}

func Test_get2PathsWithPrefix(t *testing.T) {
	server := setUp(false)

	prefixPath, err := utils.ParseGNMIElements([]string{"test1:cont1a", "cont2a"})

	leafAPath, err := utils.ParseGNMIElements([]string{"leaf2a"})
	assert.NilError(t, err, "Unexpected error")
	leafAPath.Target = "Device1"

	leafBPath, err := utils.ParseGNMIElements([]string{"leaf2b"})
	assert.NilError(t, err, "Unexpected error")
	leafBPath.Target = "Device1"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
		Path:   []*gnmi.Path{leafAPath, leafBPath},
	}

	result, err := server.Get(nil, &request)
	assert.NilError(t, err, "Unexpected error")

	assert.Equal(t, len(result.Notification), 1)

	assert.Equal(t, len(result.Notification[0].Update), 2)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/test1:cont1a/cont2a")
	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/leaf2a")
	assert.Equal(t, utils.StrVal(result.Notification[0].Update[0].Val), "13")
	assert.Equal(t, utils.StrPath(result.Notification[0].Update[1].Path),
		"/leaf2b")
	assert.Equal(t, utils.StrVal(result.Notification[0].Update[1].Val), "3.14159")
}

func Test_getWithPrefixNoOtherPaths(t *testing.T) {
	server := setUp(false)

	prefixPath, err := utils.ParseGNMIElements([]string{"test1:cont1a", "cont2a"})
	prefixPath.Target = "Device1"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	result, err := server.Get(nil, &request)
	assert.NilError(t, err, "Unexpected error")

	assert.Equal(t, len(result.Notification), 1)

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/test1:cont1a/cont2a")

	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/")
	assert.Check(t, strings.Contains(utils.StrVal(result.Notification[0].Update[0].Val),
		`"leaf2b": "3.14159"`))
}

func Test_targetDoesNotExist(t *testing.T) {
	server := setUp(false)

	prefixPath, err := utils.ParseGNMIElements([]string{"test1:cont1a", "cont2a"})
	assert.NilError(t, err, "Unexpected error")
	prefixPath.Target = "Device3"

	request := gnmi.GetRequest{
		Prefix: prefixPath,
	}

	_, err = server.Get(nil, &request)
	assert.ErrorContains(t, err, "No Configuration found for")
}

// Target does exist, but specified path does not
// No error - just an empty value
func Test_pathDoesNotExist(t *testing.T) {
	server := setUp(false)

	prefixPath, err := utils.ParseGNMIElements([]string{"test1:cont1a", "cont2a"})
	assert.NilError(t, err, "Unexpected error")
	prefixPath.Target = "Device1"
	path, err := utils.ParseGNMIElements([]string{"leaf2w"})
	assert.NilError(t, err, "Unexpected error")

	request := gnmi.GetRequest{
		Prefix: prefixPath,
		Path:   []*gnmi.Path{path},
	}

	result, err := server.Get(nil, &request)
	assert.NilError(t, err, "Unexpected error")

	assert.Equal(t, len(result.Notification), 1)

	assert.Equal(t, len(result.Notification[0].Update), 1)

	assert.Equal(t, utils.StrPath(result.Notification[0].Prefix),
		"/test1:cont1a/cont2a")

	assert.Equal(t, utils.StrPath(result.Notification[0].Update[0].Path),
		"/leaf2w")
	assert.Assert(t, result.Notification[0].Update[0].Val == nil)
}
