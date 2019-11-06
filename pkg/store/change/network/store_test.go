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
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNetworkChangeStore(t *testing.T) {
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

	ch := make(chan stream.Event)
	_, err = store2.Watch(ch)
	assert.NoError(t, err)

	change1 := &networkchange.NetworkChange{
		ID: "change-1",
		Changes: []*devicechange.Change{
			{

				DeviceID:      device1,
				DeviceVersion: "1.0.0",
				Values: []*devicechange.ChangeValue{
					{
						Path: "foo",
						Value: &devicechange.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  devicechange.ValueType_STRING,
						},
					},
					{
						Path: "bar",
						Value: &devicechange.TypedValue{
							Bytes: []byte("Hello world again!"),
							Type:  devicechange.ValueType_STRING,
						},
					},
				},
			},
			{
				DeviceID: device2,
				Values: []*devicechange.ChangeValue{
					{
						Path: "baz",
						Value: &devicechange.TypedValue{
							Bytes: []byte("Goodbye world!"),
							Type:  devicechange.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	change2 := &networkchange.NetworkChange{
		ID: "change-2",
		Changes: []*devicechange.Change{
			{
				DeviceID: device1,
				Values: []*devicechange.ChangeValue{
					{
						Path:    "foo",
						Removed: true,
					},
				},
			},
		},
	}

	// Create a new change
	err = store1.Create(change1)
	assert.NoError(t, err)
	assert.Equal(t, networkchange.ID("change-1"), change1.ID)
	assert.Equal(t, networkchange.Index(1), change1.Index)
	assert.NotEqual(t, networkchange.Revision(0), change1.Revision)

	// Get the change
	change1, err = store2.Get("change-1")
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	assert.Equal(t, networkchange.ID("change-1"), change1.ID)
	assert.Equal(t, networkchange.Index(1), change1.Index)
	assert.NotEqual(t, networkchange.Revision(0), change1.Revision)

	// Create another change
	err = store2.Create(change2)
	assert.NoError(t, err)
	assert.Equal(t, networkchange.ID("change-2"), change2.ID)
	assert.Equal(t, networkchange.Index(2), change2.Index)
	assert.NotEqual(t, networkchange.Revision(0), change2.Revision)

	// Verify events were received for the changes
	changeEvent := nextEvent(t, ch)
	assert.Equal(t, networkchange.ID("change-1"), changeEvent.ID)
	changeEvent = nextEvent(t, ch)
	assert.Equal(t, networkchange.ID("change-2"), changeEvent.ID)

	// Watch events for a specific change
	changeCh := make(chan stream.Event)
	_, err = store1.Watch(changeCh, WithChangeID(change2.ID))
	assert.NoError(t, err)

	// Update one of the changes
	change2.Status.State = changetypes.State_RUNNING
	revision := change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	event := (<-changeCh).Object.(*networkchange.NetworkChange)
	assert.Equal(t, change2.ID, event.ID)
	assert.Equal(t, change2.Revision, event.Revision)

	// Get previous and next changes
	change1, err = store2.Get("change-1")
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	change2, err = store2.GetNext(change1.Index)
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	assert.Equal(t, networkchange.Index(2), change2.Index)

	change2, err = store2.Get("change-2")
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	change1, err = store2.GetPrev(change2.Index)
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	assert.Equal(t, networkchange.Index(1), change1.Index)

	// Read and then update the change
	change2, err = store2.Get("change-2")
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	change2.Status.State = changetypes.State_COMPLETE
	revision = change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	event = (<-changeCh).Object.(*networkchange.NetworkChange)
	assert.Equal(t, change2.ID, event.ID)
	assert.Equal(t, change2.Revision, event.Revision)

	// Verify that concurrent updates fail
	change11, err := store1.Get("change-1")
	assert.NoError(t, err)
	change12, err := store2.Get("change-1")
	assert.NoError(t, err)

	change11.Status.State = changetypes.State_COMPLETE
	err = store1.Update(change11)
	assert.NoError(t, err)

	change12.Status.State = changetypes.State_FAILED
	err = store2.Update(change12)
	assert.Error(t, err)

	// Verify events were received again
	changeEvent = nextEvent(t, ch)
	assert.Equal(t, networkchange.ID("change-2"), changeEvent.ID)
	changeEvent = nextEvent(t, ch)
	assert.Equal(t, networkchange.ID("change-2"), changeEvent.ID)
	changeEvent = nextEvent(t, ch)
	assert.Equal(t, networkchange.ID("change-1"), changeEvent.ID)

	// List the changes
	changes := make(chan *networkchange.NetworkChange)
	_, err = store1.List(changes)
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
	change, err := store2.Get("change-2")
	assert.NoError(t, err)
	assert.Nil(t, change)

	event = (<-changeCh).Object.(*networkchange.NetworkChange)
	assert.Equal(t, change2.ID, event.ID)

	change = &networkchange.NetworkChange{
		ID: "change-3",
		Changes: []*devicechange.Change{
			{
				DeviceID: device1,
				Values: []*devicechange.ChangeValue{
					{
						Path: "foo",
						Value: &devicechange.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  devicechange.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	err = store1.Create(change)
	assert.NoError(t, err)

	change = &networkchange.NetworkChange{
		ID: "change-4",
		Changes: []*devicechange.Change{
			{
				DeviceID: device2,
				Values: []*devicechange.ChangeValue{
					{
						Path: "bar",
						Value: &devicechange.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  devicechange.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	err = store1.Create(change)
	assert.NoError(t, err)

	ch = make(chan stream.Event)
	_, err = store1.Watch(ch, WithReplay())
	assert.NoError(t, err)

	change = nextEvent(t, ch)
	assert.Equal(t, networkchange.Index(1), change.Index)
	change = nextEvent(t, ch)
	assert.Equal(t, networkchange.Index(3), change.Index)
	change = nextEvent(t, ch)
	assert.Equal(t, networkchange.Index(4), change.Index)
}

func nextEvent(t *testing.T, ch chan stream.Event) *networkchange.NetworkChange {
	select {
	case c := <-ch:
		return c.Object.(*networkchange.NetworkChange)
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
