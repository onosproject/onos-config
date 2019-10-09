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

package network

import (
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/controller"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchanges "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchange "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

const (
	device1 = device.ID("device-1")
	device2 = device.ID("device-2")
	device3 = device.ID("device-3")
	device4 = device.ID("device-4")
)

const (
	network1 = networkchange.ID("network:1")
	network2 = networkchange.ID("network:2")
	network3 = networkchange.ID("network:3")
)

// TestNetworkControllerSuccess verifies that the network controller applies device changes
func TestNetworkControllerSuccess(t *testing.T) {
	leaderships, devices, networkChanges, deviceChanges := newStores(t)
	defer leaderships.Close()
	defer networkChanges.Close()
	defer deviceChanges.Close()

	controller := newController(t, leaderships, devices, networkChanges, deviceChanges)
	defer controller.Stop()

	// Create watches on device and network changes
	deviceCh1 := make(chan *devicechange.Change)
	err := deviceChanges.Watch(device1, deviceCh1)
	assert.NoError(t, err)

	deviceCh2 := make(chan *devicechange.Change)
	err = deviceChanges.Watch(device2, deviceCh2)
	assert.NoError(t, err)

	networkCh := make(chan *networkchange.NetworkChange)
	err = networkChanges.Watch(networkCh)
	assert.NoError(t, err)

	// Create a network change
	change1 := newChange(device1, device2)

	err = networkChanges.Create(change1)
	assert.NoError(t, err)

	// Verify the change is propagated to devices
	deviceChange := nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, devicechange.ID("device-1:1"), deviceChange.ID)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	deviceChange = nextDeviceEvent(t, deviceCh2)
	assert.Equal(t, devicechange.ID("device-2:1"), deviceChange.ID)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	// After the device change have been created, the controller should apply
	// the network change since no changes on the devices are in progress
	networkChange := nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// The controller will also update change information in the parent
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Applying the network change should result in the device changes being applied
	deviceChange = nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, devicechange.ID("device-1:1"), deviceChange.ID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	deviceChange = nextDeviceEvent(t, deviceCh2)
	assert.Equal(t, devicechange.ID("device-2:1"), deviceChange.ID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	// Marking one device change successful should not result in the network change update
	deviceChange, err = deviceChanges.Get("device-2:1")
	assert.NoError(t, err)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	// Check that the network change state has not changed
	networkChange, err = networkChanges.Get("network:1")
	assert.NoError(t, err)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Marking the other device change successful should result in the network change being successful
	deviceChange, err = deviceChanges.Get("device-1:1")
	assert.NoError(t, err)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_SUCCEEDED, networkChange.Status.State)

	networkChange, err = networkChanges.Get("network:1")
	assert.NoError(t, err)
	assert.Equal(t, change.State_SUCCEEDED, networkChange.Status.State)
}

// TestNetworkControllerError verifies that the network controller fails a change when a device change fails
// due to a southbound error
func TestNetworkControllerError(t *testing.T) {
	leaderships, devices, networkChanges, deviceChanges := newStores(t)
	defer leaderships.Close()
	defer networkChanges.Close()
	defer deviceChanges.Close()

	controller := newController(t, leaderships, devices, networkChanges, deviceChanges)
	defer controller.Stop()

	// Create watches on device and network changes
	deviceCh1 := make(chan *devicechange.Change)
	err := deviceChanges.Watch(device1, deviceCh1)
	assert.NoError(t, err)

	deviceCh2 := make(chan *devicechange.Change)
	err = deviceChanges.Watch(device2, deviceCh2)
	assert.NoError(t, err)

	networkCh := make(chan *networkchange.NetworkChange)
	err = networkChanges.Watch(networkCh)
	assert.NoError(t, err)

	// Create a network change
	change1 := newChange(device1, device2)

	err = networkChanges.Create(change1)
	assert.NoError(t, err)

	// Verify the change is propagated to devices
	deviceChange := nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, devicechange.ID("device-1:1"), deviceChange.ID)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	deviceChange = nextDeviceEvent(t, deviceCh2)
	assert.Equal(t, devicechange.ID("device-2:1"), deviceChange.ID)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	// After the device change have been created, the controller should apply
	// the network change since no changes on the devices are in progress
	networkChange := nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// The controller will also update change information in the parent
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Applying the network change should result in the device changes being applied
	deviceChange = nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, devicechange.ID("device-1:1"), deviceChange.ID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	deviceChange = nextDeviceEvent(t, deviceCh2)
	assert.Equal(t, devicechange.ID("device-2:1"), deviceChange.ID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	// Marking one device change successful should not result in the network change update
	deviceChange, err = deviceChanges.Get("device-2:1")
	assert.NoError(t, err)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	// Check that the network change state has not changed
	networkChange, err = networkChanges.Get("network:1")
	assert.NoError(t, err)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Marking the other device change successful should result in the network change being successful
	deviceChange, err = deviceChanges.Get("device-1:1")
	assert.NoError(t, err)
	deviceChange.Status.State = change.State_FAILED
	deviceChange.Status.Reason = change.Reason_ERROR
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_FAILED, networkChange.Status.State)

	networkChange, err = networkChanges.Get("network:1")
	assert.NoError(t, err)
	assert.Equal(t, change.State_FAILED, networkChange.Status.State)
}

