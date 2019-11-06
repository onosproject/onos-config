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
	"fmt"
	"github.com/onosproject/onos-config/api/types"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/api/types/device"
	snapshottype "github.com/onosproject/onos-config/api/types/snapshot"
	devicesnapshot "github.com/onosproject/onos-config/api/types/snapshot/device"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	device1 = device.ID("device-1")
)

func TestReconcileDeviceSnapshotIndex(t *testing.T) {
	changes, snapshots := newStores(t)
	defer changes.Close()
	defer snapshots.Close()

	reconciler := &Reconciler{
		changes:   changes,
		snapshots: snapshots,
	}

	// Create a device-1 change 1
	deviceChange1 := newSet(1, device1, "foo", time.Now(), changetypes.Phase_CHANGE, changetypes.State_COMPLETE)
	err := changes.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-1 change 2
	deviceChange2 := newSet(2, device1, "bar", time.Now(), changetypes.Phase_CHANGE, changetypes.State_COMPLETE)
	err = changes.Create(deviceChange2)
	assert.NoError(t, err)

	// Create a device-1 change 4
	deviceChange3 := newRemove(4, device1, "foo", time.Now(), changetypes.Phase_CHANGE, changetypes.State_COMPLETE)
	err = changes.Create(deviceChange3)
	assert.NoError(t, err)

	// Create a device-1 change 5
	deviceChange4 := newSet(5, device1, "foo", time.Now(), changetypes.Phase_CHANGE, changetypes.State_COMPLETE)
	err = changes.Create(deviceChange4)
	assert.NoError(t, err)

	// Create a device snapshot
	deviceSnapshot := &devicesnapshot.DeviceSnapshot{
		DeviceID:              device1,
		DeviceVersion:         "1.0.0",
		MaxNetworkChangeIndex: 4,
	}
	err = snapshots.Create(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err := reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the snapshot was not changed
	revision := deviceSnapshot.Revision
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, revision, deviceSnapshot.Revision)

	// Set the snapshot state to RUNNING
	deviceSnapshot.Status.State = snapshottype.State_RUNNING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the snapshot was set to COMPLETE
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshottype.State_COMPLETE, deviceSnapshot.Status.State)

	// Verify the correct snapshot was taken
	snapshot, err := snapshots.Load(deviceSnapshot.GetVersionedDeviceID())
	assert.NoError(t, err)
	assert.Equal(t, devicechange.Index(3), snapshot.ChangeIndex)
	assert.Len(t, snapshot.Values, 2)
	for _, value := range snapshot.Values {
		switch value.GetPath() {
		case "/bar/msg":
			assert.Equal(t, "Hello world 3", value.GetValue().ValueToString())
		case "/bar/meaning":
			assert.Equal(t, "42", value.GetValue().ValueToString())
		default:
			t.Error("Unexpected value", value.GetPath())
		}
	}

	// Verify changes have not been deleted
	deviceChange1, err = changes.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange1)

	// Set the snapshot phase to DELETE
	deviceSnapshot.Status.Phase = snapshottype.Phase_DELETE
	deviceSnapshot.Status.State = snapshottype.State_PENDING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify changes have not been deleted again
	deviceChange1, err = changes.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange1)

	// Set the snapshot phase to RUNNING
	deviceSnapshot.Status.State = snapshottype.State_RUNNING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify changes have been deleted
	deviceChange1, err = changes.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.Nil(t, deviceChange1)
	deviceChange2, err = changes.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Nil(t, deviceChange2)
	deviceChange3, err = changes.Get(deviceChange3.ID)
	assert.NoError(t, err)
	assert.Nil(t, deviceChange3)

	// Verify the snapshot state is COMPLETE
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshottype.State_COMPLETE, deviceSnapshot.Status.State)
}

