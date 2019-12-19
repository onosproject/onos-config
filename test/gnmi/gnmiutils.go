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
	"github.com/onosproject/onos-config/api/diags"
	"github.com/onosproject/onos-config/api/types/change"
	"github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
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

// MakeContext returns a new context for use in GNMI requests
func MakeContext() context.Context {
	// TODO: Investigate using context.WithCancel() here
	ctx := context.Background()
	return ctx
}

// IsNetworkChangeComplete checks for a COMPLETED status on the given change, retying for 15 seconds
func IsNetworkChangeComplete(t *testing.T, networkChangeID network.ID) bool {
	listNetworkChangeRequest := &diags.ListNetworkChangeRequest{
		Subscribe:     false,
		ChangeID:      networkChangeID,
		WithoutReplay: false,
	}

	changeServiceClient, changeServiceClientErr := env.Config().NewChangeServiceClient()
	assert.Nil(t, changeServiceClientErr)
	assert.True(t, changeServiceClient != nil)

	for i := 1; i < 5; i++ {
		// Check if the network change has completed

		listNetworkChangesClient, listNetworkChangesClientErr := changeServiceClient.ListNetworkChanges(context.Background(), listNetworkChangeRequest)
		assert.Nil(t, listNetworkChangesClientErr)
		assert.True(t, listNetworkChangesClient != nil)
		networkChangeResponse, networkChangeResponseErr := listNetworkChangesClient.Recv()
		assert.Nil(t, networkChangeResponseErr)
		assert.True(t, networkChangeResponse != nil)
		if change.State_COMPLETE == networkChangeResponse.Change.Status.State {
			return true
		}

		time.Sleep(2 * time.Second)
	}
	return false
}
