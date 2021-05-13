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
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/device"
	gnb "github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	unreachableDeviceModPath          = "/system/clock/config/timezone-name"
	unreachableDeviceModValue         = "Europe/Rome"
	unreachableDeviceModDeviceName    = "unreachable-dev-1"
	unreachableDeviceModDeviceVersion = "1.0.0"
	unreachableDeviceModDeviceType    = "Devicesim"
	unreachableDeviceAddress          = "198.18.0.1:4"
)

// TestUnreachableDevice tests set/query of a single GNMI path to a device that will never respond
func (s *TestSuite) TestUnreachableDevice(t *testing.T) {
	deviceClient, deviceClientError := gnmi.NewTopoClient()
	assert.NotNil(t, deviceClient)
	assert.Nil(t, deviceClientError)
	newDevice := &device.Device{
		ID:          unreachableDeviceModDeviceName,
		Revision:    0,
		Address:     unreachableDeviceAddress,
		Target:      "",
		Version:     unreachableDeviceModDeviceVersion,
		Timeout:     nil,
		Credentials: device.Credentials{},
		TLS:         device.TLSConfig{},
		Type:        unreachableDeviceModDeviceType,
		Role:        "",
		Protocols:   nil,
	}
	addRequest := &topo.CreateRequest{Object: device.ToObject(newDevice)}
	addResponse, addResponseError := deviceClient.Create(context.Background(), addRequest)
	assert.NotNil(t, addResponse)
	assert.Nil(t, addResponseError)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Set a value using gNMI client to the device
	extNameDeviceType := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnb.GnmiExtensionDeviceType,
			Msg: []byte(unreachableDeviceModDeviceType),
		},
	}
	extNameDeviceVersion := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnb.GnmiExtensionVersion,
			Msg: []byte(unreachableDeviceModDeviceVersion),
		},
	}
	extensions := []*gnmi_ext.Extension{{Ext: &extNameDeviceType}, {Ext: &extNameDeviceVersion}}

	devicePath := gnmi.GetDevicePathWithValue(unreachableDeviceModDeviceName, unreachableDeviceModPath, unreachableDeviceModValue, proto.StringVal)

	// Set the value - should return a pending change
	gnmi.SetGNMIValueOrFail(t, gnmiClient, devicePath, gnmi.NoPaths, extensions)

	// Check that the value was set correctly in the cache
	gnmi.CheckGNMIValue(t, gnmiClient, devicePath, unreachableDeviceModValue, 0, "Query after set returned the wrong value")
}
