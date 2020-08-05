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
	"testing"
	"time"

	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-topo/api/topo"
	device "github.com/onosproject/onos-topo/api/topo"
	"github.com/stretchr/testify/assert"
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
	t.Skip()
	deviceClient, deviceClientError := gnmi.NewDeviceServiceClient()
	assert.NotNil(t, deviceClient)
	assert.Nil(t, deviceClientError)

	newKind := &topo.Object{
		ID:   createRemoveDeviceModDeviceType,
		Type: topo.Object_KIND,
		Obj: &topo.Object_Kind{
			Kind: &topo.Kind{
				Name: createRemoveDeviceModDeviceType,
			},
		},
		Attributes: map[string]string{
			topo.Address:  "",
			topo.Version:  "",
			topo.TLSPlain: "true",
			topo.Timeout:  "10",
		},
	}
	setRequest := &topo.SetRequest{Objects: []*topo.Object{newKind}}
	setResponse, setResponseError := deviceClient.Set(context.Background(), setRequest)
	assert.NotNil(t, setResponse)
	assert.Nil(t, setResponseError)
	newDevice := &device.Object{
		ID:   createRemoveDeviceModDeviceName,
		Type: device.Object_ENTITY,
		Obj: &device.Object_Entity{
			Entity: &topo.Entity{
				KindID: createRemoveDeviceModDeviceType,
			},
		},
		Attributes: map[string]string{
			device.Address:  createRemoveDeviceModDeviceName + ":11161",
			device.Version:  createRemoveDeviceModDeviceVersion,
			device.TLSPlain: "true",
			device.Timeout:  "10",
		},
	}
	setRequest = &device.SetRequest{Objects: []*device.Object{newDevice}}
	setResponse, setResponseError = deviceClient.Set(context.Background(), setRequest)
	assert.NotNil(t, setResponse)
	assert.Nil(t, setResponseError)

	//  Start a new simulated device
	simulator := helm.
		Chart("device-simulator").
		Release(offlineDeviceName)
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
	gnmi.DeleteSimulator(t, simulator)
	gnmi.WaitForDeviceUnavailable(t, createRemoveDeviceModDeviceName, 1*time.Minute)

	// Set a value using gNMI client - device is down
	setPath2 := gnmi.GetDevicePathWithValue(createRemoveDeviceModDeviceName, createRemoveDeviceModPath, createRemoveDeviceModValue2, proto.StringVal)
	networkChangeID2 := gnmi.SetGNMIValueOrFail(t, c, setPath2, gnmi.NoPaths, gnmi.NoExtensions)
	assert.True(t, networkChangeID2 != "")

	//  Restart simulated device
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

	// Clean up the simulator
	gnmi.DeleteSimulator(t, simulator)
}
