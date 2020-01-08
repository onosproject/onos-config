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
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"testing"
)

const (
	oneLiveOneDeadDeviceModPath          = "/system/clock/config/timezone-name"
	oneLiveOneDeadDeviceModValue         = "Europe/Rome"
	oneLiveOneDeadDeviceName    = "dead-dev-1"
	oneLiveOneDeadDeviceVersion = "1.0.0"
	oneLiveOneDeadDeviceType    = "Devicesim"
	oneLiveOneDeadDeviceAddress          = "198.18.0.1:4"
)

// TestOneLiveOneDeadDevice tests GNMI operations to an offline device followed by operations to a connected device
func (s *TestSuite) TestOneLiveOneDeadDevice(t *testing.T) {

	// Make a GNMI client to use for requests
	gnmiClient := getGNMIClientOrFail(t)

	// Set a value using gNMI client to the offline device
	extNameDeviceType := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionDeviceType,
			Msg: []byte(oneLiveOneDeadDeviceType),
		},
	}
	extNameDeviceVersion := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionVersion,
			Msg: []byte(oneLiveOneDeadDeviceVersion),
		},
	}
	extensions := []*gnmi_ext.Extension{{Ext: &extNameDeviceType}, {Ext: &extNameDeviceVersion}}

	offlineDevicePath := getDevicePathWithValue("offline-device", oneLiveOneDeadDeviceModPath, oneLiveOneDeadDeviceModValue, StringVal)

	// Set the value - should return a pending change
	setGNMIValueOrFail(t, gnmiClient, offlineDevicePath, noPaths, extensions)

	// Check that the value was set correctly in the cache
	checkGNMIValue(t, gnmiClient, offlineDevicePath, oneLiveOneDeadDeviceModValue, 0, "Query after set returned the wrong value")

	// Create an online device
	simulator := env.NewSimulator().AddOrDie()

	// Set a value to the online device
	onlineDevicePath := getDevicePathWithValue(simulator.Name(), oneLiveOneDeadDeviceModPath, oneLiveOneDeadDeviceModValue, StringVal)

	// Set a value using gNMI client
	setGNMIValueOrFail(t, gnmiClient, onlineDevicePath, noPaths, noExtensions)

	// Check that the value was set correctly
	checkGNMIValue(t, gnmiClient, onlineDevicePath, oneLiveOneDeadDeviceModValue, 0, "Query after set returned the wrong value")
}
