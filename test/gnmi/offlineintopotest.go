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
//

package gnmi

import (
	"context"
	"github.com/onosproject/onos-config/api/diags"
	"github.com/onosproject/onos-config/api/types/change"
	"github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

const (
	offlineInTopoModPath          = "/system/clock/config/timezone-name"
	offlineInTopoModValue         = "Europe/Rome"
	offlineInTopoModDeviceName    = "offline-dev-1"
	offlineInTopoModDeviceVersion = "1.0.0"
	offlineInTopoModDeviceType    = "Devicesim"
)

// TestOfflineDeviceInTopo tests set/query of a single GNMI path to a single device that is in the config but offline
func (s *TestSuite) TestOfflineDeviceInTopo(t *testing.T) {
	deviceClient, deviceClientError := env.Topo().NewDeviceServiceClient()
	assert.NotNil(t, deviceClient)
	assert.Nil(t, deviceClientError)
	newDevice := &device.Device{
		ID:          offlineInTopoModDeviceName,
		Revision:    0,
		Address:     offlineInTopoModDeviceName + ":10161",
		Target:      "",
		Version:     offlineInTopoModDeviceVersion,
		Timeout:     nil,
		Credentials: device.Credentials{},
		TLS:         device.TlsConfig{},
		Type:        offlineInTopoModDeviceType,
		Role:        "",
		Attributes:  nil,
		Protocols:   nil,
	}
	addRequest := &device.AddRequest{Device: newDevice}
	addResponse, addResponseError := deviceClient.Add(context.Background(), addRequest)
	assert.NotNil(t, addResponse)
	assert.Nil(t, addResponseError)

	// Make a GNMI client to use for requests
	c, err := env.Config().NewGNMIClient()
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Set a value using gNMI client to the offline device

	setPath := makeDevicePath(offlineInTopoModDeviceName, offlineInTopoModPath)
	setPath[0].pathDataValue = offlineInTopoModValue
	setPath[0].pathDataType = StringVal

	_, extensionsSet, errorSet := gNMISet(MakeContext(), c, setPath, noPaths, noExtensions)
	assert.NoError(t, errorSet)
	assert.Equal(t, 1, len(extensionsSet))
	extensionBefore := extensionsSet[0].GetRegisteredExt()
	assert.Equal(t, extensionBefore.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))
	networkChangeID := network.ID(extensionBefore.Msg)

	// Check that the value was set correctly
	valueAfter, extensions, errorAfter := gNMIGet(MakeContext(), c, makeDevicePath(offlineInTopoModDeviceName, offlineInTopoModPath))
	assert.NoError(t, errorAfter)
	assert.Equal(t, 0, len(extensions))
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, offlineInTopoModValue, valueAfter[0].pathDataValue, "Query after set returned the wrong value: %s\n", valueAfter)

	// Check for pending state on the network change
	changeServiceClient, changeServiceClientErr := env.Config().NewChangeServiceClient()
	assert.Nil(t, changeServiceClientErr)
	assert.True(t, changeServiceClient != nil)
	listNetworkChangeRequest := &diags.ListNetworkChangeRequest{
		Subscribe:     false,
		ChangeID:      networkChangeID,
		WithoutReplay: false,
	}
	listNetworkChangesClient, listNetworkChangesClientErr := changeServiceClient.ListNetworkChanges(context.Background(), listNetworkChangeRequest)
	assert.Nil(t, listNetworkChangesClientErr)
	assert.True(t, listNetworkChangesClient != nil)
	networkChangeResponse, networkChangeResponseErr := listNetworkChangesClient.Recv()
	assert.Nil(t, networkChangeResponseErr)
	assert.True(t, networkChangeResponse != nil)
	assert.Equal(t, change.State_PENDING, networkChangeResponse.Change.Status.State)

	// Start the device simulator
	simulator := env.NewSimulator().SetName(offlineInTopoModDeviceName)
	simulatorEnv := simulator.AddOrDie()
	time.Sleep(2 * time.Second)

	// Interrogate the device to check that the value was set properly
	deviceGnmiClient := getDeviceGNMIClient(t, simulatorEnv)
	checkDeviceValue(t, deviceGnmiClient, makeDevicePath(offlineInTopoModDeviceName, offlineInTopoModPath), offlineInTopoModValue)

	// Check that the network change has completed
	networkChangeResponse, networkChangeResponseErr = listNetworkChangesClient.Recv()
	assert.Nil(t, networkChangeResponseErr)
	assert.True(t, networkChangeResponse != nil)
	assert.Equal(t, change.State_COMPLETE, networkChangeResponse.Change.Status.State)
}
