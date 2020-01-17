// Copyright 2020-present Open Networking Foundation.
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

package state

import (
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/api/types/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestDeviceStateStore tests that device changes are propagated to the device state store
func TestDeviceStateStore(t *testing.T) {
	changeStore, err := networkchangestore.NewLocalStore()
	assert.NoError(t, err)
	snapshotStore, err := devicesnapstore.NewLocalStore()
	assert.NoError(t, err)

	store, err := NewStore(changeStore, snapshotStore)
	assert.NoError(t, err)
	deviceID := device.NewVersionedID("test", "1.0.0")

	state, err := store.Get(deviceID, 0)
	assert.NoError(t, err)
	assert.Len(t, state, 0)

	change := &networkchange.NetworkChange{
		Changes: []*devicechange.Change{
			{
				DeviceID:      "test",
				DeviceVersion: "1.0.0",
				DeviceType:    "Stratum",
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
	err = changeStore.Create(change)
	assert.NoError(t, err)

	state, err = store.Get(deviceID, change.Revision)
	assert.NoError(t, err)
	assert.Len(t, state, 1)

	change = &networkchange.NetworkChange{
		Changes: []*devicechange.Change{
			{
				DeviceID:      "test",
				DeviceVersion: "1.0.0",
				DeviceType:    "Stratum",
				Values: []*devicechange.ChangeValue{
					{
						Path:    "foo",
						Removed: true,
					},
				},
			},
		},
	}
	err = changeStore.Create(change)
	assert.NoError(t, err)

	state, err = store.Get(deviceID, change.Revision)
	assert.NoError(t, err)
	assert.Len(t, state, 0)
}
