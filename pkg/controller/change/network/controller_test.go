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
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchanges "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

const (
	device1 = devicetopo.ID("device-1")
	device2 = devicetopo.ID("device-2")
	device3 = devicetopo.ID("device-3")
	device4 = devicetopo.ID("device-4")
)

const (
	change1 = networkchangetypes.ID("change-1")
)

// TestReconcilerChangeRollback tests applying and then rolling back a change
func TestReconcilerChangeRollback(t *testing.T) {
	_, networkChanges, deviceChanges := newStores(t)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		networkChanges: networkChanges,
		deviceChanges:  deviceChanges,
	}

	// Create a network change
	networkChange := newChange(change1, device1, device2)
	err := networkChanges.Create(networkChange)
	assert.NoError(t, err)

	// Reconcile the network change
	ok, err := reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that device changes were created
	deviceChange1, err := deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err := deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// The reconciler should have changed its state to RUNNING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_RUNNING, networkChange.Status.State)

	// But device change states should remain in PENDING state
	deviceChange1, err = deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that device change states were changed to RUNNING
	deviceChange1, err = deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_RUNNING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_RUNNING, deviceChange2.Status.State)

	// Complete one of the devices
	deviceChange1.Status.State = change.State_COMPLETE
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network change was not completed
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_RUNNING, networkChange.Status.State)

	// Complete the other device
	deviceChange2.Status.State = change.State_COMPLETE
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Reconcile the network change one more time
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network change is complete
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_COMPLETE, networkChange.Status.State)

	// Set the change to the ROLLBACK phase
	networkChange.Status.Phase = change.Phase_ROLLBACK
	networkChange.Status.State = change.State_PENDING
	err = networkChanges.Update(networkChange)
	assert.NoError(t, err)

	// Reconcile the rollback
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that the rollback is running
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Verify that device change phases were changed to ROLLBACK
	deviceChange1, err = deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the rollback again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that the rollback is running
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, networkChange.Status.Phase)
	assert.Equal(t, change.State_RUNNING, networkChange.Status.State)

	// Reconcile the rollback again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that device change states were changed to RUNNING
	deviceChange1, err = deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_RUNNING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_RUNNING, deviceChange2.Status.State)
}

// TestReconcilerError tests an error reverting a change to PENDING
func TestReconcilerError(t *testing.T) {
	_, networkChanges, deviceChanges := newStores(t)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		networkChanges: networkChanges,
		deviceChanges:  deviceChanges,
	}

	// Create a network change
	networkChange := newChange(change1, device1, device2)
	err := networkChanges.Create(networkChange)
	assert.NoError(t, err)

	// Reconcile the network change
	ok, err := reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that device changes were created
	deviceChange1, err := deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err := deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// The reconciler should have changed its state to RUNNING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_RUNNING, networkChange.Status.State)

	// But device change states should remain in PENDING state
	deviceChange1, err = deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that device change states were changed to RUNNING
	deviceChange1, err = deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_RUNNING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_RUNNING, deviceChange2.Status.State)

	// Complete one of the devices
	deviceChange1.Status.State = change.State_COMPLETE
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network change was not completed
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_RUNNING, networkChange.Status.State)

	// Fail the other device
	deviceChange2, err = deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	deviceChange2.Status.State = change.State_FAILED
	deviceChange2.Status.Reason = change.Reason_ERROR
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network change is still RUNNING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_RUNNING, networkChange.Status.State)

	// Verify the change to device-1 is being rolled back
	deviceChange1, err = deviceChanges.Get("change-1:device-1")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_RUNNING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_FAILED, deviceChange2.Status.State)

	// Set the device-1 change rollback to COMPLETE
	deviceChange1.Status.State = change.State_COMPLETE
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Reconcile the network change
	ok, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that the network change returned to PENDING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)
}

func newStores(t *testing.T) (devicestore.Store, networkchanges.Store, devicechanges.Store) {
	ctrl := gomock.NewController(t)

	stream := NewMockDeviceService_ListClient(ctrl)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: &devicetopo.Device{ID: device1}}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: &devicetopo.Device{ID: device2}}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: &devicetopo.Device{ID: device3}}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: &devicetopo.Device{ID: device4}}, nil)
	stream.EXPECT().Recv().Return(nil, io.EOF)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().List(gomock.Any(), gomock.Any()).Return(stream, nil).AnyTimes()

	devices, err := devicestore.NewStore(client)
	assert.NoError(t, err)
	networkChanges, err := networkchanges.NewLocalStore()
	assert.NoError(t, err)
	deviceChanges, err := devicechanges.NewLocalStore()
	assert.NoError(t, err)
	return devices, networkChanges, deviceChanges
}

func newChange(id networkchangetypes.ID, devices ...devicetopo.ID) *networkchangetypes.NetworkChange {
	changes := make([]*devicechangetypes.Change, len(devices))
	for i, device := range devices {
		changes[i] = &devicechangetypes.Change{
			DeviceID: device,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path: "foo",
					Value: &devicechangetypes.TypedValue{
						Bytes: []byte("Hello world!"),
						Type:  devicechangetypes.ValueType_STRING,
					},
				},
			},
		}
	}
	return &networkchangetypes.NetworkChange{
		ID:      id,
		Changes: changes,
	}
}
