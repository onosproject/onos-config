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

	gnb "github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/onosproject/onos-topo/api/topo"
	device "github.com/onosproject/onos-topo/api/topo"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
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
	deviceClient, deviceClientError := gnmi.NewDeviceServiceClient()
	assert.NotNil(t, deviceClient)
	assert.Nil(t, deviceClientError)
	newKind := &topo.Object{
		ID:   unreachableDeviceModDeviceType,
		Type: topo.Object_KIND,
		Obj: &topo.Object_Kind{
			Kind: &topo.Kind{
				Name: unreachableDeviceModDeviceType,
			},
		},
		Attributes: map[string]string{},
	}
	setRequest := &topo.SetRequest{Objects: []*topo.Object{newKind}}
	setResponse, setResponseError := deviceClient.Set(context.Background(), setRequest)
	assert.NotNil(t, setResponse)
	assert.Nil(t, setResponseError)
	newDevice := &device.Object{
		ID:   unreachableDeviceModDeviceName,
		Type: device.Object_ENTITY,
		Obj: &device.Object_Entity{
			Entity: &device.Entity{
				KindID: unreachableDeviceModDeviceType,
			},
		},
		Attributes: map[string]string{
			device.Address: unreachableDeviceAddress,
			device.Version: unreachableDeviceModDeviceVersion,
		},
	}
	addRequest := &device.SetRequest{Objects: []*device.Object{newDevice}}
	addResponse, addResponseError := deviceClient.Set(context.Background(), addRequest)
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
