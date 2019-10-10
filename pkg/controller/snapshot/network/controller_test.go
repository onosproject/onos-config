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
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	"github.com/onosproject/onos-config/pkg/types"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchange "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-config/pkg/types/snapshot"
	networksnap "github.com/onosproject/onos-config/pkg/types/snapshot/network"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	device1 = device.ID("device-1")
	device2 = device.ID("device-2")
	device3 = device.ID("device-3")
)

const (
	snapshot1 = networksnap.ID("snapshot:1")
	snapshot2 = networksnap.ID("snapshot:2")
)

// TestReconcilerChangeRollback tests applying and then rolling back a change
func TestReconcilerChangeRollback(t *testing.T) {
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
	networkChange := newNetworkChange(device1)
	err := networkChanges.Create(networkChange)
	assert.NoError(t, err)

	networkChange = newNetworkChange(device1, device2)
	err = networkChanges.Create(networkChange)
	assert.NoError(t, err)

	networkChange = newNetworkChange(device2)
	err = networkChanges.Create(networkChange)
	assert.NoError(t, err)

	networkChange = newNetworkChange(device3)
	err = networkChanges.Create(networkChange)
	assert.NoError(t, err)

	// Create a network snapshot request
	networkSnapshot := &networksnap.NetworkSnapshot{
		Timestamp: time.Now().Add(1 * time.Minute),
	}
	err = networkSnapshots.Create(networkSnapshot)
	assert.NoError(t, err)

	// Reconcile the network snapshot
	ok, err := reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that devices were found when no devices were specified
	networkSnapshot, err = networkSnapshots.Get(snapshot1)
	assert.NoError(t, err)
	assert.True(t, len(networkSnapshot.Devices) > 0)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that device snapshots were created
	deviceSnapshot1, err := deviceSnapshots.Get("device-snapshot:1")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot1.Status.State)
	deviceSnapshot2, err := deviceSnapshots.Get("device-snapshot:2")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot2.Status.State)
	deviceSnapshot3, err := deviceSnapshots.Get("device-snapshot:3")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot3.Status.State)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// The reconciler should have changed its state to RUNNING
	networkSnapshot, err = networkSnapshots.Get(snapshot1)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// But device change states should remain in PENDING state
	deviceSnapshot1, err = deviceSnapshots.Get("device-snapshot:1")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot1.Status.State)
	deviceSnapshot2, err = deviceSnapshots.Get("device-snapshot:2")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot2.Status.State)
	deviceSnapshot3, err = deviceSnapshots.Get("device-snapshot:3")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot3.Status.State)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that device snapshot states were changed to RUNNING
	deviceSnapshot1, err = deviceSnapshots.Get("device-snapshot:1")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, deviceSnapshot1.Status.State)
	deviceSnapshot2, err = deviceSnapshots.Get("device-snapshot:2")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, deviceSnapshot2.Status.State)
	deviceSnapshot3, err = deviceSnapshots.Get("device-snapshot:3")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, deviceSnapshot3.Status.State)

	// Complete one of the snapshots
	deviceSnapshot1.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot1)
	assert.NoError(t, err)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot was not completed
	networkSnapshot, err = networkSnapshots.Get(snapshot1)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// Complete another snapshot
	deviceSnapshot2.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot2)
	assert.NoError(t, err)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot was not completed
	networkSnapshot, err = networkSnapshots.Get(snapshot1)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// Complete the last snapshot
	deviceSnapshot3.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot3)
	assert.NoError(t, err)

	// Reconcile the network snapshot one more time
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot is complete
	networkSnapshot, err = networkSnapshots.Get(snapshot1)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_COMPLETE, networkSnapshot.Status.State)

	// Create another snapshot for a single device
	networkSnapshot = &networksnap.NetworkSnapshot{
		Devices: []*networksnap.DeviceSnapshotRef{
			{
				DeviceID: device1,
			},
		},
		Timestamp: time.Now().Add(1 * time.Minute),
	}
	err = networkSnapshots.Create(networkSnapshot)
	assert.NoError(t, err)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that the device snapshot was created
	deviceSnapshot4, err := deviceSnapshots.Get("device-snapshot:4")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot4.Status.State)

	// Reconcile the network change again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// The reconciler should have changed its state to RUNNING
	networkSnapshot, err = networkSnapshots.Get(snapshot2)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, networkSnapshot.Status.State)

	// But device change states should remain in PENDING state
	deviceSnapshot4, err = deviceSnapshots.Get("device-snapshot:4")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_PENDING, deviceSnapshot4.Status.State)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that device snapshot states were changed to RUNNING
	deviceSnapshot4, err = deviceSnapshots.Get("device-snapshot:4")
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_RUNNING, deviceSnapshot4.Status.State)

	// Complete the snapshot
	deviceSnapshot4.Status.State = snapshot.State_COMPLETE
	err = deviceSnapshots.Update(deviceSnapshot4)
	assert.NoError(t, err)

	// Reconcile the network snapshot again
	ok, err = reconciler.Reconcile(types.ID(networkSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the network snapshot is complete
	networkSnapshot, err = networkSnapshots.Get(snapshot2)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_COMPLETE, networkSnapshot.Status.State)
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

func newNetworkChange(devices ...device.ID) *networkchange.NetworkChange {
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
