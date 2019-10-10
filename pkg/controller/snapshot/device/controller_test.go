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
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/types/snapshot"
	devicesnap "github.com/onosproject/onos-config/pkg/types/snapshot/device"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	device1 = device.ID("device-1")
	device2 = device.ID("device-2")
)

func TestReconcilerChangeSuccess(t *testing.T) {
	changes, snapshots := newStores(t)
	defer changes.Close()
	defer snapshots.Close()

	reconciler := &Reconciler{
		changes:   changes,
		snapshots: snapshots,
	}

	// Create a device-1 change 1
	deviceChange1 := newChange(device1)
	err := changes.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-2 change 1
	deviceChange2 := newChange(device2)
	err = changes.Create(deviceChange2)
	assert.NoError(t, err)

	// Create a device snapshot
	deviceSnapshot := &devicesnap.DeviceSnapshot{
		Timestamp: time.Now().Add(1 * time.Hour * -1),
	}
	err = snapshots.Create(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err := reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// No snapshots for the devices should exist
	snapshot1, err := snapshots.Load(devicesnap.ID(device1))
	assert.NoError(t, err)
	assert.Nil(t, snapshot1)
	snapshot2, err := snapshots.Load(devicesnap.ID(device2))
	assert.NoError(t, err)
	assert.Nil(t, snapshot2)

	// Set the snapshot state to RUNNING
	deviceSnapshot.Status.State = snapshot.State_RUNNING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot again
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that the snapshot state is COMPLETE
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_COMPLETE, deviceSnapshot.Status.State)

	// Verify that no snapshot was created for either device since no changes were present prior to the snapshot time
	snapshot1, err = snapshots.Load(devicesnap.ID(device1))
	assert.NoError(t, err)
	assert.Nil(t, snapshot1)
	snapshot2, err = snapshots.Load(devicesnap.ID(device2))
	assert.NoError(t, err)
	assert.Nil(t, snapshot2)

	// Create another snapshot with a timestamp greater than the change timestamps
	deviceSnapshot = &devicesnap.DeviceSnapshot{
		Timestamp: time.Now().Add(1 * time.Hour),
	}
	err = snapshots.Create(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// No snapshots for the devices should exist
	snapshot1, err = snapshots.Load(devicesnap.ID(device1))
	assert.NoError(t, err)
	assert.Nil(t, snapshot1)
	snapshot2, err = snapshots.Load(devicesnap.ID(device2))
	assert.NoError(t, err)
	assert.Nil(t, snapshot2)

	// Set the snapshot state to RUNNING
	deviceSnapshot.Status.State = snapshot.State_RUNNING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot again
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that the snapshot state is COMPLETE
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_COMPLETE, deviceSnapshot.Status.State)

	// Verify that no snapshot was created for either device since the changes had not been applied
	snapshot1, err = snapshots.Load(devicesnap.ID(device1))
	assert.NoError(t, err)
	assert.Nil(t, snapshot1)
	snapshot2, err = snapshots.Load(devicesnap.ID(device2))
	assert.NoError(t, err)
	assert.Nil(t, snapshot2)

	// Update the change states to COMPLETE
	deviceChange1.Status.State = change.State_COMPLETE
	err = changes.Update(deviceChange1)
	assert.NoError(t, err)
	deviceChange2.Status.State = change.State_COMPLETE
	err = changes.Update(deviceChange2)
	assert.NoError(t, err)

	// Create one more snapshot with a timestamp greater than the change timestamps
	deviceSnapshot = &devicesnap.DeviceSnapshot{
		Timestamp: time.Now().Add(1 * time.Hour),
	}
	err = snapshots.Create(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// No snapshots for the devices should exist
	snapshot1, err = snapshots.Load(devicesnap.ID(device1))
	assert.NoError(t, err)
	assert.Nil(t, snapshot1)
	snapshot2, err = snapshots.Load(devicesnap.ID(device2))
	assert.NoError(t, err)
	assert.Nil(t, snapshot2)

	// Set the snapshot state to RUNNING
	deviceSnapshot.Status.State = snapshot.State_RUNNING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot again
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify that the snapshot state is COMPLETE
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshot.State_COMPLETE, deviceSnapshot.Status.State)

	// Verify that no snapshot was created for either device since the changes had not been applied
	snapshot1, err = snapshots.Load(devicesnap.ID(device1))
	assert.NoError(t, err)
	assert.Nil(t, snapshot1)
	snapshot2, err = snapshots.Load(devicesnap.ID(device2))
	assert.NoError(t, err)
	assert.Nil(t, snapshot2)
}

func newStores(t *testing.T) (devicechangestore.Store, devicesnapstore.Store) {
	changes, err := devicechangestore.NewLocalStore()
	assert.NoError(t, err)
	snapshots, err := devicesnapstore.NewLocalStore()
	assert.NoError(t, err)
	return changes, snapshots
}

func newChange(device device.ID) *devicechange.Change {
	return &devicechange.Change{
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
