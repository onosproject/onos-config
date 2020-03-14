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
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-test/pkg/helm"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-test/pkg/util/random"
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
	simulator := helm.Helm().
		Chart("/etc/charts/device-simulator").
		Release(random.NewPetName(2))
	err := simulator.Install(true)
	assert.NoError(t, err)

	// Wait for config to connect to the device
	gnmi.WaitForDeviceAvailable(t, createRemoveDeviceModDeviceName, 1*time.Minute)

	// Make a GNMI client to use for requests
	c := gnmi.GetGNMIClientOrFail(t)

	devicePath := gnmi.GetDevicePathWithValue(createRemoveDeviceModDeviceName, createRemoveDeviceModPath, createRemoveDeviceModValue1, proto.StringVal)

	// Set a value using gNMI client - device is up
	networkChangeID := gnmi.SetGNMIValueOrFail(t, c, devicePath, gnmi.NoPaths, gnmi.NoExtensions)
	assert.True(t, networkChangeID != "")

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, c, devicePath, createRemoveDeviceModValue1, 0, "Query after set returned the wrong value")

	// Wait for config to reconnect to the device
	gnmi.WaitForDeviceAvailable(t, createRemoveDeviceModDeviceName, 1*time.Minute)

	// Check that the network change has completed
	gnmi.WaitForNetworkChangeComplete(t, networkChangeID, 10*time.Second)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient := gnmi.GetDeviceGNMIClientOrFail(t, simulator)
	gnmi.CheckDeviceValue(t, deviceGnmiClient, devicePath, createRemoveDeviceModValue1)

	//  Shut down the simulator
	err = simulator.Uninstall()
	assert.NoError(t, err)
	gnmi.WaitForDeviceUnavailable(t, createRemoveDeviceModDeviceName, 1*time.Minute)

	// Set a value using gNMI client - device is down
	setPath2 := gnmi.GetDevicePathWithValue(createRemoveDeviceModDeviceName, createRemoveDeviceModPath, createRemoveDeviceModValue2, proto.StringVal)
	networkChangeID2 := gnmi.SetGNMIValueOrFail(t, c, setPath2, gnmi.NoPaths, gnmi.NoExtensions)
	assert.True(t, networkChangeID2 != "")

	// Restart simulated device
	err = simulator.Install(true)
	assert.NoError(t, err)

	// Wait for config to connect to the device
	gnmi.WaitForDeviceAvailable(t, createRemoveDeviceModDeviceName, 2*time.Minute)

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, c, devicePath, createRemoveDeviceModValue2, 0, "Query after set 2 returns wrong value")

	// Check that the network change has completed
	gnmi.WaitForNetworkChangeComplete(t, networkChangeID2, 10*time.Second)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient2 := gnmi.GetDeviceGNMIClientOrFail(t, simulator)
	gnmi.CheckDeviceValue(t, deviceGnmiClient2, devicePath, createRemoveDeviceModValue2)

	// Clean up the simulator, as it isn't under onit control
	err = simulator.Uninstall()
	assert.NoError(t, err)
}
