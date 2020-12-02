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
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

func TestDeviceStore(t *testing.T) {
	ctrl := gomock.NewController(t)

	device1 := &topodevice.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}
	device2 := &topodevice.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}
	device3 := &topodevice.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}

	stream := NewMockDeviceService_ListClient(ctrl)
	stream.EXPECT().Recv().Return(&topodevice.ListResponse{Device: device1}, nil)
	stream.EXPECT().Recv().Return(&topodevice.ListResponse{Device: device2}, nil)
	stream.EXPECT().Recv().Return(&topodevice.ListResponse{Device: device3}, nil)
	stream.EXPECT().Recv().Return(nil, io.EOF)

	stream.EXPECT().Recv().Return(&topodevice.ListResponse{Device: device1}, nil)
	stream.EXPECT().Recv().Return(&topodevice.ListResponse{Device: device2}, nil)
	stream.EXPECT().Recv().Return(&topodevice.ListResponse{Device: device3}, nil)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().List(gomock.Any(), gomock.Any()).Return(stream, nil)
	client.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&topodevice.GetResponse{Device: device1}, nil)

	store := topoStore{
		client: client,
	}

	device, err := store.Get(device1.ID)
	assert.NoError(t, err)
	assert.Equal(t, device1.ID, device.ID)

	ch := make(chan *topodevice.Device)
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

	device1 := &topodevice.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}

	device1Connected := &topodevice.Device{
		ID:       "device-1",
		Revision: 1,
		Address:  "device-1:1234",
		Version:  "1.0.0",
	}

	protocolState := new(topodevice.ProtocolState)
	protocolState.Protocol = topodevice.Protocol_GNMI
	protocolState.ConnectivityState = topodevice.ConnectivityState_REACHABLE
	protocolState.ChannelState = topodevice.ChannelState_CONNECTED
	protocolState.ServiceState = topodevice.ServiceState_AVAILABLE
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&topodevice.GetResponse{Device: device1}, nil)

	store := topoStore{
		client: client,
	}

	device, err := store.Get(device1.ID)
	assert.NoError(t, err)
	assert.Equal(t, device1.ID, device.ID)

	client.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&topodevice.UpdateResponse{Device: device1Connected}, nil)

	deviceUpdated, err := store.Update(device1Connected)
	assert.NoError(t, err)
	assert.Equal(t, deviceUpdated.ID, device1Connected.ID)
	assert.Equal(t, deviceUpdated.Protocols[0].Protocol, topodevice.Protocol_GNMI)
	assert.Equal(t, deviceUpdated.Protocols[0].ConnectivityState, topodevice.ConnectivityState_REACHABLE)
	assert.Equal(t, deviceUpdated.Protocols[0].ChannelState, topodevice.ChannelState_CONNECTED)
	assert.Equal(t, deviceUpdated.Protocols[0].ServiceState, topodevice.ServiceState_AVAILABLE)

}

func nextDevice(t *testing.T, ch chan *topodevice.Device) *topodevice.Device {
	select {
	case d := <-ch:
		return d
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
