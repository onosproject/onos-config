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
	"github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

const (
	createRemoveDeviceModPath          = "/system/clock/config/timezone-name"
	createRemoveDeviceModValue1        = "Europe/Paris"
	createRemoveDeviceModValue2        = "Europe/London"
	createRemoveDeviceModValue3        = "Europe/Munich"
	createRemoveDeviceModDeviceName    = "offline-dev-1"
	createRemoveDeviceModDeviceVersion = "1.0.0"
	createRemoveDeviceModDeviceType    = "Devicesim"
)

// TestCreateRemovedDevice tests set/query of a single GNMI path to a single device that is created, removed, then created again
func (s *TestSuite) TestCreateRemovedDevice(t *testing.T) {
	deviceClient, deviceClientError := env.Topo().NewDeviceServiceClient()
	assert.NotNil(t, deviceClient)
	assert.Nil(t, deviceClientError)
	timeout := 10 * time.Second
	newDevice := &device.Device{
		ID:      createRemoveDeviceModDeviceName,
		Address: createRemoveDeviceModDeviceName + ":11161",
		Type:    createRemoveDeviceModDeviceType,
		Version: createRemoveDeviceModDeviceVersion,
		Timeout: &timeout,
		TLS: device.TlsConfig{
			Plain: true,
		},
	}
	addRequest := &device.AddRequest{Device: newDevice}
	addResponse, addResponseError := deviceClient.Add(context.Background(), addRequest)
	assert.NotNil(t, addResponse)
	assert.Nil(t, addResponseError)

	//  Start a new simulated device
	simulator := env.NewSimulator().SetName(createRemoveDeviceModDeviceName).SetAddDevice(false)
	simulatorEnv := simulator.AddOrDie()
	time.Sleep(2 * time.Second)

	// Make a GNMI client to use for requests
	c, err := env.Config().NewGNMIClient()
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Set a value using gNMI client - device is up
	setPath := makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath)
	setPath[0].pathDataValue = createRemoveDeviceModValue1
	setPath[0].pathDataType = StringVal

	_, extensionsSet, errorSet := gNMISet(testutils.MakeContext(), c, setPath, noPaths, noExtensions)
	assert.NoError(t, errorSet)
	assert.Equal(t, 1, len(extensionsSet))
	extensionBefore := extensionsSet[0].GetRegisteredExt()
	assert.Equal(t, extensionBefore.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))
	networkChangeID := network.ID(extensionBefore.Msg)

	// Check that the value was set correctly
	valueAfter, extensions, errorAfter := gNMIGet(testutils.MakeContext(), c, makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath))
	assert.NoError(t, errorAfter)
	assert.Equal(t, 0, len(extensions))
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, createRemoveDeviceModValue1, valueAfter[0].pathDataValue, "Query after set returned the wrong value: %s\n", valueAfter)

	// Wait for config to connect to the device
	testutils.WaitForDeviceAvailable(t, createRemoveDeviceModDeviceName, 1*time.Minute)

	// Check that the network change has completed
	testutils.WaitForNetworkChangeComplete(t, networkChangeID)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient := getDeviceGNMIClient(t, simulatorEnv)
	checkDeviceValue(t, deviceGnmiClient, makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath), createRemoveDeviceModValue1)

	//  Shut down the simulator
	simulatorEnv.KillOrDie()
	simulatorEnv.RemoveOrDie()
	time.Sleep(60 * time.Second)

	// Set a value using gNMI client - device is down
	setPath2 := makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath)
	setPath2[0].pathDataValue = createRemoveDeviceModValue2
	setPath2[0].pathDataType = StringVal
	_, extensionsSet2, errorSet2 := gNMISet(testutils.MakeContext(), c, setPath, noPaths, noExtensions)
	assert.NoError(t, errorSet2)
	assert.Equal(t, 1, len(extensionsSet2))
	extensionBefore2 := extensionsSet2[0].GetRegisteredExt()
	assert.Equal(t, extensionBefore2.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))
	//networkChangeID2 := network.ID(extensionBefore2.Msg)

	//  Restart simulated device
	simulatorEnv2 := simulator.AddOrDie()

	// Wait for config to connect to the device
	testutils.WaitForDeviceAvailable(t, createRemoveDeviceModDeviceName, 2*time.Minute)

	// Check that the network change has completed
	//testutils.WaitForNetworkChangeComplete(t, networkChangeID2)
	time.Sleep(30 * time.Second)

	// Check that the value was set correctly
	valueAfter2, extensions2, errorAfter2 := gNMIGet(testutils.MakeContext(), c, makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath))
	assert.NoError(t, errorAfter2)
	assert.Equal(t, 0, len(extensions2))
	assert.NotEqual(t, "", valueAfter2, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, createRemoveDeviceModValue2, valueAfter2[0].pathDataValue, "Query after set returned the wrong value: %s\n", valueAfter2)

	//time.Sleep(5 * time.Minute)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient2 := getDeviceGNMIClient(t, simulatorEnv2)
	checkDeviceValue(t, deviceGnmiClient2, makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath), createRemoveDeviceModValue2)
}
