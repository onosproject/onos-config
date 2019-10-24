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
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"

	"gotest.tools/assert"
	"io"
	"strings"
	"testing"
	"time"
)

var deviceChanges []devicechangetypes.DeviceChange
var networkChanges []networkchangetypes.NetworkChange

func generateDeviceChangeData(count int) {
	deviceChanges = make([]devicechangetypes.DeviceChange, count)
	now := time.Now()
	for cfgIdx := range deviceChanges {
		deviceID := fmt.Sprintf("device-%d", cfgIdx)

		deviceChanges[cfgIdx] = devicechangetypes.DeviceChange{
			ID:       devicechangetypes.ID(deviceID),
			Index:    devicechangetypes.Index(cfgIdx),
			Revision: 0,
			NetworkChange: devicechangetypes.NetworkChangeRef{
				ID:    "network-1",
				Index: 0,
			},
			Change: &devicechangetypes.Change{
				DeviceID:      devicetopo.ID(deviceID),
				DeviceVersion: "1.0.0",
				Values: []*devicechangetypes.ChangeValue{
					{
						Path:    "/aa/bb/cc",
						Value:   devicechangetypes.NewTypedValueString("Test1"),
						Removed: false,
					},
					{
						Path:    "/aa/bb/dd",
						Value:   devicechangetypes.NewTypedValueString("Test2"),
						Removed: false,
					},
				},
			},
			Status: change.Status{
				Phase:   change.Phase(cfgIdx % 2),
				State:   change.State(cfgIdx % 4),
				Reason:  change.Reason(cfgIdx % 2),
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
	CaptureOutput(outputBuffer)
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
	CaptureOutput(outputBuffer)
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
