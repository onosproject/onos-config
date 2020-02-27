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
	gnb "github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
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
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Set a value using gNMI client to the offline device
	extNameDeviceType := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnb.GnmiExtensionDeviceType,
			Msg: []byte("Devicesim"),
		},
	}
	extNameDeviceVersion := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnb.GnmiExtensionVersion,
			Msg: []byte("1.0.0"),
		},
	}
	extensions := []*gnmi_ext.Extension{{Ext: &extNameDeviceType}, {Ext: &extNameDeviceVersion}}
	devicePath := gnmi.GetDevicePathWithValue(offlineDeviceName, modPath, modValue, proto.StringVal)
	networkChangeID := gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, extensions)

	// Check that the value was set correctly
	simulatorEnv := simulator.AddOrDie()
	time.Sleep(2 * time.Second)
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, modValue, 0, "Query after set returned the wrong value")

	// Check that the value was set properly on the device
	gnmi.WaitForNetworkChangeComplete(t, networkChangeID, 10*time.Second)
	deviceGnmiClient := gnmi.GetDeviceGNMIClientOrFail(t, simulatorEnv)
	gnmi.CheckDeviceValue(t, deviceGnmiClient, devicePath, modValue)
}
