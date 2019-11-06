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
	"github.com/onosproject/onos-config/api/types"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/api/types/snapshot"
	devicesnapshot "github.com/onosproject/onos-config/api/types/snapshot/device"
	networksnapshot "github.com/onosproject/onos-config/api/types/snapshot/network"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	device1 = device.ID("device-1")
	device2 = device.ID("device-2")
	device3 = device.ID("device-3")
)

func TestReconcileNetworkSnapshotPhaseState(t *testing.T) {
	networkChanges, networkSnapshots, deviceSnapshots := newStores(t)
	defer networkChanges.Close()
	defer networkSnapshots.Close()
	defer deviceSnapshots.Close()

	reconciler := &Reconciler{
		networkChanges:   networkChanges,
		networkSnapshots: networkSnapshots,
		deviceSnapshots:  deviceSnapshots,
	}

	// Create network and device changes in the completed state
	networkChange1 := newNetworkChange("change-1", changetypes.Phase_CHANGE, changetypes.State_COMPLETE, device1)
	err := networkChanges.Create(networkChange1)
	assert.NoError(t, err)

	networkChange2 := newNetworkChange("change-2", changetypes.Phase_CHANGE, changetypes.State_PENDING, device1, device2)
	err = networkChanges.Create(networkChange2)
	assert.NoError(t, err)

	networkChange3 := newNetworkChange("change-3", changetypes.Phase_CHANGE, changetypes.State_PENDING, device2)
	err = networkChanges.Create(networkChange3)
	assert.NoError(t, err)

	networkChange4 := newNetworkChange("change-4", changetypes.Phase_CHANGE, changetypes.State_COMPLETE, device3)
	err = networkChanges.Create(networkChange4)
	assert.NoError(t, err)

	// Create a network snapshot request
	networkSnapshot := &networksnapshot.NetworkSnapshot{}
	err = networkSnapshots.Create(networkSnapshot)
	assert.NoError(t, err)

	// Reconcile the network snapshot
	ok, err := reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that no device snapshots were created
	deviceSnapshot1, err := deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device1, "1.0.0"))
	assert.NoError(t, err)
	assert.Nil(t, deviceSnapshot1)
	deviceSnapshot2, err := deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device2, "1.0.0"))
	assert.NoError(t, err)
	assert.Nil(t, deviceSnapshot2)
	deviceSnapshot3, err := deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device3, "1.0.0"))
	assert.NoError(t, err)
	assert.Nil(t, deviceSnapshot3)

	// Verify the network snapshot state is RUNNING
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify network changes were marked for deletion
	networkChange1, err = networkChanges.Get(networkChange1.ID)
	assert.NoError(t, err)
	assert.True(t, networkChange1.Deleted)
	networkChange2, err = networkChanges.Get(networkChange2.ID)
	assert.NoError(t, err)
	assert.False(t, networkChange2.Deleted)
	networkChange3, err = networkChanges.Get(networkChange3.ID)
	assert.NoError(t, err)
	assert.False(t, networkChange3.Deleted)
	networkChange4, err = networkChanges.Get(networkChange4.ID)
	assert.NoError(t, err)
	assert.True(t, networkChange4.Deleted)

	// Verify device snapshots were created
	deviceSnapshot1, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device1, "1.0.0"))
	assert.NoError(t, err)
	assert.NotNil(t, deviceSnapshot1)
	deviceSnapshot2, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device2, "1.0.0"))
	assert.NoError(t, err)
	assert.NotNil(t, deviceSnapshot2)
	deviceSnapshot3, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device3, "1.0.0"))
	assert.NoError(t, err)
	assert.NotNil(t, deviceSnapshot3)

	// Verify the network snapshot state is not yet COMPLETE but device snapshot refs have been created
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)
	assert.Len(t, networkSnapshot.Refs, 3)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot state is still RUNNING
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// Set a device snapshot to COMPLETE
	deviceSnapshot1.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot1)
	assert.NoError(t, err)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot state is still RUNNING
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// Complete the remaining device snapshots
	deviceSnapshot2.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot2)
	assert.NoError(t, err)
	deviceSnapshot3.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot3)
	assert.NoError(t, err)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot state is PENDING in the DELETE phase
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, networkSnapshot.Status.Phase)
	assert.Equal(t, snapshot.State_PENDING, networkSnapshot.Status.State)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the device snapshots are PENDING in the DELETE phase
	deviceSnapshot1, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device1, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot1.Status.Phase)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot1.Status.State)
	deviceSnapshot2, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device2, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot2.Status.Phase)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot2.Status.State)
	deviceSnapshot3, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device3, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot3.Status.Phase)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot3.Status.State)

	// Verify the network snapshot state is RUNNING
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, networkSnapshot.Status.Phase)
	assert.Equal(t, snapshot.State_PENDING, networkSnapshot.Status.State)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the device snapshots are RUNNING in the DELETE phase
	deviceSnapshot1, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device1, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot1.Status.Phase)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot1.Status.State)
	deviceSnapshot2, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device2, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot2.Status.Phase)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot2.Status.State)
	deviceSnapshot3, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device3, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot3.Status.Phase)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot3.Status.State)

	// Verify the network snapshot state is RUNNING
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, networkSnapshot.Status.Phase)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the device snapshots are RUNNING in the DELETE phase
	deviceSnapshot1, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device1, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot1.Status.Phase)
	assert.Equal(t, snapshot.State_RUNNING, deviceSnapshot1.Status.State)
	deviceSnapshot2, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device2, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot2.Status.Phase)
	assert.Equal(t, snapshot.State_RUNNING, deviceSnapshot2.Status.State)
	deviceSnapshot3, err = deviceSnapshots.Get(devicesnapshot.GetSnapshotID(types.ID(networkSnapshot.ID), device3, "1.0.0"))
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, deviceSnapshot3.Status.Phase)
	assert.Equal(t, snapshot.State_RUNNING, deviceSnapshot3.Status.State)

	// Verify the network snapshot state is RUNNING
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, networkSnapshot.Status.Phase)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// Set a device snapshot to COMPLETE
	deviceSnapshot1.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot1)
	assert.NoError(t, err)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot state is still RUNNING
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// Complete the remaining device snapshots
	deviceSnapshot2.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot2)
	assert.NoError(t, err)
	deviceSnapshot3.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot3)
	assert.NoError(t, err)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot is COMPLETE
	networkSnapshot, err = networkSnapshots.Get(networkSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.Phase_DELETE, networkSnapshot.Status.Phase)
	assert.Equal(t, snapshot.State_COMPLETE, networkSnapshot.Status.State)

	// Verify network changes were deleted
	networkChange1, err = networkChanges.Get(networkChange1.ID)
	assert.NoError(t, err)
	assert.Nil(t, networkChange1)
	networkChange2, err = networkChanges.Get(networkChange2.ID)
	assert.NoError(t, err)
	assert.NotNil(t, networkChange2)
	networkChange3, err = networkChanges.Get(networkChange3.ID)
	assert.NoError(t, err)
	assert.NotNil(t, networkChange3)
	networkChange4, err = networkChanges.Get(networkChange4.ID)
	assert.NoError(t, err)
	assert.Nil(t, networkChange4)
}

