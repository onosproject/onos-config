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
	"github.com/onosproject/onos-api/go/onos/topo"
	"testing"

	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/test/mocks"
	"github.com/onosproject/onos-lib-go/pkg/cluster"

	"github.com/golang/mock/gomock"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"gotest.tools/assert"
)

const (
	device1        = "device1"
	deviceVersion1 = "1.0.0"
)

type AllMocks struct {
	DeviceStore     devicestore.Store
	MastershipStore mastership.Store
	TopoClient      *mocks.MockTopoClient
}

func setUp(name string, t *testing.T) *AllMocks {

	ctrl := gomock.NewController(t)
	client := mocks.NewMockTopoClient(ctrl)
	deviceStore, err := devicestore.NewStore(client)
	assert.NilError(t, err)

	node1 := cluster.NodeID("node1")
	mastershipStore, err := mastership.NewLocalStore(name, node1)
	assert.NilError(t, err)

	allMocks := AllMocks{
		DeviceStore:     deviceStore,
		MastershipStore: mastershipStore,
		TopoClient:      client,
	}

	return &allMocks
}

func TestUpdateDisconnectedDevice(t *testing.T) {
	allMocks := setUp("TestUpdateDisconnectedDevice", t)

	device1Disconnected := &topodevice.Device{
		ID:         device1,
		Revision:   1,
		Address:    "device1:1234",
		Version:    deviceVersion1,
		Attributes: make(map[string]string),
	}

	allMocks.TopoClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&topo.GetResponse{Object: topodevice.ToObject(device1Disconnected)}, nil).AnyTimes()

	protocolState := new(topo.ProtocolState)
	protocolState.Protocol = topo.Protocol_GNMI
	device1Disconnected.Protocols = append(device1Disconnected.Protocols, protocolState)
	device1Disconnected.Attributes[mastershipTermKey] = "0"

	allMocks.TopoClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&topo.UpdateResponse{Object: topodevice.ToObject(device1Disconnected)}, nil).AnyTimes()

	state, err := allMocks.MastershipStore.GetMastership(device1Disconnected.ID)
	assert.NilError(t, err)

	session1 := &Session{
		device:          device1Disconnected,
		deviceStore:     allMocks.DeviceStore,
		mastershipState: state,
	}

	err = session1.updateDisconnectedDevice()
	assert.NilError(t, err)
	updatedDevice, err := session1.deviceStore.Get(device1)
	assert.NilError(t, err)
	newDeviceTerm := updatedDevice.Attributes[mastershipTermKey]
	assert.Equal(t, newDeviceTerm, "1")
	assert.Equal(t, updatedDevice.Protocols[0].ConnectivityState, topo.ConnectivityState_UNREACHABLE)
	assert.Equal(t, updatedDevice.Protocols[0].ChannelState, topo.ChannelState_DISCONNECTED)
	assert.Equal(t, updatedDevice.Protocols[0].ServiceState, topo.ServiceState_UNAVAILABLE)
}

func TestUpdateConnectedDevice(t *testing.T) {
	allMocks := setUp("TestUpdateConnectedDevice", t)

	device1Connected := &topodevice.Device{
		ID:         device1,
		Revision:   1,
		Address:    "device1:1234",
		Version:    deviceVersion1,
		Attributes: make(map[string]string),
	}

	allMocks.TopoClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&topo.GetResponse{Object: topodevice.ToObject(device1Connected)}, nil).AnyTimes()

	protocolState := new(topo.ProtocolState)
	protocolState.Protocol = topo.Protocol_GNMI
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)
	device1Connected.Attributes[mastershipTermKey] = "0"

	allMocks.TopoClient.EXPECT().Update(gomock.Any(), gomock.Any()).Return(&topo.UpdateResponse{Object: topodevice.ToObject(device1Connected)}, nil).AnyTimes()

	state, err := allMocks.MastershipStore.GetMastership(device1Connected.ID)
	assert.NilError(t, err)

	session1 := &Session{
		device:          device1Connected,
		deviceStore:     allMocks.DeviceStore,
		mastershipState: state,
	}

	err = session1.updateConnectedDevice()
	assert.NilError(t, err)
	updatedDevice, err := session1.deviceStore.Get(device1)
	assert.NilError(t, err)
	newDeviceTerm := updatedDevice.Attributes[mastershipTermKey]
	assert.Equal(t, newDeviceTerm, "1")
	assert.Equal(t, updatedDevice.Protocols[0].ConnectivityState, topo.ConnectivityState_REACHABLE)
	assert.Equal(t, updatedDevice.Protocols[0].ChannelState, topo.ChannelState_CONNECTED)
	assert.Equal(t, updatedDevice.Protocols[0].ServiceState, topo.ServiceState_AVAILABLE)

}
