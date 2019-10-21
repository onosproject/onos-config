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
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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

	device1 := devicetopo.ID("device-1")
	device2 := devicetopo.ID("device-2")

	ch1 := make(chan *devicechangetypes.DeviceChange)
	err = store2.Watch(device1, ch1)
	assert.NoError(t, err)

	ch2 := make(chan *devicechangetypes.DeviceChange)
	err = store2.Watch(device2, ch2)
	assert.NoError(t, err)

	change1 := &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID("network-change-1"),
			Index: 1,
		},
		Change: &devicechangetypes.Change{
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
	}

	change2 := &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID("network-change-2"),
			Index: 2,
		},
		Change: &devicechangetypes.Change{
			DeviceID: device1,
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
	}

	// Create a new change
	err = store1.Create(change1)
	assert.NoError(t, err)
	assert.Equal(t, devicechangetypes.ID("network-change-1:device-1"), change1.ID)
	assert.Equal(t, devicechangetypes.Index(1), change1.Index)
	assert.NotEqual(t, devicechangetypes.Revision(0), change1.Revision)

	// Get the change
	change1, err = store2.Get(devicechangetypes.ID("network-change-1:device-1"))
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	assert.Equal(t, devicechangetypes.ID("network-change-1:device-1"), change1.ID)
	assert.Equal(t, devicechangetypes.Index(1), change1.Index)
	assert.NotEqual(t, devicechangetypes.Revision(0), change1.Revision)

	// Append another change
	err = store2.Create(change2)
	assert.NoError(t, err)
	assert.Equal(t, devicechangetypes.ID("network-change-2:device-1"), change2.ID)
	assert.Equal(t, devicechangetypes.Index(2), change2.Index)
	assert.NotEqual(t, devicechangetypes.Revision(0), change2.Revision)

	change3 := &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID("network-change-3"),
			Index: 3,
		},
		Change: &devicechangetypes.Change{
			DeviceID: device1,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:    "foo",
					Removed: true,
				},
			},
		},
	}

	// Append another change
	err = store1.Create(change3)
	assert.NoError(t, err)
	assert.Equal(t, devicechangetypes.ID("network-change-3:device-1"), change3.ID)
	assert.Equal(t, devicechangetypes.Index(3), change3.Index)
	assert.NotEqual(t, devicechangetypes.Revision(0), change3.Revision)

	// For two devices
	change4 := &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID("network-change-3"),
			Index: 3,
		},
		Change: &devicechangetypes.Change{
			DeviceID: device2,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path: "foo",
					Value: &devicechangetypes.TypedValue{
						Bytes: []byte("bar"),
						Type:  devicechangetypes.ValueType_STRING,
					},
				},
			},
		},
	}

	// Append another change
	err = store1.Create(change4)
	assert.NoError(t, err)
	assert.Equal(t, devicechangetypes.ID("network-change-3:device-2"), change4.ID)
	assert.Equal(t, devicechangetypes.Index(1), change4.Index)
	assert.NotEqual(t, devicechangetypes.Revision(0), change4.Revision)

	// Verify events were received for the changes
	changeEvent := nextDeviceChange(t, ch1)
	assert.Equal(t, devicechangetypes.ID("network-change-1:device-1"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, ch1)
	assert.Equal(t, devicechangetypes.ID("network-change-2:device-1"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, ch1)
	assert.Equal(t, devicechangetypes.ID("network-change-3:device-1"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, ch2)
	assert.Equal(t, devicechangetypes.ID("network-change-3:device-2"), changeEvent.ID)

	// Update one of the changes
	change2.Status.State = change.State_RUNNING
	revision := change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	// Read and then update the change
	change2, err = store2.Get(devicechangetypes.ID("network-change-2:device-1"))
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	change2.Status.State = change.State_RUNNING
	revision = change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	// Verify that concurrent updates fail
	change31, err := store1.Get(devicechangetypes.ID("network-change-3:device-1"))
	assert.NoError(t, err)
	change32, err := store2.Get(devicechangetypes.ID("network-change-3:device-1"))
	assert.NoError(t, err)

	change31.Status.State = change.State_RUNNING
	err = store1.Update(change31)
	assert.NoError(t, err)

	change32.Status.State = change.State_FAILED
	err = store2.Update(change32)
	assert.Error(t, err)

	// Verify device events were received again
	changeEvent = nextDeviceChange(t, ch1)
	assert.Equal(t, devicechangetypes.ID("network-change-2:device-1"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, ch1)
	assert.Equal(t, devicechangetypes.ID("network-change-2:device-1"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, ch1)
	assert.Equal(t, devicechangetypes.ID("network-change-3:device-1"), changeEvent.ID)

	// List the changes for a device
	changes := make(chan *devicechangetypes.DeviceChange)
	err = store1.List(device1, changes)
	assert.NoError(t, err)

	changeEvent = nextDeviceChange(t, changes)
	assert.Equal(t, devicechangetypes.ID("network-change-1:device-1"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, changes)
	assert.Equal(t, devicechangetypes.ID("network-change-2:device-1"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, changes)
	assert.Equal(t, devicechangetypes.ID("network-change-3:device-1"), changeEvent.ID)

	select {
	case _, ok := <-changes:
		assert.False(t, ok)
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	// Delete a change
	err = store1.Delete(change2)
	assert.NoError(t, err)
	change2, err = store2.Get("network-change-2:device-1")
	assert.NoError(t, err)
	assert.Nil(t, change2)

	// Ensure existing changes are replayed in order on watch
	watches := make(chan *devicechangetypes.DeviceChange)
	err = store1.Watch(device1, watches)
	assert.NoError(t, err)

	change := nextDeviceChange(t, watches)
	assert.Equal(t, devicechangetypes.ID("network-change-1:device-1"), change.ID)
	change = nextDeviceChange(t, watches)
	assert.Equal(t, devicechangetypes.ID("network-change-3:device-1"), change.ID)
}

func nextDeviceChange(t *testing.T, ch chan *devicechangetypes.DeviceChange) *devicechangetypes.DeviceChange {
	select {
	case change := <-ch:
		return change
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
