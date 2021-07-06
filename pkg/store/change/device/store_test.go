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
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	types "github.com/onosproject/onos-api/go/onos/config"
	changetypes "github.com/onosproject/onos-api/go/onos/config/change"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/stretchr/testify/assert"
	assert2 "gotest.tools/assert"
	"testing"
	"time"
)

func TestDeviceStore(t *testing.T) {
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
		test.WithDebugLogs())
	assert.NoError(t, test.Start())
	defer test.Stop()

	client1, err := test.NewClient("node-1")
	assert.NoError(t, err)

	client2, err := test.NewClient("node-2")
	assert.NoError(t, err)

	store1, err := NewAtomixStore(client1)
	assert.NoError(t, err)

	store2, err := NewAtomixStore(client2)
	assert.NoError(t, err)

	device1 := device.ID("device-1")
	device2 := device.ID("device-2")

	ch1 := make(chan stream.Event)
	_, err = store2.Watch(device.NewVersionedID(device1, "1.0.0"), ch1)
	assert.NoError(t, err)

	ch2 := make(chan stream.Event)
	_, err = store2.Watch(device.NewVersionedID(device2, "1.0.0"), ch2)
	assert.NoError(t, err)

	change1 := &devicechange.DeviceChange{
		Index: 1,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID("network-change-1"),
			Index: 1,
		},
		Change: &devicechange.Change{
			DeviceID:      device1,
			DeviceType:    "Stratum",
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
	}

	change2 := &devicechange.DeviceChange{
		Index: 2,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID("network-change-2"),
			Index: 2,
		},
		Change: &devicechange.Change{
			DeviceID:      device1,
			DeviceType:    "Stratum",
			DeviceVersion: "1.0.0",
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
	}

	// Create a new change
	err = store1.Create(change1)
	assert.NoError(t, err)
	assert.Equal(t, devicechange.ID("network-change-1:device-1:1.0.0"), change1.ID)
	assert.Equal(t, devicechange.Index(1), change1.Index)
	assert.NotEqual(t, devicechange.Revision(0), change1.Revision)

	// Get the change
	change1, err = store2.Get(devicechange.ID("network-change-1:device-1:1.0.0"))
	assert.NoError(t, err)
	assert.NotNil(t, change1)
	assert.Equal(t, devicechange.ID("network-change-1:device-1:1.0.0"), change1.ID)
	assert.Equal(t, devicechange.Index(1), change1.Index)
	assert.NotEqual(t, devicechange.Revision(0), change1.Revision)

	// Append another change
	err = store2.Create(change2)
	assert.NoError(t, err)
	assert.Equal(t, devicechange.ID("network-change-2:device-1:1.0.0"), change2.ID)
	assert.Equal(t, devicechange.Index(2), change2.Index)
	assert.NotEqual(t, devicechange.Revision(0), change2.Revision)

	change3 := &devicechange.DeviceChange{
		Index: 3,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID("network-change-3"),
			Index: 3,
		},
		Change: &devicechange.Change{
			DeviceID:      device1,
			DeviceType:    "Stratum",
			DeviceVersion: "1.0.0",
			Values: []*devicechange.ChangeValue{
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
	assert.Equal(t, devicechange.ID("network-change-3:device-1:1.0.0"), change3.ID)
	assert.Equal(t, devicechange.Index(3), change3.Index)
	assert.NotEqual(t, devicechange.Revision(0), change3.Revision)

	// For two devices
	change4 := &devicechange.DeviceChange{
		Index: 4,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID("network-change-3"),
			Index: 3,
		},
		Change: &devicechange.Change{
			DeviceID:      device2,
			DeviceType:    "Stratum",
			DeviceVersion: "1.0.0",
			Values: []*devicechange.ChangeValue{
				{
					Path: "foo",
					Value: &devicechange.TypedValue{
						Bytes: []byte("bar"),
						Type:  devicechange.ValueType_STRING,
					},
				},
			},
		},
	}

	// Append another change
	err = store1.Create(change4)
	assert.NoError(t, err)
	assert.Equal(t, devicechange.ID("network-change-3:device-2:1.0.0"), change4.ID)
	assert.Equal(t, devicechange.Index(4), change4.Index)
	assert.NotEqual(t, devicechange.Revision(0), change4.Revision)

	// Verify events were received for the changes
	changeEvent := nextEvent(t, ch1)
	assert.Equal(t, devicechange.ID("network-change-1:device-1:1.0.0"), changeEvent.ID)
	changeEvent = nextEvent(t, ch1)
	assert.Equal(t, devicechange.ID("network-change-2:device-1:1.0.0"), changeEvent.ID)
	changeEvent = nextEvent(t, ch1)
	assert.Equal(t, devicechange.ID("network-change-3:device-1:1.0.0"), changeEvent.ID)
	changeEvent = nextEvent(t, ch2)
	assert.Equal(t, devicechange.ID("network-change-3:device-2:1.0.0"), changeEvent.ID)

	// Watch events for a specific change
	changeCh := make(chan stream.Event)
	_, err = store1.Watch(device.NewVersionedID(device1, "1.0.0"), changeCh, WithChangeID(change2.ID))
	assert.NoError(t, err)

	// Update one of the changes
	change2.Status.Incarnation++
	revision := change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	event := (<-changeCh).Object.(*devicechange.DeviceChange)
	assert.Equal(t, change2.ID, event.ID)
	assert.Equal(t, change2.Revision, event.Revision)

	// Read and then update the change
	change2, err = store2.Get(devicechange.ID("network-change-2:device-1:1.0.0"))
	assert.NoError(t, err)
	assert.NotNil(t, change2)
	change2.Status.Incarnation++
	revision = change2.Revision
	err = store1.Update(change2)
	assert.NoError(t, err)
	assert.NotEqual(t, revision, change2.Revision)

	event = (<-changeCh).Object.(*devicechange.DeviceChange)
	assert.Equal(t, change2.ID, event.ID)
	assert.Equal(t, change2.Revision, event.Revision)

	// Verify that concurrent updates fail
	change31, err := store1.Get(devicechange.ID("network-change-3:device-1:1.0.0"))
	assert.NoError(t, err)
	change32, err := store2.Get(devicechange.ID("network-change-3:device-1:1.0.0"))
	assert.NoError(t, err)

	change31.Status.Incarnation++
	err = store1.Update(change31)
	assert.NoError(t, err)

	change32.Status.State = changetypes.State_FAILED
	err = store2.Update(change32)
	assert.Error(t, err)

	// Verify device events were received again
	changeEvent = nextEvent(t, ch1)
	assert.Equal(t, devicechange.ID("network-change-2:device-1:1.0.0"), changeEvent.ID)
	changeEvent = nextEvent(t, ch1)
	assert.Equal(t, devicechange.ID("network-change-2:device-1:1.0.0"), changeEvent.ID)
	changeEvent = nextEvent(t, ch1)
	assert.Equal(t, devicechange.ID("network-change-3:device-1:1.0.0"), changeEvent.ID)

	// List the changes for a device
	changes := make(chan *devicechange.DeviceChange)
	_, err = store1.List(device.NewVersionedID(device1, "1.0.0"), changes)
	assert.NoError(t, err)

	changeEvent = nextDeviceChange(t, changes)
	assert.Equal(t, devicechange.ID("network-change-1:device-1:1.0.0"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, changes)
	assert.Equal(t, devicechange.ID("network-change-2:device-1:1.0.0"), changeEvent.ID)
	changeEvent = nextDeviceChange(t, changes)
	assert.Equal(t, devicechange.ID("network-change-3:device-1:1.0.0"), changeEvent.ID)

	select {
	case _, ok := <-changes:
		assert.False(t, ok)
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	// Delete a change
	err = store1.Delete(change2)
	assert.NoError(t, err)
	change, err := store2.Get("network-change-2:device-1:1.0.0")
	assert.Error(t, err)
	assert.Nil(t, change)
	assert.True(t, errors.IsNotFound(err))

	event = (<-changeCh).Object.(*devicechange.DeviceChange)
	assert.Equal(t, change2.ID, event.ID)

	// Ensure existing changes are replayed in order on watch
	watches := make(chan stream.Event)
	_, err = store1.Watch(device.NewVersionedID(device1, "1.0.0"), watches, WithReplay())
	assert.NoError(t, err)

	change = nextEvent(t, watches)
	assert.Equal(t, devicechange.ID("network-change-1:device-1:1.0.0"), change.ID)
	change = nextEvent(t, watches)
	assert.Equal(t, devicechange.ID("network-change-3:device-1:1.0.0"), change.ID)
}

func nextDeviceChange(t *testing.T, ch chan *devicechange.DeviceChange) *devicechange.DeviceChange {
	select {
	case change := <-ch:
		return change
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}

func nextEvent(t *testing.T, ch chan stream.Event) *devicechange.DeviceChange {
	select {
	case change := <-ch:
		return change.Object.(*devicechange.DeviceChange)
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}

func Test_badpath(t *testing.T) {
	badpath := "does_not_have_any_slash"
	expected := fmt.Sprintf("invalid path %s. Must match %s", badpath, devicechange.PathRegexp)
	conf1, err1 := devicechange.NewChangeValue(badpath, devicechange.NewTypedValueString("123"), false)

	assert2.Error(t, err1, expected, "Expected error on ", badpath)

	assert2.Assert(t, conf1 == nil, "Expected config to be empty on error")

	badpath = "//two/contiguous/slashes"
	_, err2 := devicechange.NewChangeValue(badpath, devicechange.NewTypedValueString("123"), false)
	assert2.ErrorContains(t, err2, badpath, "Expected error on path", badpath)

	badpath = "/test*"
	_, err3 := devicechange.NewChangeValue(badpath, devicechange.NewTypedValueString("123"), false)
	assert2.ErrorContains(t, err3, badpath, "Expected error on path", badpath)
}
