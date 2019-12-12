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
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	createRemoveDeviceModPath          = "/system/clock/config/timezone-name"
	createRemoveDeviceModValue1         = "Europe/Paris"
	createRemoveDeviceModValue2         = "Europe/London"
	createRemoveDeviceModValue3         = "Europe/Munich"
	createRemoveDeviceModDeviceName    = "offline-dev-1"
	createRemoveDeviceModDeviceVersion = "1.0.0"
	createRemoveDeviceModDeviceType    = "Devicesim"
)

// TestOfflineDeviceInTopo tests set/query of a single GNMI path to a single device that is in the config but offline
func (s *TestSuite) TestCreateRemovedDevice(t *testing.T) {

	//  Start a new simulated device
	simulator := env.NewSimulator().SetName(createRemoveDeviceModDeviceName)
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

	_, extensionsSet, errorSet := gNMISet(MakeContext(), c, setPath, noPaths, noExtensions)
	assert.NoError(t, errorSet)
	assert.Equal(t, 1, len(extensionsSet))
	extensionBefore := extensionsSet[0].GetRegisteredExt()
	assert.Equal(t, extensionBefore.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))

	// Check that the value was set correctly
	valueAfter, extensions, errorAfter := gNMIGet(MakeContext(), c, makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath))
	assert.NoError(t, errorAfter)
	assert.Equal(t, 0, len(extensions))
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, createRemoveDeviceModValue1, valueAfter[0].pathDataValue, "Query after set returned the wrong value: %s\n", valueAfter)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient := getDeviceGNMIClient(t, simulatorEnv)
	checkDeviceValue(t, deviceGnmiClient, makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath), createRemoveDeviceModValue1)

	//  Shut down the simulator
	simulatorEnv.KillOrDie()
	time.Sleep(2 * time.Second)

	// Set a value using gNMI client - device is down
	setPath2 := makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath)
	setPath2[0].pathDataValue = createRemoveDeviceModValue2
	setPath2[0].pathDataType = StringVal
	_, extensionsSet2, errorSet2 := gNMISet(MakeContext(), c, setPath, noPaths, noExtensions)
	assert.NoError(t, errorSet2)
	assert.Equal(t, 1, len(extensionsSet2))
	extensionBefore2 := extensionsSet2[0].GetRegisteredExt()
	assert.Equal(t, extensionBefore2.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))

	//  Restart simulated device
	simulatorEnv2 := simulator.
	time.Sleep(2 * time.Second)

	// Check that the value was set correctly
	valueAfter2, extensions2, errorAfter2 := gNMIGet(MakeContext(), c, makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath))
	assert.NoError(t, errorAfter2)
	assert.Equal(t, 0, len(extensions2))
	assert.NotEqual(t, "", valueAfter2, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, createRemoveDeviceModValue2, valueAfter2[0].pathDataValue, "Query after set returned the wrong value: %s\n", valueAfter2)

	// interrogate the device to check that the value was set properly
	deviceGnmiClient2 := getDeviceGNMIClient(t, simulatorEnv2)
	checkDeviceValue(t, deviceGnmiClient2, makeDevicePath(createRemoveDeviceModDeviceName, createRemoveDeviceModPath), createRemoveDeviceModValue2)
}
