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
	"github.com/onosproject/onos-config/pkg/types/snapshot"
	networksnapshot "github.com/onosproject/onos-config/pkg/types/snapshot/network"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNetworkSnapshotStore(t *testing.T) {
	node, conn := startLocalNode()
	defer node.Stop()
	defer conn.Close()

	store1, err := newLocalStore(conn)
	assert.NoError(t, err)
	defer store1.Close()

	store2, err := newLocalStore(conn)
	assert.NoError(t, err)
	defer store2.Close()

	ch := make(chan *networksnapshot.NetworkSnapshot)
	err = store2.Watch(ch)
	assert.NoError(t, err)

	retainWindow := 24 * time.Hour
	snapshot1 := &networksnapshot.NetworkSnapshot{
		ID: "snapshot-1",
		Retention: snapshot.RetentionOptions{
			RetainWindow: &retainWindow,
		},
	}

	snapshot2 := &networksnapshot.NetworkSnapshot{
		ID: "snapshot-2",
		Retention: snapshot.RetentionOptions{
			RetainWindow: &retainWindow,
		},
	}

	// Create a new snapshot
	err = store1.Create(snapshot1)
	assert.NoError(t, err)
	assert.Equal(t, networksnapshot.ID("snapshot-1"), snapshot1.ID)
	assert.Equal(t, networksnapshot.Index(1), snapshot1.Index)
	assert.NotEqual(t, networksnapshot.Revision(0), snapshot1.Revision)

	// Get the snapshot
	snapshot1, err = store2.Get("snapshot-1")
	assert.NoError(t, err)
	assert.NotNil(t, snapshot1)
	assert.Equal(t, networksnapshot.ID("snapshot-1"), snapshot1.ID)
	assert.Equal(t, networksnapshot.Index(1), snapshot1.Index)
	assert.NotEqual(t, networksnapshot.Revision(0), snapshot1.Revision)

	// Create another snapshot
	err = store2.Create(snapshot2)
	assert.NoError(t, err)
	assert.Equal(t, networksnapshot.ID("snapshot-2"), snapshot2.ID)
	assert.Equal(t, networksnapshot.Index(2), snapshot2.Index)
	assert.NotEqual(t, networksnapshot.Revision(0), snapshot2.Revision)

	// Verify events were received for the snapshots
	snapshotEvent := nextSnapshot(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot-1"), snapshotEvent.ID)
	snapshotEvent = nextSnapshot(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot-2"), snapshotEvent.ID)

	// Update one of the snapshots
	snapshot2.Status.State = snapshot.State_RUNNING
	revision := snapshot2.Revision
	err = store1.Update(snapshot2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, snapshot2.Revision)

	// Read and then update the snapshot
	snapshot2, err = store2.Get("snapshot-2")
	assert.NoError(t, err)
	assert.NotNil(t, snapshot2)
	snapshot2.Status.State = snapshot.State_COMPLETE
	revision = snapshot2.Revision
	err = store1.Update(snapshot2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, snapshot2.Revision)

	// Verify that concurrent updates fail
	snapshot11, err := store1.Get("snapshot-1")
	assert.NoError(t, err)
	snapshot12, err := store2.Get("snapshot-1")
	assert.NoError(t, err)

	snapshot11.Status.State = snapshot.State_COMPLETE
	err = store1.Update(snapshot11)
	assert.NoError(t, err)

	snapshot12.Status.State = snapshot.State_COMPLETE
	err = store2.Update(snapshot12)
	assert.Error(t, err)

	// Verify events were received again
	snapshotEvent = nextSnapshot(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot-2"), snapshotEvent.ID)
	snapshotEvent = nextSnapshot(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot-2"), snapshotEvent.ID)
	snapshotEvent = nextSnapshot(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot-1"), snapshotEvent.ID)

	// List the snapshots
	snapshots := make(chan *networksnapshot.NetworkSnapshot)
	err = store1.List(snapshots)
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
	snapshot2, err = store2.Get("snapshot-2")
	assert.NoError(t, err)
	assert.Nil(t, snapshot2)
}

func nextSnapshot(t *testing.T, ch chan *networksnapshot.NetworkSnapshot) *networksnapshot.NetworkSnapshot {
	select {
	case c := <-ch:
		return c
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
