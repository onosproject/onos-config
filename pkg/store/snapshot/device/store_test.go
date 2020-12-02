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
	"github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-api/go/onos/config/snapshot"
	devicesnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-lib-go/pkg/atomix"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDeviceSnapshotStore(t *testing.T) {
	_, address := atomix.StartLocalNode()

	store1, err := newLocalStore(address)
	assert.NoError(t, err)
	defer store1.Close()

	store2, err := newLocalStore(address)
	assert.NoError(t, err)
	defer store2.Close()

	device1 := device.ID("device-1")
	device2 := device.ID("device-2")

	ch := make(chan stream.Event)
	_, err = store2.Watch(ch)
	assert.NoError(t, err)

	snapshot1 := &devicesnapshot.DeviceSnapshot{
		DeviceID:      device1,
		DeviceVersion: "1.0.0",
		NetworkSnapshot: devicesnapshot.NetworkSnapshotRef{
			ID:    "snapshot-1",
			Index: 1,
		},
	}

	snapshot2 := &devicesnapshot.DeviceSnapshot{
		DeviceID:      device2,
		DeviceVersion: "1.0.0",
		NetworkSnapshot: devicesnapshot.NetworkSnapshotRef{
			ID:    "snapshot-2",
			Index: 2,
		},
	}

	// Create a new snapshot
	err = store1.Create(snapshot1)
	assert.NoError(t, err)
	assert.Equal(t, devicesnapshot.ID("snapshot-1:device-1:1.0.0"), snapshot1.ID)
	assert.NotEqual(t, devicesnapshot.Revision(0), snapshot1.Revision)

	// Get the snapshot
	snapshot1, err = store2.Get("snapshot-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, snapshot1)
	assert.Equal(t, devicesnapshot.ID("snapshot-1:device-1:1.0.0"), snapshot1.ID)
	assert.NotEqual(t, devicesnapshot.Revision(0), snapshot1.Revision)

	// Create another snapshot
	err = store2.Create(snapshot2)
	assert.NoError(t, err)
	assert.Equal(t, devicesnapshot.ID("snapshot-2:device-2:1.0.0"), snapshot2.ID)
	assert.NotEqual(t, devicesnapshot.Revision(0), snapshot2.Revision)

	// Verify events were received for the snapshots
	snapshotEvent := nextEvent(t, ch)
	assert.Equal(t, devicesnapshot.ID("snapshot-1:device-1:1.0.0"), snapshotEvent.ID)
	snapshotEvent = nextEvent(t, ch)
	assert.Equal(t, devicesnapshot.ID("snapshot-2:device-2:1.0.0"), snapshotEvent.ID)

	// Update one of the snapshots
	snapshot2.Status.State = snapshot.State_RUNNING
	revision := snapshot2.Revision
	err = store1.Update(snapshot2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, snapshot2.Revision)

	// Read and then update the snapshot
	snapshot2, err = store2.Get("snapshot-2:device-2:1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, snapshot2)
	snapshot2.Status.State = snapshot.State_COMPLETE
	revision = snapshot2.Revision
	err = store1.Update(snapshot2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, snapshot2.Revision)

	// Verify that concurrent updates fail
	snapshot11, err := store1.Get("snapshot-1:device-1:1.0.0")
	assert.NoError(t, err)
	snapshot12, err := store2.Get("snapshot-1:device-1:1.0.0")
	assert.NoError(t, err)

	snapshot11.Status.State = snapshot.State_COMPLETE
	err = store1.Update(snapshot11)
	assert.NoError(t, err)

	snapshot12.Status.State = snapshot.State_COMPLETE
	err = store2.Update(snapshot12)
	assert.Error(t, err)

	// Verify events were received again
	snapshotEvent = nextEvent(t, ch)
	assert.Equal(t, devicesnapshot.ID("snapshot-2:device-2:1.0.0"), snapshotEvent.ID)
	snapshotEvent = nextEvent(t, ch)
	assert.Equal(t, devicesnapshot.ID("snapshot-2:device-2:1.0.0"), snapshotEvent.ID)
	snapshotEvent = nextEvent(t, ch)
	assert.Equal(t, devicesnapshot.ID("snapshot-1:device-1:1.0.0"), snapshotEvent.ID)

	// List the snapshots
	snapshots := make(chan *devicesnapshot.DeviceSnapshot)
	_, err = store1.List(snapshots)
	assert.NoError(t, err)

	_, ok := <-snapshots
	assert.True(t, ok)
	_, ok = <-snapshots
	assert.True(t, ok)
	_, ok = <-snapshots
	assert.False(t, ok)

	// Delete a snapshot
	err = store1.Delete(snapshot2)
	assert.NoError(t, err)
	snapshot2, err = store2.Get("snapshot-2:device-2")
	assert.NoError(t, err)
	assert.Nil(t, snapshot2)

	snapshot := &devicesnapshot.DeviceSnapshot{
		DeviceID:      device1,
		DeviceVersion: "1.0.0",
	}

	err = store1.Create(snapshot)
	assert.NoError(t, err)

	snapshot = &devicesnapshot.DeviceSnapshot{
		DeviceID:      device2,
		DeviceVersion: "1.0.0",
	}

	err = store1.Create(snapshot)
	assert.NoError(t, err)

	ch = make(chan stream.Event)
	_, err = store1.Watch(ch)
	assert.NoError(t, err)

	snapshot = nextEvent(t, ch)
	assert.NotNil(t, snapshot)
	snapshot = nextEvent(t, ch)
	assert.NotNil(t, snapshot)
	snapshot = nextEvent(t, ch)
	assert.NotNil(t, snapshot)
}

func nextEvent(t *testing.T, ch chan stream.Event) *devicesnapshot.DeviceSnapshot {
	select {
	case c := <-ch:
		return c.Object.(*devicesnapshot.DeviceSnapshot)
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
