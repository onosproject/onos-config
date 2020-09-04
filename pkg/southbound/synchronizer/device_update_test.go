// Copyright 2020-present Open Networking Foundation.
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

package synchronizer

import (
	"testing"

	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/test/mocks"
	"github.com/onosproject/onos-lib-go/pkg/cluster"

	"github.com/golang/mock/gomock"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"gotest.tools/assert"
)

const (
	device1        = "device1"
	deviceVersion1 = "1.0.0"
)

type AllMocks struct {
	DeviceStore     devicestore.Store
	MastershipStore mastership.Store
	DeviceClient    *mocks.MockDeviceServiceClient
}

func setUp(t *testing.T) *AllMocks {

	ctrl := gomock.NewController(t)
	client := mocks.NewMockDeviceServiceClient(ctrl)
	deviceStore, err := devicestore.NewStore(client)
	assert.NilError(t, err)

	node1 := cluster.NodeID("node1")
	mastershipStore, err := mastership.NewLocalStore("TestUpdateDisconnectedDevice", node1)
	assert.NilError(t, err)

	allMocks := AllMocks{
		DeviceStore:     deviceStore,
		MastershipStore: mastershipStore,
		DeviceClient:    client,
	}

	return &allMocks
}

func TestUpdateDisconnectedDevice(t *testing.T) {
	allMocks := setUp(t)

	device1Disconnected := &topodevice.Device{
		ID:         device1,
		Revision:   1,
		Address:    "device1:1234",
		Version:    deviceVersion1,
		Attributes: make(map[string]string),
	}

	allMocks.DeviceClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&topodevice.GetResponse{Device: device1Disconnected}, nil).AnyTimes()

	protocolState := new(topodevice.ProtocolState)
	protocolState.Protocol = topodevice.Protocol_GNMI
	device1Disconnected.Protocols = append(device1Disconnected.Protocols, protocolState)
	device1Disconnected.Attributes[mastershipTermKey] = "0"

	allMocks.DeviceClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&topodevice.UpdateResponse{Device: device1Disconnected}, nil).AnyTimes()

	session1 := &Session{
		device:          device1Disconnected,
		mastershipStore: allMocks.MastershipStore,
		deviceStore:     allMocks.DeviceStore,
	}

	err := session1.updateDisconnectedDevice()
	assert.NilError(t, err)
	updatedDevice, err := session1.deviceStore.Get(device1)
	assert.NilError(t, err)
	newDeviceTerm := updatedDevice.Attributes[mastershipTermKey]
	assert.Equal(t, newDeviceTerm, "1")
	assert.Equal(t, updatedDevice.Protocols[0].ConnectivityState, topodevice.ConnectivityState_UNREACHABLE)
	assert.Equal(t, updatedDevice.Protocols[0].ChannelState, topodevice.ChannelState_DISCONNECTED)
	assert.Equal(t, updatedDevice.Protocols[0].ServiceState, topodevice.ServiceState_UNAVAILABLE)
}

func TestUpdateConnectedDevice(t *testing.T) {
	allMocks := setUp(t)

	node1 := cluster.NodeID("node1")
	mastershipStore1, err := mastership.NewLocalStore("TestUpdateConnectedDevice", node1)
	assert.NilError(t, err)

	device1Connected := &topodevice.Device{
		ID:         device1,
		Revision:   1,
		Address:    "device1:1234",
		Version:    deviceVersion1,
		Attributes: make(map[string]string),
	}

	allMocks.DeviceClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&topodevice.GetResponse{Device: device1Connected}, nil).AnyTimes()

	protocolState := new(topodevice.ProtocolState)
	protocolState.Protocol = topodevice.Protocol_GNMI
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)
	device1Connected.Attributes[mastershipTermKey] = "0"

	allMocks.DeviceClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&topodevice.UpdateResponse{Device: device1Connected}, nil).AnyTimes()

	session1 := &Session{
		device:          device1Connected,
		mastershipStore: mastershipStore1,
		deviceStore:     allMocks.DeviceStore,
	}

	err = session1.updateConnectedDevice()
	assert.NilError(t, err)
	updatedDevice, err := session1.deviceStore.Get(device1)
	assert.NilError(t, err)
	newDeviceTerm := updatedDevice.Attributes[mastershipTermKey]
	assert.Equal(t, newDeviceTerm, "1")
	assert.Equal(t, updatedDevice.Protocols[0].ConnectivityState, topodevice.ConnectivityState_REACHABLE)
	assert.Equal(t, updatedDevice.Protocols[0].ChannelState, topodevice.ChannelState_CONNECTED)
	assert.Equal(t, updatedDevice.Protocols[0].ServiceState, topodevice.ServiceState_AVAILABLE)

}
