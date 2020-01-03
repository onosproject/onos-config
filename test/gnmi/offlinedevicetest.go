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
	testutils "github.com/onosproject/onos-config/test/utils"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"testing"
	"time"
)

const (
	modPath           = "/system/clock/config/timezone-name"
	modValue          = "Europe/Rome"
	offlineDeviceName = "test-offline-device-1"
)

// TestOfflineDevice tests set/query of a single GNMI path to a single device that is initially not in the config
func (s *TestSuite) TestOfflineDevice(t *testing.T) {
	simulator := env.NewSimulator().SetName(offlineDeviceName)

	// Make a GNMI client to use for requests
	gnmiClient := getGNMIClientOrFail(t)

	// Set a value using gNMI client to the offline device
	extNameDeviceType := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionDeviceType,
			Msg: []byte("Devicesim"),
		},
	}
	extNameDeviceVersion := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionVersion,
			Msg: []byte("1.0.0"),
		},
	}
	extensions := []*gnmi_ext.Extension{{Ext: &extNameDeviceType}, {Ext: &extNameDeviceVersion}}
	devicePath := getDevicePathWithValue(offlineDeviceName, modPath, modValue, StringVal)
	networkChangeID := setGNMIValueOrFail(t, gnmiClient, devicePath, noPaths, extensions)

	// Check that the value was set correctly
	simulatorEnv := simulator.AddOrDie()
	time.Sleep(2 * time.Second)
	checkGNMIValue(t, gnmiClient, devicePath, modValue, 0, "Query after set returned the wrong value")

	// Check that the value was set properly on the device
	testutils.WaitForNetworkChangeComplete(t, networkChangeID)
	deviceGnmiClient := getDeviceGNMIClientOrFail(t, simulatorEnv)
	checkDeviceValue(t, deviceGnmiClient, devicePath, modValue)
}
