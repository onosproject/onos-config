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

// Unit tests for watch and list device-changes CLI
package cli

import (
	"bytes"
	"fmt"
	"github.com/onosproject/onos-config/api/diags"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-lib-go/pkg/cli"
	"gotest.tools/assert"
	"io"
	"strings"
	"testing"
	"time"
)

var deviceChanges []devicechange.DeviceChange
var networkChanges []networkchange.NetworkChange

func generateDeviceChangeData(count int) {
	deviceChanges = make([]devicechange.DeviceChange, count)
	now := time.Now()
	for cfgIdx := range deviceChanges {
		deviceID := fmt.Sprintf("device-%d", cfgIdx)

		deviceChanges[cfgIdx] = devicechange.DeviceChange{
			ID:       devicechange.ID(deviceID),
			Index:    devicechange.Index(cfgIdx),
			Revision: 0,
			NetworkChange: devicechange.NetworkChangeRef{
				ID:    "network-1",
				Index: 0,
			},
			Change: &devicechange.Change{
				DeviceID:      device.ID(deviceID),
				DeviceVersion: "1.0.0",
				Values: []*devicechange.ChangeValue{
					{
						Path:    "/aa/bb/cc",
						Value:   devicechange.NewTypedValueString("Test1"),
						Removed: false,
					},
					{
						Path:    "/aa/bb/dd",
						Value:   devicechange.NewTypedValueString("Test2"),
						Removed: false,
					},
				},
			},
			Status: changetypes.Status{
				Phase:   changetypes.Phase(cfgIdx % 2),
				State:   changetypes.State(cfgIdx % 4),
				Reason:  changetypes.Reason(cfgIdx % 2),
				Message: "Test",
			},
			Created: now,
			Updated: now,
		}
	}
}

var nextDevChIndex = 0

func recvDeviceChangesMock() (*diags.ListDeviceChangeResponse, error) {
	if nextDevChIndex < len(deviceChanges) {
		netw := deviceChanges[nextDevChIndex]
		nextDevChIndex++

		return &diags.ListDeviceChangeResponse{
			Change: &netw,
		}, nil
	}
	return nil, io.EOF
}

func Test_WatchDeviceChanges(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	cli.CaptureOutput(outputBuffer)
	generateDeviceChangeData(4)
	generateNetworkChangeData(4)

	configsClient := MockChangeServiceListDeviceChangesClient{
		recvFn: recvDeviceChangesMock,
	}

	setUpMockClients(MockClientsConfig{
		listDeviceChangesClient: &configsClient,
	})

	deviceChangesCmd := getWatchDeviceChangesCommand()
	err := deviceChangesCmd.RunE(deviceChangesCmd, []string{"device-1"})
	assert.NilError(t, err)
	output := outputBuffer.String()
	assert.Equal(t, strings.Count(output, "Test2"), len(deviceChanges))

}

var nextListDevChIndex = 0

func recvListDeviceChangesMock() (*diags.ListDeviceChangeResponse, error) {
	if nextListDevChIndex < len(deviceChanges) {
		netw := deviceChanges[nextListDevChIndex]
		nextListDevChIndex++

		return &diags.ListDeviceChangeResponse{
			Change: &netw,
		}, nil
	}
	return nil, io.EOF
}

func Test_ListDeviceChanges(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	cli.CaptureOutput(outputBuffer)
	generateDeviceChangeData(4)
	generateNetworkChangeData(4)

	configsClient := MockChangeServiceListDeviceChangesClient{
		recvFn: recvListDeviceChangesMock,
	}

	setUpMockClients(MockClientsConfig{
		listDeviceChangesClient: &configsClient,
	})

	deviceChangesCmd := getWatchDeviceChangesCommand()
	err := deviceChangesCmd.RunE(deviceChangesCmd, []string{"device-1"})
	assert.NilError(t, err)
	output := outputBuffer.String()
	assert.Equal(t, strings.Count(output, "Test2"), len(deviceChanges))

}
