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
	netChangeStore.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(func(ch chan<- stream.Event, opts ...networkchangestore.WatchOption) (stream.Context, error) {
		chVal.Store(ch)
		return stream.NewContext(func() {
		}), nil
	}).AnyTimes()

	cache, err := NewCache(netChangeStore)
	assert.NoError(t, err)

	cacheChan := make(chan *Info, 10)
	t.Logf("Watching cache")
	cache.Watch(cacheChan)

	cacheChan2 := make(chan *Info, 10)
	t.Logf("Watching cache a 2nd time")
	cache.Watch(cacheChan2)

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
	assert.Len(t, devices, 1)

	devices = cache.GetDevicesByID("device-2")
	assert.Len(t, devices, 2)

	devices = cache.GetDevices()
	assert.Len(t, devices, 4)

	devices = cache.GetDevicesByType("Stratum")
	assert.Len(t, devices, 3)

	devices = cache.GetDevicesByVersion("Stratum", "1.0.0")
	assert.Len(t, devices, 2)

	go func() {
		count := 0
		for update := range cacheChan {
			t.Logf("Cache updated %v", update)
			count++
		}
		assert.Equal(t, 3, count)
	}()

	go func() {
		count2 := 0
		for update := range cacheChan2 {
			t.Logf("Cache updated %v", update)
			count2++
		}
		assert.Equal(t, 3, count2)
	}()

	// Wait for the test to complete
	time.Sleep(20 * time.Millisecond)

}
