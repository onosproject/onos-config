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

package config

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	createRemoveTargetModPath          = "/system/clock/config/timezone-name"
	createRemoveTargetModValue1        = "Europe/Paris"
	createRemoveTargetModValue2        = "Europe/London"
	createRemoveTargetModDeviceName    = "offline-sim-crd"
	createRemoveTargetModDeviceVersion = "1.0.0"
	createRemoveTargetModDeviceType    = "devicesim-1.0.x"
)

// TestCreatedRemovedDevice tests set/query of a single GNMI path to a single device that is created, removed, then created again
func (s *TestSuite) TestCreatedRemovedDevice(t *testing.T) {
	t.Skip()
	topoClient, err := gnmi.NewTopoClient()
	assert.NotNil(t, topoClient)
	assert.Nil(t, err)

	newTarget := &topo.Object{
		ID:   createRemoveTargetModDeviceName,
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{
				KindID: createRemoveTargetModDeviceType,
			},
		},
	}

	_ = newTarget.SetAspect(&topo.Configurable{
		Type:    createRemoveTargetModDeviceType,
		Address: createRemoveTargetModDeviceName + ":11161",
		Version: createRemoveTargetModDeviceVersion,
		Timeout: uint64((10 * time.Second).Milliseconds()),
	})

	_ = newTarget.SetAspect(&topo.TLSOptions{Plain: true})

	err = topoClient.Create(context.Background(), newTarget)
	assert.NoError(t, err)

	//  Start a new simulated device
	simulator := helm.
		Chart("device-simulator").
		Release(offlineTargetName)
	err = simulator.Install(true)
	assert.NoError(t, err)

	// Wait for config to connect to the device
	gnmi.WaitForTargetAvailable(t, createRemoveTargetModDeviceName, 1*time.Minute)

	// Make a GNMI client to use for requests
	c := gnmi.GetGNMIClientOrFail(t)

	targetPath := gnmi.GetTargetPathWithValue(createRemoveTargetModDeviceName, createRemoveTargetModPath, createRemoveTargetModValue1, proto.StringVal)

	// Set a value using gNMI client - device is up
	transactionID, transactionIndex := gnmi.SetGNMIValueOrFail(t, c, targetPath, gnmi.NoPaths, gnmi.NoExtensions)
	assert.True(t, transactionID != "")

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, c, targetPath, createRemoveTargetModValue1, 0, "Query after set returned the wrong value")

	// Wait for config to reconnect to the device
	gnmi.WaitForTargetAvailable(t, createRemoveTargetModDeviceName, 1*time.Minute)

	// Check that the network change has completed
	gnmi.WaitForTransactionComplete(t, transactionID, transactionIndex, 10*time.Second)

	// interrogate the device to check that the value was set properly
	targetGnmiClient := gnmi.GetTargetGNMIClientOrFail(t, simulator)
	gnmi.CheckTargetValue(t, targetGnmiClient, targetPath, createRemoveTargetModValue1)

	//  Shut down the simulator
	gnmi.DeleteSimulator(t, simulator)
	gnmi.WaitForTargetUnavailable(t, createRemoveTargetModDeviceName, 1*time.Minute)

	// Set a value using gNMI client - device is down
	setPath2 := gnmi.GetTargetPathWithValue(createRemoveTargetModDeviceName, createRemoveTargetModPath, createRemoveTargetModValue2, proto.StringVal)
	transactionID2, transactionIndex2 := gnmi.SetGNMIValueOrFail(t, c, setPath2, gnmi.NoPaths, gnmi.NoExtensions)
	assert.True(t, transactionID2 != "")

	//  Restart simulated device
	err = simulator.Install(true)
	assert.NoError(t, err)

	// Wait for config to connect to the device
	gnmi.WaitForTargetAvailable(t, createRemoveTargetModDeviceName, 2*time.Minute)

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, c, targetPath, createRemoveTargetModValue2, 0, "Query after set 2 returns wrong value")

	// Check that the network change has completed
	gnmi.WaitForTransactionComplete(t, transactionID2, transactionIndex2, 10*time.Second)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient2 := gnmi.GetTargetGNMIClientOrFail(t, simulator)
	gnmi.CheckTargetValue(t, deviceGnmiClient2, targetPath, createRemoveTargetModValue2)

	// Clean up the simulator
	gnmi.DeleteSimulator(t, simulator)
}
