// Copyright 2020-present Open Networking Foundation.
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
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"strings"
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

// getGNMIValue generates a GET request on the given client for a path on a device
func getGNMIValue(ctx context.Context, c client.Impl, paths []DevicePath) ([]DevicePath, []*gnmi_ext.Extension, error) {
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

// setGNMIValue generates a SET request on the given client for update and delete paths on a device
func setGNMIValue(ctx context.Context, c client.Impl, updatePaths []DevicePath, deletePaths []DevicePath, extensions []*gnmi_ext.Extension) (string, []*gnmi_ext.Extension, error) {
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

func getDevicePath(device string, path string) []DevicePath {
	return getDevicePathWithValue(device, path, "", "")
}

func getDevicePathWithValue(device string, path string, value string, valueType string) []DevicePath {
	devicePath := make([]DevicePath, 1)
	devicePath[0].deviceName = device
	devicePath[0].path = path
	devicePath[0].pathDataValue = value
	devicePath[0].pathDataType = valueType
	return devicePath
}
