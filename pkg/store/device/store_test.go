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

package device

import (
	"github.com/golang/mock/gomock"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

func TestDeviceStore(t *testing.T) {
	ctrl := gomock.NewController(t)

	device1 := &devicetopo.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}
	device2 := &devicetopo.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}
	device3 := &devicetopo.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}

	stream := NewMockDeviceService_ListClient(ctrl)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: device1}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: device2}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: device3}, nil)
	stream.EXPECT().Recv().Return(nil, io.EOF)

	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: device1}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: device2}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: device3}, nil)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().List(gomock.Any(), gomock.Any()).Return(stream, nil)
	client.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&devicetopo.GetResponse{Device: device1}, nil)

	store := topoStore{
		client: client,
	}

	device, err := store.Get(device1.ID)
	assert.NoError(t, err)
	assert.Equal(t, device1.ID, device.ID)

	ch := make(chan *devicetopo.Device)
	err = store.List(ch)
	assert.NoError(t, err)

	device = nextDevice(t, ch)
	assert.Equal(t, device1.ID, device.ID)
	device = nextDevice(t, ch)
	assert.Equal(t, device2.ID, device.ID)
	device = nextDevice(t, ch)
	assert.Equal(t, device3.ID, device.ID)
}

func TestUpdateDevice(t *testing.T) {
	ctrl := gomock.NewController(t)

	device1 := &devicetopo.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}

	device1Connected := &devicetopo.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}

	protocolState := new(devicetopo.ProtocolState)
	protocolState.Protocol = devicetopo.Protocol_GNMI
	protocolState.ConnectivityState = devicetopo.ConnectivityState_REACHABLE
	protocolState.ChannelState = devicetopo.ChannelState_CONNECTED
	protocolState.ServiceState = devicetopo.ServiceState_AVAILABLE
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&devicetopo.GetResponse{Device: device1}, nil)

	store := topoStore{
		client: client,
	}

	device, err := store.Get(device1.ID)
	assert.NoError(t, err)
	assert.Equal(t, device1.ID, device.ID)

	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&devicetopo.UpdateResponse{Device: device1Connected}, nil)

	deviceUpdated, err := store.Update(device1Connected)
	assert.NoError(t, err)
	assert.Equal(t, deviceUpdated.ID, device1Connected.ID)
	assert.Equal(t, deviceUpdated.Protocols[0].Protocol, devicetopo.Protocol_GNMI)
	assert.Equal(t, deviceUpdated.Protocols[0].ConnectivityState, devicetopo.ConnectivityState_REACHABLE)
	assert.Equal(t, deviceUpdated.Protocols[0].ChannelState, devicetopo.ChannelState_CONNECTED)
	assert.Equal(t, deviceUpdated.Protocols[0].ServiceState, devicetopo.ServiceState_AVAILABLE)

}

func nextDevice(t *testing.T, ch chan *devicetopo.Device) *devicetopo.Device {
	select {
	case d := <-ch:
		return d
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