func TestReconcileDeviceSnapshotPhaseState(t *testing.T) {
	changes, snapshots := newStores(t)
	defer changes.Close()
	defer snapshots.Close()

	reconciler := &Reconciler{
		changes:   changes,
		snapshots: snapshots,
	}

	// Create a device-1 change 1
	deviceChange1 := newSet(1, device1, "foo", time.Now(), changetypes.Phase_CHANGE, changetypes.State_COMPLETE)
	err := changes.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-1 change 2
	deviceChange2 := newSet(2, device1, "bar", time.Now(), changetypes.Phase_CHANGE, changetypes.State_COMPLETE)
	err = changes.Create(deviceChange2)
	assert.NoError(t, err)

	// Create a device-1 change 4
	deviceChange3 := newRemove(3, device1, "foo", time.Now(), changetypes.Phase_ROLLBACK, changetypes.State_COMPLETE)
	err = changes.Create(deviceChange3)
	assert.NoError(t, err)

	// Create a device snapshot
	deviceSnapshot := &devicesnapshot.DeviceSnapshot{
		DeviceID:              device1,
		DeviceVersion:         "1.0.0",
		MaxNetworkChangeIndex: 3,
	}
	err = snapshots.Create(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err := reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the snapshot was not changed
	revision := deviceSnapshot.Revision
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, revision, deviceSnapshot.Revision)

	// Set the snapshot state to RUNNING
	deviceSnapshot.Status.State = snapshottype.State_RUNNING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify the snapshot was set to COMPLETE
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshottype.State_COMPLETE, deviceSnapshot.Status.State)

	// Verify the correct snapshot was taken
	snapshot, err := snapshots.Load(deviceSnapshot.GetVersionedDeviceID())
	assert.NoError(t, err)
	assert.Equal(t, devicechange.Index(3), snapshot.ChangeIndex)
	assert.Len(t, snapshot.Values, 4)

	// Verify changes have not been deleted
	deviceChange1, err = changes.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange1)

	// Set the snapshot phase to DELETE
	deviceSnapshot.Status.Phase = snapshottype.Phase_DELETE
	deviceSnapshot.Status.State = snapshottype.State_PENDING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify changes have not been deleted again
	deviceChange1, err = changes.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange1)

	// Set the snapshot phase to RUNNING
	deviceSnapshot.Status.State = snapshottype.State_RUNNING
	err = snapshots.Update(deviceSnapshot)
	assert.NoError(t, err)

	// Reconcile the snapshot
	ok, err = reconciler.Reconcile(types.ID(deviceSnapshot.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Verify changes have been deleted
	deviceChange1, err = changes.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.Nil(t, deviceChange1)
	deviceChange2, err = changes.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Nil(t, deviceChange2)
	deviceChange3, err = changes.Get(deviceChange3.ID)
	assert.NoError(t, err)
	assert.Nil(t, deviceChange3)

	// Verify the snapshot state is COMPLETE
	deviceSnapshot, err = snapshots.Get(deviceSnapshot.ID)
	assert.NoError(t, err)
	assert.Equal(t, snapshottype.State_COMPLETE, deviceSnapshot.Status.State)
}

func newStores(t *testing.T) (devicechangestore.Store, devicesnapstore.Store) {
	changes, err := devicechangestore.NewLocalStore()
	assert.NoError(t, err)
	snapshots, err := devicesnapstore.NewLocalStore()
	assert.NoError(t, err)
	return changes, snapshots
}

func newSet(index networkchange.Index, device device.ID, path string, created time.Time, phase changetypes.Phase, state changetypes.State) *devicechange.DeviceChange {
	return newChange(index, created, phase, state, &devicechange.Change{
		DeviceID:      device,
		DeviceVersion: "1.0.0",
		DeviceType:    "Stratum",
		Values: []*devicechange.ChangeValue{
			{
				Path:  fmt.Sprintf("/%s/msg", path),
				Value: devicechange.NewTypedValueString(fmt.Sprintf("Hello world %d", len(path))),
			},
			{
				Path:  fmt.Sprintf("/%s/meaning", path),
				Value: devicechange.NewTypedValueInt64(39 + len(path)),
			},
		},
	})
}

func newRemove(index networkchange.Index, device device.ID, path string, created time.Time, phase changetypes.Phase, state changetypes.State) *devicechange.DeviceChange {
	return newChange(index, created, phase, state, &devicechange.Change{
		DeviceID:      device,
		DeviceVersion: "1.0.0",
		DeviceType:    "Stratum",
		Values: []*devicechange.ChangeValue{
			{
				Path:    fmt.Sprintf("/%s", path),
				Removed: true,
			},
		},
	})
}

func newChange(index networkchange.Index, created time.Time, phase changetypes.Phase, state changetypes.State, change *devicechange.Change) *devicechange.DeviceChange {
	return &devicechange.DeviceChange{
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("network-change-%d", index)),
			Index: types.Index(index),
		},
		Change: change,
		Status: changetypes.Status{
			Phase: phase,
			State: state,
		},
		Created: created,
	}
}
