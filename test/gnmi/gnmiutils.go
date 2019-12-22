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
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/utils"
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
	"time"
)

// DevicePath describes the results of a get operation for a single path
// It specifies the device, path, and value
type DevicePath struct {
	deviceName    string
	path          string
	pathDataType  string
	pathDataValue string
}

var noPaths = make([]DevicePath, 0)
var noExtensions = make([]*gnmi_ext.Extension, 0)

func convertGetResults(response *gpb.GetResponse) ([]DevicePath, []*gnmi_ext.Extension, error) {
	entryCount := len(response.Notification)
	result := make([]DevicePath, entryCount)

	for index, notification := range response.Notification {
		value := notification.Update[0].Val

		result[index].deviceName = notification.Update[0].Path.Target
		pathString := ""

		for _, elem := range notification.Update[0].Path.Elem {
			pathString = pathString + "/" + elem.Name
		}
		result[index].path = pathString

		result[index].pathDataType = "string_val"
		if value != nil {
			result[index].pathDataValue = utils.StrVal(value)
		} else {
			result[index].pathDataValue = ""
		}
	}

	return result, response.Extension, nil
}

func extractSetTransactionID(response *gpb.SetResponse) string {
	return string(response.Extension[0].GetRegisteredExt().Msg)
}

// gNMIGet generates a GET request on the given client for a path on a device
func gNMIGet(ctx context.Context, c client.Impl, paths []DevicePath) ([]DevicePath, []*gnmi_ext.Extension, error) {
	protoString := ""
	for _, devicePath := range paths {
		protoString = protoString + MakeProtoPath(devicePath.deviceName, devicePath.path)
	}

	getTZRequest := &gpb.GetRequest{}
	if err := proto.UnmarshalText(protoString, getTZRequest); err != nil {
		fmt.Printf("unable to parse gnmi.GetRequest from %q : %v", protoString, err)
		return nil, nil, err
	}

	response, err := c.(*gclient.Client).Get(ctx, getTZRequest)
	if err != nil || response == nil {
		return nil, nil, err
	}
	return convertGetResults(response)
}

// gNMISet generates a SET request on the given client for update and delete paths on a device
func gNMISet(ctx context.Context, c client.Impl, updatePaths []DevicePath, deletePaths []DevicePath, extensions []*gnmi_ext.Extension) (string, []*gnmi_ext.Extension, error) {
	var protoBuilder strings.Builder
	for _, updatePath := range updatePaths {
		protoBuilder.WriteString(MakeProtoUpdatePath(updatePath))
	}
	for _, deletePath := range deletePaths {
		protoBuilder.WriteString(MakeProtoDeletePath(deletePath.deviceName, deletePath.path))
	}

	setTZRequest := &gpb.SetRequest{}

	if err := proto.UnmarshalText(protoBuilder.String(), setTZRequest); err != nil {
		return "", nil, err
	}

	setTZRequest.Extension = extensions
	setResult, setError := c.(*gclient.Client).Set(ctx, setTZRequest)
	if setError != nil {
		return "", nil, setError
	}
	return extractSetTransactionID(setResult), setResult.Extension, nil
}

func getDevicePaths(devices []string, paths []string) []DevicePath {
	var devicePaths = make([]DevicePath, len(paths)*len(devices))
	pathIndex := 0
	for _, device := range devices {
		for _, path := range paths {
			devicePaths[pathIndex].deviceName = device
			devicePaths[pathIndex].path = path
			pathIndex++
		}
	}
	return devicePaths
}

func getDevicePathsWithValues(devices []string, paths []string, values []string) []DevicePath {
	var devicePaths = getDevicePaths(devices, paths)
	valueIndex := 0
	for range devices {
		for _, value := range values {
			devicePaths[valueIndex].pathDataValue = value
			devicePaths[valueIndex].pathDataType = StringVal
			valueIndex++
		}
	}
	return devicePaths
}

func checkDeviceValue(t *testing.T, deviceGnmiClient client.Impl, devicePaths []DevicePath, expectedValue string) {
	for i := 0; i < 30; i++ {
		deviceValues, extensions, deviceValuesError := gNMIGet(testutils.MakeContext(), deviceGnmiClient, devicePaths)
		if deviceValuesError == nil {
			assert.NoError(t, deviceValuesError, "GNMI get operation to device returned an error")
			assert.Equal(t, expectedValue, deviceValues[0].pathDataValue, "Query after set returned the wrong value: %s\n", expectedValue)
			assert.Equal(t, 0, len(extensions))
			return
		} else if status.Code(deviceValuesError) == codes.Unavailable {
			time.Sleep(1 * time.Second)
		} else {
			assert.Fail(t, "Failed to query device: %v", deviceValuesError)
		}
	}
	assert.Fail(t, "Failed to query device")
}

func getDeviceGNMIClient(t *testing.T, simulator env.SimulatorEnv) client.Impl {
	deviceGnmiClient, deviceGnmiClientError := simulator.NewGNMIClient()
	assert.NoError(t, deviceGnmiClientError)
	assert.True(t, deviceGnmiClient != nil, "Fetching device client returned nil")
	return deviceGnmiClient
}
