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
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-topo/api/device"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
	"strconv"
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
	deviceClient, deviceClientError := env.Topo().NewDeviceServiceClient()
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
		TLS:         device.TlsConfig{},
		Type:        unreachableDeviceModDeviceType,
		Role:        "",
		Attributes:  nil,
		Protocols:   nil,
	}
	addRequest := &device.AddRequest{Device: newDevice}
	addResponse, addResponseError := deviceClient.Add(context.Background(), addRequest)
	assert.NotNil(t, addResponse)
	assert.Nil(t, addResponseError)

	// Make a GNMI client to use for requests
	c, err := env.Config().NewGNMIClient()
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Set a value using gNMI client to the device
	extNameDeviceType := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionDeviceType,
			Msg: []byte(unreachableDeviceModDeviceType),
		},
	}
	extNameDeviceVersion := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionVersion,
			Msg: []byte(unreachableDeviceModDeviceVersion),
		},
	}
	extensions := []*gnmi_ext.Extension{{Ext: &extNameDeviceType}, {Ext: &extNameDeviceVersion}}

	setPath := makeDevicePath(unreachableDeviceModDeviceName, unreachableDeviceModPath)
	setPath[0].pathDataValue = unreachableDeviceModValue
	setPath[0].pathDataType = StringVal

	// Set the value - should return a pending change
	_, extensionsSet, errorSet := gNMISet(MakeContext(), c, setPath, noPaths, extensions)
	assert.NoError(t, errorSet)
	assert.Equal(t, 1, len(extensionsSet))
	extensionBefore := extensionsSet[0].GetRegisteredExt()
	assert.Equal(t, extensionBefore.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))

	// Check that the value was set correctly in the cache
	valueAfter, extensions, errorAfter := gNMIGet(MakeContext(), c, makeDevicePath(unreachableDeviceModDeviceName, unreachableDeviceModPath))
	assert.NoError(t, errorAfter)
	assert.Equal(t, 0, len(extensions))
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, unreachableDeviceModValue, valueAfter[0].pathDataValue, "Query after set returned the wrong value: %s\n", valueAfter)
}
