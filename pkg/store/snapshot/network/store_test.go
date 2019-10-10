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
	"github.com/onosproject/onos-topo/pkg/northbound/device"
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

	device1 := device.ID("device-1")
	device2 := device.ID("device-2")

	ch := make(chan *networksnapshot.NetworkSnapshot)
	err = store2.Watch(ch)
	assert.NoError(t, err)

	change1 := &networksnapshot.NetworkSnapshot{
		Devices:   []device.ID{},
		Timestamp: time.Now().AddDate(0, 0, -7),
	}

	change2 := &networksnapshot.NetworkSnapshot{
		Devices:   []device.ID{device1, device2},
		Timestamp: time.Now().AddDate(0, 0, -1),
	}

	// Create a new change
	err = store1.Create(change1)
	assert.NoError(t, err)
	assert.Equal(t, networksnapshot.ID("snapshot:1"), change1.ID)
	assert.Equal(t, networksnapshot.Index(1), change1.Index)
	assert.NotEqual(t, networksnapshot.Revision(0), change1.Revision)

	// Get the change
	change1, err = store2.Get("snapshot:1")
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	assert.Equal(t, networksnapshot.ID("snapshot:1"), change1.ID)
	assert.Equal(t, networksnapshot.Index(1), change1.Index)
	assert.NotEqual(t, networksnapshot.Revision(0), change1.Revision)

	// Create another change
	err = store2.Create(change2)
	assert.NoError(t, err)
	assert.Equal(t, networksnapshot.ID("snapshot:2"), change2.ID)
	assert.Equal(t, networksnapshot.Index(2), change2.Index)
	assert.NotEqual(t, networksnapshot.Revision(0), change2.Revision)

	// Verify events were received for the changes
	changeEvent := nextChange(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot:1"), changeEvent.ID)
	changeEvent = nextChange(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot:2"), changeEvent.ID)

	// Update one of the changes
	change2.Status.State = snapshot.State_RUNNING
	revision := change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	// Read and then update the change
	change2, err = store2.Get("snapshot:2")
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	change2.Status.State = snapshot.State_COMPLETE
	revision = change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	// Verify that concurrent updates fail
	change11, err := store1.Get("snapshot:1")
	assert.NoError(t, err)
	change12, err := store2.Get("snapshot:1")
	assert.NoError(t, err)

	change11.Status.State = snapshot.State_COMPLETE
	err = store1.Update(change11)
	assert.NoError(t, err)

	change12.Status.State = snapshot.State_FAILED
	err = store2.Update(change12)
	assert.Error(t, err)

	// Verify events were received again
	changeEvent = nextChange(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot:2"), changeEvent.ID)
	changeEvent = nextChange(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot:2"), changeEvent.ID)
	changeEvent = nextChange(t, ch)
	assert.Equal(t, networksnapshot.ID("snapshot:1"), changeEvent.ID)

	// List the changes
	changes := make(chan *networksnapshot.NetworkSnapshot)
	err = store1.List(changes)
	assert.NoError(t, err)

	_, ok := <-changes
	assert.True(t, ok)
	_, ok = <-changes
	assert.True(t, ok)
	_, ok = <-changes
	assert.False(t, ok)

	// Delete a change
	err = store1.Delete(change2)
	assert.NoError(t, err)
	change2, err = store2.Get("snapshot:2")
	assert.NoError(t, err)
	assert.Nil(t, change2)

	change := &networksnapshot.NetworkSnapshot{
		Devices:   []device.ID{},
		Timestamp: time.Now().AddDate(0, 0, -7),
	}

	err = store1.Create(change)
	assert.NoError(t, err)

	change = &networksnapshot.NetworkSnapshot{
		Devices:   []device.ID{},
		Timestamp: time.Now().AddDate(0, 0, -7),
	}

	err = store1.Create(change)
	assert.NoError(t, err)

	ch = make(chan *networksnapshot.NetworkSnapshot)
	err = store1.Watch(ch)
	assert.NoError(t, err)

	change = nextChange(t, ch)
	assert.Equal(t, networksnapshot.Index(1), change.Index)
	change = nextChange(t, ch)
	assert.Equal(t, networksnapshot.Index(3), change.Index)
	change = nextChange(t, ch)
	assert.Equal(t, networksnapshot.Index(4), change.Index)
}

func nextChange(t *testing.T, ch chan *networksnapshot.NetworkSnapshot) *networksnapshot.NetworkSnapshot {
	select {
	case c := <-ch:
		return c
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
