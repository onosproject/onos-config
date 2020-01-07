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
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	createRemoveDeviceModPath          = "/system/clock/config/timezone-name"
	createRemoveDeviceModValue1        = "Europe/Paris"
	createRemoveDeviceModValue2        = "Europe/London"
	createRemoveDeviceModDeviceName    = "offline-sim-crd"
	createRemoveDeviceModDeviceVersion = "1.0.0"
	createRemoveDeviceModDeviceType    = "Devicesim"
)

// TestCreatedRemovedDevice tests set/query of a single GNMI path to a single device that is created, removed, then created again
func (s *TestSuite) TestCreatedRemovedDevice(t *testing.T) {
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

	// Wait for config to connect to the device
	testutils.WaitForDeviceAvailable(t, createRemoveDeviceModDeviceName, 1*time.Minute)

	// Make a GNMI client to use for requests
	c := getGNMIClientOrFail(t)

	devicePath := getDevicePathWithValue(createRemoveDeviceModDeviceName, createRemoveDeviceModPath, createRemoveDeviceModValue1, StringVal)

	// Set a value using gNMI client - device is up
	networkChangeID := setGNMIValueOrFail(t, c, devicePath, noPaths, noExtensions)
	assert.True(t, networkChangeID != "")

	// Check that the value was set correctly
	checkGNMIValue(t, c, devicePath, createRemoveDeviceModValue1, 0, "Query after set returned the wrong value")

	// Wait for config to reconnect to the device
	testutils.WaitForDeviceAvailable(t, createRemoveDeviceModDeviceName, 1*time.Minute)

	// Check that the network change has completed
	testutils.WaitForNetworkChangeComplete(t, networkChangeID)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient := getDeviceGNMIClientOrFail(t, simulatorEnv)
	checkDeviceValue(t, deviceGnmiClient, devicePath, createRemoveDeviceModValue1)

	//  Shut down the simulator
	simulatorEnv.KillOrDie()
	simulatorEnv.RemoveOrDie()
	testutils.WaitForDeviceUnavailable(t, createRemoveDeviceModDeviceName, 1*time.Minute)

	// Set a value using gNMI client - device is down
	setPath2 := getDevicePathWithValue(createRemoveDeviceModDeviceName, createRemoveDeviceModPath, createRemoveDeviceModValue2, StringVal)
	networkChangeID2 := setGNMIValueOrFail(t, c, setPath2, noPaths, noExtensions)
	assert.True(t, networkChangeID2 != "")

	//  Restart simulated device
	time.Sleep(45 * time.Second) // Wait for simulator to shut down. Is there a better way to do this?
	simulatorEnv2 := simulator.AddOrDie()

	// Wait for config to connect to the device
	testutils.WaitForDeviceAvailable(t, createRemoveDeviceModDeviceName, 2*time.Minute)

	// Check that the value was set correctly
	checkGNMIValue(t, c, devicePath, createRemoveDeviceModValue2, 0, "Query after set 2 returns wrong value")

	// Check that the network change has completed
	testutils.WaitForNetworkChangeComplete(t, networkChangeID2)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient2 := getDeviceGNMIClientOrFail(t, simulatorEnv2)
	checkDeviceValue(t, deviceGnmiClient2, devicePath, createRemoveDeviceModValue2)

	// Clean up the simulator, as it isn't under onit control
	simulatorEnv.KillOrDie()
	simulatorEnv.RemoveOrDie()
}