func newStores(t *testing.T) (networkchangestore.Store, networksnapstore.Store, devicesnapstore.Store) {
	networkChanges, err := networkchangestore.NewLocalStore()
	assert.NoError(t, err)
	networkSnapshots, err := networksnapstore.NewLocalStore()
	assert.NoError(t, err)
	deviceSnapshots, err := devicesnapstore.NewLocalStore()
	assert.NoError(t, err)
	return networkChanges, networkSnapshots, deviceSnapshots
}

func newNetworkChange(id networkchange.ID, phase changetypes.Phase, state changetypes.State, devices ...device.ID) *networkchange.NetworkChange {
	changes := make([]*devicechange.Change, len(devices))
	for i, device := range devices {
		changes[i] = &devicechange.Change{
			DeviceID: device,
			Values: []*devicechange.ChangeValue{
				{
					Path: "foo",
					Value: &devicechange.TypedValue{
						Bytes: []byte("Hello world!"),
						Type:  devicechange.ValueType_STRING,
					},
				},
			},
		}
	}

	refs := make([]*networkchange.DeviceChangeRef, len(devices))
	for i, device := range devices {
		refs[i] = &networkchange.DeviceChangeRef{
			DeviceChangeID: devicechange.NewID(types.ID(id), device, "1.0.0"),
		}
	}
	return &networkchange.NetworkChange{
		ID:      id,
		Changes: changes,
		Refs:    refs,
		Status: changetypes.Status{
			Phase: phase,
			State: state,
		},
	}
}
