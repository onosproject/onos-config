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
//

package cache

import (
	"github.com/golang/mock/gomock"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	devicebase "github.com/onosproject/onos-api/go/onos/config/device"
	devicesnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeviceCache(t *testing.T) {
	chNwChangesVal := &atomic.Value{}
	chSnapshotsVal := &atomic.Value{}

	ctrl := gomock.NewController(t)
	netChangeStore := store.NewMockNetworkChangesStore(ctrl)
	netChangeStore.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ch chan<- stream.Event, opts ...networkchangestore.WatchOption) (stream.Context, error) {
			chNwChangesVal.Store(ch)
			return stream.NewContext(func() {
			}), nil
		}).AnyTimes()

	devSnapshotStore := store.NewMockDeviceSnapshotStore(ctrl)
	devSnapshotStore.EXPECT().Watch(gomock.Any()).DoAndReturn(
		func(chSs chan<- stream.Event) (stream.Context, error) {
			chSnapshotsVal.Store(chSs)
			return stream.NewContext(func() {
			}), nil
		}).AnyTimes()
	cache, err := NewCache(netChangeStore, devSnapshotStore)
	assert.NoError(t, err)

	// Before there are any listeners - create an entry in Network Change store
	chNwChanges := chNwChangesVal.Load().(chan<- stream.Event)
	chSnapshots := chSnapshotsVal.Load().(chan<- stream.Event)

	chNwChanges <- stream.Event{
		Type: stream.Created,
		Object: &networkchange.NetworkChange{
			ID:    "network-change-1",
			Index: 1,
			Changes: []*devicechange.Change{
				{
					DeviceID:      "device-1",
					DeviceType:    "Stratum",
					DeviceVersion: "3.0.0",
				},
				{
					DeviceID:      "device-2",
					DeviceType:    "Stratum",
					DeviceVersion: "1.0.0",
				},
			},
		},
	}

	////// A fist call to Watch with replay ////////////////
	t.Log("Setting up chan 1")
	cacheChan1 := make(chan stream.Event)
	go func() {
		var count int
		breakout := false
		for {
			select {
			case eventObj := <-cacheChan1:
				event, ok := eventObj.Object.(*Info)
				assert.True(t, ok)
				t.Log("Chan 1 Event", event)
				count++
				if count == 6 { // Expecting 6 results on chan 1
					breakout = true
				}
			case <-time.After(3 * time.Second):
				t.Error("Timed out waiting for cache event on stream 1")
			}
			if breakout {
				break
			}
		}
	}()

	watcher1Ctx, err := cache.Watch(cacheChan1, true)
	assert.NoError(t, err)
	assert.NotNil(t, watcher1Ctx)
	// defer watcher1Ctx.Close() // Will close later in this test

	////// A second call to Watch with no replay ////////////////
	t.Log("Setting up chan 2")
	cacheChan2 := make(chan stream.Event)
	go func() {
		for {
			var count int
			breakout := false
			select {
			case eventObj := <-cacheChan2:
				event, ok := eventObj.Object.(*Info)
				assert.True(t, ok)
				t.Log("Chan 2 Event", event)
				count++
				if count == 4 { // Expecting 4 results on chan 2
					breakout = true
				}
			case <-time.After(3 * time.Second):
				t.Error("Timed out waiting for cache event on stream 2")
			}
			if breakout {
				break
			}
		}
	}()
	watcher2Ctx, err := cache.Watch(cacheChan2, false)
	assert.NoError(t, err)
	assert.NotNil(t, watcher1Ctx)
	defer watcher2Ctx.Close()

	//chNwChanges = chNwChangesVal.Load().(chan<- stream.Event)
	chNwChanges <- stream.Event{
		Type: stream.Created,
		Object: &networkchange.NetworkChange{
			ID:    "network-change-1",
			Index: 1,
			Changes: []*devicechange.Change{
				{
					DeviceID:      "device-1",
					DeviceType:    "Stratum",
					DeviceVersion: "1.0.0",
				},
				{
					DeviceID:      "device-2",
					DeviceType:    "Stratum",
					DeviceVersion: "1.0.0",
				},
			},
		},
	}
	chNwChanges <- stream.Event{
		Type: stream.Created,
		Object: &networkchange.NetworkChange{
			ID:    "network-change-1",
			Index: 1,
			Changes: []*devicechange.Change{
				{
					DeviceID:      "device-2",
					DeviceType:    "Stratum",
					DeviceVersion: "2.0.0",
				},
				{
					DeviceID:      "device-1",
					DeviceType:    "Stratum",
					DeviceVersion: "1.0.0",
				},
			},
		},
	}
	// Send a Device Snapshot - this might have been the result of a previous compaction
	// there are no network changes left but it still exists
	chSnapshots <- stream.Event{
		Type: stream.Created,
		Object: &devicesnapshot.DeviceSnapshot{
			ID:            "dev-snapshot-1",
			DeviceID:      "device-old-ss",
			DeviceType:    "TestDevice",
			DeviceVersion: "2.0.0",
		},
	}

	// Send an event for something that already exists as a NW change
	// Should be ignored
	chSnapshots <- stream.Event{
		Type: stream.Created,
		Object: &devicesnapshot.DeviceSnapshot{
			ID:            "dev-snapshot-2",
			DeviceID:      "device-2",
			DeviceType:    "Devicesim",
			DeviceVersion: "1.0.0",
		},
	}

	// A network change could come after a Dev Snapshot
	chNwChanges <- stream.Event{
		Type: stream.Created,
		Object: &networkchange.NetworkChange{
			ID:    "network-change-1",
			Index: 1,
			Changes: []*devicechange.Change{
				{
					DeviceID:      "device-3",
					DeviceType:    "NotStratum",
					DeviceVersion: "1.0.0",
				},
			},
		},
	}

	// Need to wait for the event to be read by the cache
	time.Sleep(10 * time.Millisecond)

	devices := cache.GetDevicesByID("device-1")
	assert.Len(t, devices, 2)
	for _, device := range devices {
		switch device.Version {
		case "1.0.0":
		case "3.0.0":
			assert.Equal(t, devicebase.Type("Stratum"), device.Type)
		default:
			t.Error("Unexpected version for device-1", device.Version)
		}
	}

	devices = cache.GetDevicesByID("device-2")
	assert.Len(t, devices, 2)

	devices = cache.GetDevices()
	assert.Len(t, devices, 6)

	devices = cache.GetDevicesByType("Stratum")
	assert.Len(t, devices, 4)

	devices = cache.GetDevicesByVersion("Stratum", "1.0.0")
	assert.Len(t, devices, 2)

	devices = cache.GetDevicesByID("device-old-ss")
	assert.Len(t, devices, 1)
	assert.Equal(t, devicebase.Version("2.0.0"), devices[0].Version)
	assert.Equal(t, devicebase.Type("TestDevice"), devices[0].Type)

	/////////// unregister the first watcher ///////////////////
	watcher1Ctx.Close()

	// Make another change
	chNwChanges <- stream.Event{
		Type: stream.Created,
		Object: &networkchange.NetworkChange{
			ID:    "network-change-4",
			Index: 1,
			Changes: []*devicechange.Change{
				{
					DeviceID:      "device-1",
					DeviceType:    "Stratum",
					DeviceVersion: "4.0.0",
				},
			},
		},
	}

	////////////// Send a deleted event - should be ignored ////////////////////
	chNwChanges <- stream.Event{
		Type: stream.Deleted,
		Object: &networkchange.NetworkChange{
			ID:    "network-change-4",
			Index: 1,
			Changes: []*devicechange.Change{
				{
					DeviceID:      "device-1",
					DeviceType:    "Stratum",
					DeviceVersion: "5.0.0",
				},
			},
		},
	}

	// Wait for the test to complete
	time.Sleep(20 * time.Millisecond)
}
