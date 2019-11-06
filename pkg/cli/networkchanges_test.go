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

// Unit tests for watch and list network-changes CLI
package cli

import (
	"bytes"
	"fmt"
	"github.com/onosproject/onos-config/api/diags"
	"github.com/onosproject/onos-config/api/types/change"
	"github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/api/types/change/network"
	"gotest.tools/assert"
	"io"
	"strings"
	"testing"
	"time"
)

func generateNetworkChangeData(count int) {
	networkChanges = make([]network.NetworkChange, count)
	now := time.Now()

	for cfgIdx := range networkChanges {
		networkID := fmt.Sprintf("a_new_network_change-%d", cfgIdx)

		networkChanges[cfgIdx] = network.NetworkChange{
			ID:       network.ID(networkID),
			Index:    network.Index(cfgIdx),
			Revision: 0,
			Status: change.Status{
				Phase:   change.Phase(cfgIdx % 2),
				State:   change.State(cfgIdx % 4),
				Reason:  change.Reason(cfgIdx % 2),
				Message: "Test",
			},
			Created: now,
			Updated: now,
			Changes: []*device.Change{
				{
					DeviceID:      "device-1",
					DeviceVersion: "1.0.0",
					Values: []*device.ChangeValue{
						{
							Path:    "/aa/bb/cc",
							Value:   device.NewTypedValueString("Test1"),
							Removed: false,
						},
						{
							Path:    "/aa/bb/dd",
							Value:   device.NewTypedValueString("Test2"),
							Removed: false,
						},
					},
				},
				{
					DeviceID:      "device-2",
					DeviceVersion: "1.0.0",
					Values: []*device.ChangeValue{
						{
							Path:    "/aa/bb/cc",
							Value:   device.NewTypedValueString("Test3"),
							Removed: false,
						},
						{
							Path:    "/aa/bb/dd",
							Value:   device.NewTypedValueString("Test4"),
							Removed: false,
						},
					},
				},
			},
			Refs: []*network.DeviceChangeRef{
				{DeviceChangeID: "device-1:1"},
				{DeviceChangeID: "device-2:1"},
			},
			Deleted: false,
		}
	}
}

func Test_WatchNetworkChanges(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)
	generateDeviceChangeData(4)
	generateNetworkChangeData(4)

	configsClient := MockChangeServiceListNetworkChangesClient{
		recvFn: recvWatchNetworkChangesMock,
	}

	setUpMockClients(MockClientsConfig{
		listNetworkChangesClient: &configsClient,
	})

	networkChangesCmd := getWatchNetworkChangesCommand()
	err := networkChangesCmd.RunE(networkChangesCmd, nil)
	assert.NilError(t, err)
	output := outputBuffer.String()
	assert.Equal(t, strings.Count(output, "a_new_network_change"), len(networkChanges))
}

var nextWatchNwChIndex int

func recvWatchNetworkChangesMock() (*diags.ListNetworkChangeResponse, error) {
	if nextWatchNwChIndex < len(networkChanges) {
		netw := networkChanges[nextWatchNwChIndex]
		nextWatchNwChIndex++

		return &diags.ListNetworkChangeResponse{
			Change: &netw,
		}, nil
	}
	return nil, io.EOF
}

func Test_GetNetworkChanges(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)
	generateDeviceChangeData(4)
	generateNetworkChangeData(4)
	var nextNwChIndex = 0

	configsClient := MockChangeServiceListNetworkChangesClient{
		recvFn:      recvListNetworkChangesMock,
		recvCounter: func() int { return nextNwChIndex },
	}

	setUpMockClients(MockClientsConfig{
		listNetworkChangesClient: &configsClient,
	})

	networkChangesCmd := getListNetworkChangesCommand()
	err := networkChangesCmd.RunE(networkChangesCmd, nil)
	assert.NilError(t, err)
	output := outputBuffer.String()
	assert.Equal(t, strings.Count(output, "a_new_network_change"), len(networkChanges))

}

var nextListNwChIndex int

func recvListNetworkChangesMock() (*diags.ListNetworkChangeResponse, error) {
	if nextListNwChIndex < len(networkChanges) {
		netw := networkChanges[nextListNwChIndex]
		nextListNwChIndex++

		return &diags.ListNetworkChangeResponse{
			Change: &netw,
		}, nil
	}
	return nil, io.EOF
}
