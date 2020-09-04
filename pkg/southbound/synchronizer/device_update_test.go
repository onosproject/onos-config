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

	"github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-lib-go/pkg/cluster"

	"github.com/golang/mock/gomock"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"

	topodevice "github.com/onosproject/onos-topo/api/device"
	"gotest.tools/assert"
)

type AllMocks struct {
	MockStores *mockstore.MockStores
}

const (
	device1        = "device1"
	deviceVersion1 = "1.0.0"
)

func setUp(t *testing.T) *AllMocks {
	var (
		allMocks AllMocks
	)

	ctrl := gomock.NewController(t)

	// Mock Mastership Store
	mockMastershipStore := mockstore.NewMockMastershipStore(ctrl)
	mockMastershipStore.EXPECT().Watch(gomock.Any(), gomock.Any()).AnyTimes()

	// Mock Device Store
	mockDeviceStore := mockstore.NewMockDeviceStore(ctrl)
	mockDeviceStore.EXPECT().Watch(gomock.Any()).AnyTimes()

	mockStores := &mockstore.MockStores{
		DeviceStore:     mockDeviceStore,
		MastershipStore: mockMastershipStore,
	}

	allMocks.MockStores = mockStores

	return &allMocks
}

func TestUpdateDisconnectedDevice(t *testing.T) {
	mocks := setUp(t)

	device1Disconnected := &topodevice.Device{
		ID:         device1,
		Revision:   1,
		Address:    "device1:1234",
		Version:    deviceVersion1,
		Attributes: make(map[string]string),
	}

	mocks.MockStores.DeviceStore.EXPECT().Get(device1)

	protocolState := new(topodevice.ProtocolState)
	protocolState.Protocol = topodevice.Protocol_GNMI
	device1Disconnected.Protocols = append(device1Disconnected.Protocols, protocolState)
	device1Disconnected.Attributes[mastershipTermKey] = "1"

	mocks.MockStores.DeviceStore.EXPECT().Update(gomock.Any()).Return(device1Disconnected, nil)
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(device1Disconnected, nil)

	mastershipState := mastership.Mastership{
		Device: device1Disconnected.ID,
		Term:   2,
		Master: cluster.NodeID("node1"),
	}
	mocks.MockStores.MastershipStore.EXPECT().GetMastership(device1Disconnected.ID).Return(&mastershipState, nil)

	session := &Session{
		device:          device1Disconnected,
		mastershipStore: mocks.MockStores.MastershipStore,
		deviceStore:     mocks.MockStores.DeviceStore,
	}

	err := session.updateDisconnectedDevice()
	assert.NilError(t, err)
}

func TestUpdateConnectedDevice(t *testing.T) {
	mocks := setUp(t)

	device1Connected := &topodevice.Device{
		ID:         device1,
		Revision:   1,
		Address:    "device1:1234",
		Version:    deviceVersion1,
		Attributes: make(map[string]string),
	}

	mocks.MockStores.DeviceStore.EXPECT().Get(device1)

	protocolState := new(topodevice.ProtocolState)
	protocolState.Protocol = topodevice.Protocol_GNMI
	device1Connected.Protocols = append(device1Connected.Protocols, protocolState)
	device1Connected.Attributes[mastershipTermKey] = "1"

	mocks.MockStores.DeviceStore.EXPECT().Update(gomock.Any()).Return(device1Connected, nil)
	mocks.MockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(device1Connected, nil)

	mastershipState := mastership.Mastership{
		Device: device1Connected.ID,
		Term:   2,
		Master: cluster.NodeID("node1"),
	}
	mocks.MockStores.MastershipStore.EXPECT().GetMastership(device1Connected.ID).Return(&mastershipState, nil)

	session := &Session{
		device:          device1Connected,
		mastershipStore: mocks.MockStores.MastershipStore,
		deviceStore:     mocks.MockStores.DeviceStore,
	}

	err := session.updateConnectedDevice()
	assert.NilError(t, err)
}
