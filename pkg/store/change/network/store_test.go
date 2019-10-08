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
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchange "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDeviceStore(t *testing.T) {
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

	ch := make(chan *networkchange.NetworkChange)
	err = store2.Watch(ch)
	assert.NoError(t, err)

	change1 := &networkchange.NetworkChange{
		Changes: []*devicechange.Change{
			{

				DeviceID: device1,
				Values: []*devicechange.Value{
					{
						Path:  "foo",
						Value: []byte("Hello world!"),
						Type:  devicechange.ValueType_STRING,
					},
					{
						Path:  "bar",
						Value: []byte("Hello world again!"),
						Type:  devicechange.ValueType_STRING,
					},
				},
			},
			{
				DeviceID: device2,
				Values: []*devicechange.Value{
					{
						Path:  "baz",
						Value: []byte("Goodbye world!"),
						Type:  devicechange.ValueType_STRING,
					},
				},
			},
		},
	}

	change2 := &networkchange.NetworkChange{
		Changes: []*devicechange.Change{
			{
				DeviceID:        device1,
				Values: []*devicechange.Value{
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
	assert.Equal(t, networkchange.ID("network:1"), change1.ID)
	assert.Equal(t, networkchange.Index(1), change1.Index)
	assert.NotEqual(t, networkchange.Revision(0), change1.Revision)

	// Get the change
	change1, err = store2.Get("network:1")
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	assert.Equal(t, networkchange.ID("network:1"), change1.ID)
	assert.Equal(t, networkchange.Index(1), change1.Index)
	assert.NotEqual(t, networkchange.Revision(0), change1.Revision)

	// Create another change
	err = store2.Create(change2)
	assert.NoError(t, err)
	assert.Equal(t, networkchange.ID("network:2"), change2.ID)
	assert.Equal(t, networkchange.Index(2), change2.Index)
	assert.NotEqual(t, networkchange.Revision(0), change2.Revision)

	// Verify events were received for the changes
	changeEvent := <-ch
	assert.Equal(t, networkchange.ID("network:1"), changeEvent.ID)
	changeEvent = <-ch
	assert.Equal(t, networkchange.ID("network:2"), changeEvent.ID)

	// Update one of the changes
	change2.Status = change.Status_APPLYING
	revision := change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	// Read and then update the change
	change2, err = store2.Get("network:2")
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	change2.Status = change.Status_SUCCEEDED
	revision = change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	// Verify that concurrent updates fail
	change11, err := store1.Get("network:1")
	assert.NoError(t, err)
	change12, err := store2.Get("network:1")
	assert.NoError(t, err)

	change11.Status = change.Status_SUCCEEDED
	err = store1.Update(change11)
	assert.NoError(t, err)

	change12.Status = change.Status_FAILED
	err = store2.Update(change12)
	assert.Error(t, err)

	// Verify events were received again
	changeEvent = <-ch
	assert.Equal(t, networkchange.ID("network:2"), changeEvent.ID)
	changeEvent = <-ch
	assert.Equal(t, networkchange.ID("network:2"), changeEvent.ID)
	changeEvent = <-ch
	assert.Equal(t, networkchange.ID("network:1"), changeEvent.ID)

	// List the changes
	changes := make(chan *networkchange.NetworkChange)
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
	change2, err = store2.Get("network:2")
	assert.NoError(t, err)
	assert.Nil(t, change2)
}
