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
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
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

	device1 := devicetopo.ID("device-1")
	device2 := devicetopo.ID("device-2")

	ch := make(chan *networkchangetypes.NetworkChange)
	err = store2.Watch(ch)
	assert.NoError(t, err)

	change1 := &networkchangetypes.NetworkChange{
		ID: "change-1",
		Changes: []*devicechangetypes.Change{
			{

				DeviceID: device1,
				Values: []*devicechangetypes.ChangeValue{
					{
						Path: "foo",
						Value: &devicechangetypes.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  devicechangetypes.ValueType_STRING,
						},
					},
					{
						Path: "bar",
						Value: &devicechangetypes.TypedValue{
							Bytes: []byte("Hello world again!"),
							Type:  devicechangetypes.ValueType_STRING,
						},
					},
				},
			},
			{
				DeviceID: device2,
				Values: []*devicechangetypes.ChangeValue{
					{
						Path: "baz",
						Value: &devicechangetypes.TypedValue{
							Bytes: []byte("Goodbye world!"),
							Type:  devicechangetypes.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	change2 := &networkchangetypes.NetworkChange{
		ID: "change-2",
		Changes: []*devicechangetypes.Change{
			{
				DeviceID: device1,
				Values: []*devicechangetypes.ChangeValue{
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
	assert.Equal(t, networkchangetypes.ID("change-1"), change1.ID)
	assert.Equal(t, networkchangetypes.Index(1), change1.Index)
	assert.NotEqual(t, networkchangetypes.Revision(0), change1.Revision)

	// Get the change
	change1, err = store2.Get("change-1")
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	assert.Equal(t, networkchangetypes.ID("change-1"), change1.ID)
	assert.Equal(t, networkchangetypes.Index(1), change1.Index)
	assert.NotEqual(t, networkchangetypes.Revision(0), change1.Revision)

	// Create another change
	err = store2.Create(change2)
	assert.NoError(t, err)
	assert.Equal(t, networkchangetypes.ID("change-2"), change2.ID)
	assert.Equal(t, networkchangetypes.Index(2), change2.Index)
	assert.NotEqual(t, networkchangetypes.Revision(0), change2.Revision)

	// Verify events were received for the changes
	changeEvent := nextChange(t, ch)
	assert.Equal(t, networkchangetypes.ID("change-1"), changeEvent.ID)
	changeEvent = nextChange(t, ch)
	assert.Equal(t, networkchangetypes.ID("change-2"), changeEvent.ID)

	// Update one of the changes
	change2.Status.State = change.State_RUNNING
	revision := change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	// Get previous and next changes
	change1, err = store2.Get("change-1")
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	change2, err = store2.GetNext(change1.Index)
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	assert.Equal(t, networkchangetypes.Index(2), change2.Index)

	change2, err = store2.Get("change-2")
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	change1, err = store2.GetPrev(change2.Index)
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	assert.Equal(t, networkchangetypes.Index(1), change1.Index)

	// Read and then update the change
	change2, err = store2.Get("change-2")
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	change2.Status.State = change.State_COMPLETE
	revision = change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	// Verify that concurrent updates fail
	change11, err := store1.Get("change-1")
	assert.NoError(t, err)
	change12, err := store2.Get("change-1")
	assert.NoError(t, err)

	change11.Status.State = change.State_COMPLETE
	err = store1.Update(change11)
	assert.NoError(t, err)

	change12.Status.State = change.State_FAILED
	err = store2.Update(change12)
	assert.Error(t, err)

	// Verify events were received again
	changeEvent = nextChange(t, ch)
	assert.Equal(t, networkchangetypes.ID("change-2"), changeEvent.ID)
	changeEvent = nextChange(t, ch)
	assert.Equal(t, networkchangetypes.ID("change-2"), changeEvent.ID)
	changeEvent = nextChange(t, ch)
	assert.Equal(t, networkchangetypes.ID("change-1"), changeEvent.ID)

	// List the changes
	changes := make(chan *networkchangetypes.NetworkChange)
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
	change2, err = store2.Get("change-2")
	assert.NoError(t, err)
	assert.Nil(t, change2)

	change := &networkchangetypes.NetworkChange{
		ID: "change-3",
		Changes: []*devicechangetypes.Change{
			{
				DeviceID: device1,
				Values: []*devicechangetypes.ChangeValue{
					{
						Path: "foo",
						Value: &devicechangetypes.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  devicechangetypes.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	err = store1.Create(change)
	assert.NoError(t, err)

	change = &networkchangetypes.NetworkChange{
		ID: "change-4",
		Changes: []*devicechangetypes.Change{
			{
				DeviceID: device2,
				Values: []*devicechangetypes.ChangeValue{
					{
						Path: "bar",
						Value: &devicechangetypes.TypedValue{
							Bytes: []byte("Hello world!"),
							Type:  devicechangetypes.ValueType_STRING,
						},
					},
				},
			},
		},
	}

	err = store1.Create(change)
	assert.NoError(t, err)

	ch = make(chan *networkchangetypes.NetworkChange)
	err = store1.Watch(ch)
	assert.NoError(t, err)

	change = nextChange(t, ch)
	assert.Equal(t, networkchangetypes.Index(1), change.Index)
	change = nextChange(t, ch)
	assert.Equal(t, networkchangetypes.Index(3), change.Index)
	change = nextChange(t, ch)
	assert.Equal(t, networkchangetypes.Index(4), change.Index)
}

func nextChange(t *testing.T, ch chan *networkchangetypes.NetworkChange) *networkchangetypes.NetworkChange {
	select {
	case c := <-ch:
		return c
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
