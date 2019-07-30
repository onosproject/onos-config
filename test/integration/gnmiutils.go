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

package integration

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
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

func convertGetResults(response *gpb.GetResponse) ([]DevicePath, error) {
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

	return result, nil
}

func extractSetTransactionID(response *gpb.SetResponse) string {
	return string(response.Extension[0].GetRegisteredExt().Msg)
}

// GNMIGet generates a GET request on the given client for a path on a device
func GNMIGet(ctx context.Context, c client.Impl, paths []DevicePath) ([]DevicePath, error) {
	protoString := ""
	for _, devicePath := range paths {
		protoString = protoString + MakeProtoPath(devicePath.deviceName, devicePath.path)
	}

	getTZRequest := &gpb.GetRequest{}
	if err := proto.UnmarshalText(protoString, getTZRequest); err != nil {
		fmt.Printf("unable to parse gnmi.GetRequest from %q : %v", protoString, err)
		return nil, err
	}

	response, err := c.(*gclient.Client).Get(ctx, getTZRequest)
	if err != nil || response == nil {
		return nil, err
	}

	return convertGetResults(response)
}

// GNMISet generates a SET request on the given client for a path on a device
func GNMISet(ctx context.Context, c client.Impl, devicePaths []DevicePath) (string, error) {
	var protoBuilder strings.Builder
	for _, devicePath := range devicePaths {
		protoBuilder.WriteString(MakeProtoUpdatePath(devicePath))
	}

	setTZRequest := &gpb.SetRequest{}
	if err := proto.UnmarshalText(protoBuilder.String(), setTZRequest); err != nil {
		return "", err
	}

	setResult, setError := c.(*gclient.Client).Set(ctx, setTZRequest)
	if setError != nil {
		return "", setError
	}
	return extractSetTransactionID(setResult), nil
}

// GNMIDelete generates a SET request on the given client to delete a path for a device
func GNMIDelete(ctx context.Context, c client.Impl, devicePaths []DevicePath) error {
	var protoBuilder strings.Builder
	for _, devicePath := range devicePaths {
		protoBuilder.WriteString(MakeProtoDeletePath(devicePath.deviceName, devicePath.path))
	}

	setTZRequest := &gpb.SetRequest{}
	if err := proto.UnmarshalText(protoBuilder.String(), setTZRequest); err != nil {
		return err
	}

	_, err := c.(*gclient.Client).Set(ctx, setTZRequest)
	return err
}

// MakeContext returns a new context for use in GNMI requests
func MakeContext() context.Context {
	// TODO: Investigate using context.WithCancel() here
	ctx := context.Background()
	return ctx
}
