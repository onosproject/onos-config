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
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/test/mocks/store"
	devicechangetype "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetype "github.com/onosproject/onos-config/pkg/types/change/network"
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

	ch := chVal.Load().(chan<- stream.Event)
	ch <- stream.Event{
		Type: stream.Created,
		Object: &networkchangetype.NetworkChange{
			ID:    "network-change-1",
			Index: 1,
			Changes: []*devicechangetype.Change{
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
		Object: &networkchangetype.NetworkChange{
			ID:    "network-change-1",
			Index: 1,
			Changes: []*devicechangetype.Change{
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
		Object: &networkchangetype.NetworkChange{
			ID:    "network-change-1",
			Index: 1,
			Changes: []*devicechangetype.Change{
				{
					DeviceID:      "device-3",
					DeviceType:    "NotStratum",
					DeviceVersion: "1.0.0",
				},
			},
		},
	}

	// Need to wait for the event to be read by the cache
	time.Sleep(1 * time.Second)

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
}
