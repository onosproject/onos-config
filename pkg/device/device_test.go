// Copyright 2021-present Open Networking Foundation.
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

package device

import (
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	deviceName    = "device-1"
	deviceDisplay = "Device 1"
	deviceType    = "devicesim"
	deviceAddress = "devicesim-1:10161"
	deviceVersion = "1.0.0"
)

func Test_ObjectToDevice(t *testing.T) {
	deviceAsObject := topo.Object{
		ID:   topo.ID(deviceName),
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{
				KindID: topo.ID(deviceType),
			},
		},
	}

	err := deviceAsObject.SetAspect(&topo.TLSOptions{
		Insecure: true,
		Plain:    true,
	})
	assert.NoError(t, err)

	err = deviceAsObject.SetAspect(&topo.Asset{
		Name: deviceDisplay,
	})
	assert.NoError(t, err)

	err = deviceAsObject.SetAspect(&topo.MastershipState{})
	assert.NoError(t, err)

	err = deviceAsObject.SetAspect(&topo.Configurable{
		Type:    deviceType,
		Address: deviceAddress,
		Version: deviceVersion,
		Timeout: uint64(time.Second.Milliseconds() * 5),
	})
	assert.NoError(t, err)

	err = deviceAsObject.SetAspect(&topo.Protocols{})
	assert.NoError(t, err)

	device, err := ToDevice(&deviceAsObject)
	assert.NoError(t, err)
	assert.NotNil(t, device)

	assert.Equal(t, deviceName, string(device.ID))
	assert.Equal(t, deviceDisplay, device.Displayname)
	assert.Equal(t, deviceType, string(device.Type))
	assert.Equal(t, deviceVersion, device.Version)
	assert.Equal(t, deviceAddress, device.Address)
	assert.Equal(t, 5000, int(device.Timeout.Milliseconds()))
	assert.True(t, device.TLS.Plain)
	assert.True(t, device.TLS.Insecure)

}

func Test_ObjectToDevice_error(t *testing.T) {
	deviceAsObject := topo.Object{
		ID:   topo.ID(deviceName),
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{
				KindID: topo.ID(deviceType),
			},
		},
	}

	_, err := ToDevice(&deviceAsObject)
	assert.EqualError(t, err, "topo entity device-1 must have 'onos.topo.Asset' aspect to work with onos-config")
}

func Test_DeviceToObject(t *testing.T) {
	timeout := time.Millisecond * 5
	d := &Device{
		ID:          ID(deviceName),
		Type:        deviceType,
		Displayname: deviceDisplay,
		Address:     deviceAddress,
		Version:     deviceVersion,
		Timeout:     &timeout,
		TLS: TLSConfig{
			Plain:    true,
			Insecure: true,
		},
	}

	deviceObject := ToObject(d)
	assert.NotNil(t, deviceObject)

	assert.Equal(t, deviceName, string(deviceObject.ID))

	asset := &topo.Asset{}
	err := deviceObject.GetAspect(asset)
	assert.NoError(t, err)
	assert.Equal(t, deviceDisplay, asset.Name)

	configurable := &topo.Configurable{}
	err = deviceObject.GetAspect(configurable)
	assert.NoError(t, err)
	assert.Equal(t, deviceVersion, configurable.Version)
	assert.Equal(t, deviceAddress, configurable.Address)

	tlsOptions := &topo.TLSOptions{}
	err = deviceObject.GetAspect(tlsOptions)
	assert.NoError(t, err)
	assert.True(t, tlsOptions.Plain)
	assert.True(t, tlsOptions.Insecure)
}
