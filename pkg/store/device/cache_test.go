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
	"github.com/golang/mock/gomock"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/stretchr/testify/assert"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeviceCache(t *testing.T) {
	chVal := &atomic.Value{}
	ctrl := gomock.NewController(t)
	netChangeStore := store.NewMockNetworkChangesStore(ctrl)
	netChangeStore.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ch chan<- stream.Event, opts ...networkchangestore.WatchOption) (stream.Context, error) {
			chVal.Store(ch)
			return stream.NewContext(func() {
			}), nil
		}).AnyTimes()

	cache, err := NewCache(netChangeStore)
	assert.NoError(t, err)

	// Before there are any listeners - create an entry in Network Change store
	ch := chVal.Load().(chan<- stream.Event)
	ch <- stream.Event{
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
				if count == 5 { // Expecting 5 results on chan 1
					breakout = true
				}
			case <-time.After(3 * time.Second):
				t.Fail()
			}
			if breakout {
				break
			}
		}
	}()

	watcher1Ctx, err := cache.Watch(cacheChan1, true)
	assert.NoError(t, err)
	assert.NotNil(t, watcher1Ctx)
	// defer watcher1Ctx.Close() // Will close in this test

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
				t.Fail()
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

	//ch = chVal.Load().(chan<- stream.Event)
	ch <- stream.Event{
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
	ch <- stream.Event{
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
	ch <- stream.Event{
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

	devices = cache.GetDevicesByID("device-2")
	assert.Len(t, devices, 2)

	devices = cache.GetDevices()
	assert.Len(t, devices, 5)

	devices = cache.GetDevicesByType("Stratum")
	assert.Len(t, devices, 4)

	devices = cache.GetDevicesByVersion("Stratum", "1.0.0")
	assert.Len(t, devices, 2)

	/////////// unregister the first watcher ///////////////////
	watcher1Ctx.Close()

	// Make another change
	ch <- stream.Event{
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
	ch <- stream.Event{
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