// TestNetworkControllerTotallyUnavailable verifies that a change remains in PENDING if devices are unavailable
func TestNetworkControllerTotallyUnavailable(t *testing.T) {

}

// TestNetworkControllerPartiallyUnavailable verifies that a change remains in PENDING if devices are unavailable
func TestNetworkControllerPartiallyUnavailable(t *testing.T) {

}

// TestNetworkControllerPendingConflicts verifies that the network controller serializes network
// changes to intersecting sets of devices
func TestNetworkControllerPendingConflicts(t *testing.T) {
	leaderships, devices, networkChanges, deviceChanges := newStores(t)
	defer leaderships.Close()
	defer networkChanges.Close()
	defer deviceChanges.Close()

	// Prior to starting the controller, enqueue changes with intersecting devices
	change1 := newChange(device1)
	err := networkChanges.Create(change1)
	assert.NoError(t, err)

	change2 := newChange(device1, device2)
	err = networkChanges.Create(change2)
	assert.NoError(t, err)

	change3 := newChange(device3)
	err = networkChanges.Create(change3)
	assert.NoError(t, err)

	// Create watches on device and network changes prior to starting the controller
	deviceCh1 := make(chan *devicechange.Change)
	err = deviceChanges.Watch(device1, deviceCh1)
	assert.NoError(t, err)

	deviceCh2 := make(chan *devicechange.Change)
	err = deviceChanges.Watch(device2, deviceCh2)
	assert.NoError(t, err)

	deviceCh3 := make(chan *devicechange.Change)
	err = deviceChanges.Watch(device3, deviceCh3)
	assert.NoError(t, err)

	networkCh := make(chan *networkchange.NetworkChange)
	err = networkChanges.Watch(networkCh)
	assert.NoError(t, err)

	networkChange := nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change2.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change3.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	controller := newController(t, leaderships, devices, networkChanges, deviceChanges)
	defer controller.Stop()

	// The controller should have processed pending changes once started.
	// This means change 1 should be in the APPLYING state.
	// Changes 2 and 3 should remain in PENDING since change2 intersects change1 and change3 intersects change2.
	// However, change 3 doesn't overlap any of the prior pending changes and thus can be applied.

	// Network 1 is in the PENDING state
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Network 1 is changed to the APPLYING state
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Network 2 is in the PENDING state
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change2.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Network 3 is in the PENDING state
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change3.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Device 1 in network 1 is in the PENDING state
	deviceChange := nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	// Device 1 in network 2 is in the PENDING state
	deviceChange = nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, types.ID(change2.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	// Device 2 in network 2 is in the PENDING state
	deviceChange = nextDeviceEvent(t, deviceCh2)
	assert.Equal(t, types.ID(change2.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	// Device 3 in network 3 is in the PENDING state
	deviceChange = nextDeviceEvent(t, deviceCh3)
	assert.Equal(t, types.ID(change3.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	// Network 3 is changed to the APPLYING state
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change3.ID, networkChange.ID)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Device 1 in network 1 is changed to the APPLYING state
	deviceChange = nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	// Device 3 in network 3 is changed to the APPLYING state
	deviceChange = nextDeviceEvent(t, deviceCh3)
	assert.Equal(t, types.ID(change3.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	// Mark the change 1 device changes as complete and verify the change is complete
	networkChange, err = networkChanges.Get(network1)
	assert.NoError(t, err)

	deviceChange, err = deviceChanges.Get(networkChange.Changes[0].ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	deviceChange = nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_SUCCEEDED, deviceChange.Status.State)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_SUCCEEDED, networkChange.Status.State)

	// Once change 1 is complete, change 2 should be applied
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change2.ID, networkChange.ID)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Device 1 in network 2 is changed to the APPLYING state
	deviceChange = nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, types.ID(change2.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	// Device 2 in network 2 is changed to the APPLYING state
	deviceChange = nextDeviceEvent(t, deviceCh2)
	assert.Equal(t, types.ID(change2.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	// Mark the change 3 device change complete
	networkChange, err = networkChanges.Get(network3)
	assert.NoError(t, err)

	deviceChange, err = deviceChanges.Get(networkChange.Changes[0].ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	deviceChange = nextDeviceEvent(t, deviceCh3)
	assert.Equal(t, types.ID(change3.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_SUCCEEDED, deviceChange.Status.State)

	// Network change 3 should be completed
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change3.ID, networkChange.ID)
	assert.Equal(t, change.State_SUCCEEDED, networkChange.Status.State)

	networkChange, err = networkChanges.Get(network3)
	assert.NoError(t, err)
	assert.Equal(t, change.State_SUCCEEDED, networkChange.Status.State)

	// Mark one of the change 2 device changes complete
	networkChange, err = networkChanges.Get(network2)
	assert.NoError(t, err)

	deviceChange, err = deviceChanges.Get(networkChange.Changes[0].ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	deviceChange = nextDeviceEvent(t, deviceCh1)
	assert.Equal(t, types.ID(change2.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_SUCCEEDED, deviceChange.Status.State)

	// Change 2 should remain in the APPLYING state
	networkChange, err = networkChanges.Get(network2)
	assert.NoError(t, err)
	assert.Equal(t, change2.ID, networkChange.ID)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Mark the other device change complete
	deviceChange, err = deviceChanges.Get(networkChange.Changes[1].ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	deviceChange = nextDeviceEvent(t, deviceCh2)
	assert.Equal(t, types.ID(change2.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_SUCCEEDED, deviceChange.Status.State)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change2.ID, networkChange.ID)
	assert.Equal(t, change.State_SUCCEEDED, networkChange.Status.State)
}

func newStores(t *testing.T) (leadership.Store, devicestore.Store, networkchanges.Store, devicechanges.Store) {
	ctrl := gomock.NewController(t)

	stream := NewMockDeviceService_ListClient(ctrl)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: &device.Device{ID: device1}}, nil)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: &device.Device{ID: device2}}, nil)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: &device.Device{ID: device3}}, nil)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: &device.Device{ID: device4}}, nil)
	stream.EXPECT().Recv().Return(nil, io.EOF)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().List(gomock.Any(), gomock.Any()).Return(stream, nil).AnyTimes()

	devices, err := devicestore.NewStore(client)
	assert.NoError(t, err)

	leadershipStore, err := leadership.NewLocalStore("TestNetworkController", cluster.NodeID("node-1"))
	assert.NoError(t, err)
	assert.NoError(t, err)
	networkChanges, err := networkchanges.NewLocalStore()
	assert.NoError(t, err)
	deviceChanges, err := devicechanges.NewLocalStore()
	assert.NoError(t, err)
	return leadershipStore, devices, networkChanges, deviceChanges
}

func newController(t *testing.T, leadershipStore leadership.Store, devices devicestore.Store, networkChanges networkchanges.Store, deviceChanges devicechanges.Store) *controller.Controller {
	controller := NewController(leadershipStore, devices, networkChanges, deviceChanges)
	err := controller.Start()
	assert.NoError(t, err)
	return controller
}

func newChange(devices ...device.ID) *networkchange.NetworkChange {
	changes := make([]*devicechange.Change, len(devices))
	for i, device := range devices {
		changes[i] = &devicechange.Change{
			DeviceID: device,
			Values: []*devicechange.Value{
				{
					Path:  "foo",
					Value: []byte("Hello world!"),
					Type:  devicechange.ValueType_STRING,
				},
			},
		}
	}
	return &networkchange.NetworkChange{
		Changes: changes,
	}
}

func nextDeviceEvent(t *testing.T, ch chan *devicechange.Change) *devicechange.Change {
	select {
	case e := <-ch:
		return e
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}

func nextNetworkEvent(t *testing.T, ch chan *networkchange.NetworkChange) *networkchange.NetworkChange {
	select {
	case e := <-ch:
		return e
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
