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
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-test/pkg/helm"
	"github.com/onosproject/onos-test/pkg/util/random"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestOneLiveOneDeadDevice tests GNMI operations to an offline device followed by operations to a connected device
func (s *TestSuite) TestOneLiveOneDeadDevice(t *testing.T) {

	const (
		modPath  = "/system/clock/config/timezone-name"
		modValue = "Europe/Rome"
	)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Set a value using gNMI client to the offline device
	offlineDevicePath := gnmi.GetDevicePathWithValue("offline-device", modPath, modValue, proto.StringVal)

	// Set the value - should return a pending change
	gnmi.SetGNMIValueOrFail(t, gnmiClient, offlineDevicePath, gnmi.NoPaths, gnmi.GetSimulatorExtensions())

	// Check that the value was set correctly in the cache
	gnmi.CheckGNMIValue(t, gnmiClient, offlineDevicePath, modValue, 0, "Query after set returned the wrong value")

	// Create an online device
	onlineSimulator := helm.Namespace().
		Chart("/etc/charts/device-simulator").
		Release(random.NewPetName(2))
	err := onlineSimulator.Install(true)
	assert.NoError(t, err)

	// Set a value to the online device
	onlineDevicePath := gnmi.GetDevicePathWithValue(onlineSimulator.Name(), modPath, modValue, proto.StringVal)
	nid := gnmi.SetGNMIValueOrFail(t, gnmiClient, onlineDevicePath, gnmi.NoPaths, gnmi.NoExtensions)
	gnmi.WaitForNetworkChangeComplete(t, nid, 10*time.Second)

	// Check that the value was set correctly in the cache
	gnmi.CheckGNMIValue(t, gnmiClient, onlineDevicePath, modValue, 0, "Query after set returned the wrong value")

	// Check that the value was set correctly on the device
	deviceGnmiClient := gnmi.GetDeviceGNMIClientOrFail(t, onlineSimulator)
	gnmi.CheckDeviceValue(t, deviceGnmiClient, onlineDevicePath, modValue)
}
