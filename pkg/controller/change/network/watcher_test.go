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
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchange "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNetworkWatcher(t *testing.T) {
	store, err := networkchangestore.NewLocalStore()
	assert.NoError(t, err)
	defer store.Close()

	watcher := &Watcher{
		Store: store,
	}

	ch := make(chan types.ID)
	err = watcher.Start(ch)
	assert.NoError(t, err)

	change1 := &networkchange.NetworkChange{
		Changes: []*devicechange.Change{
			{

				DeviceID: device.ID("device-1"),
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
				DeviceID: device.ID("device-2"),
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

	err = store.Create(change1)
	assert.NoError(t, err)

	select {
	case id := <-ch:
		assert.Equal(t, change1.ID, networkchange.ID(id))
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	change2 := &networkchange.NetworkChange{
		Changes: []*devicechange.Change{
			{
				DeviceID: device.ID("device-1"),
				Values: []*devicechange.Value{
					{
						Path:    "foo",
						Removed: true,
					},
				},
			},
		},
	}

	err = store.Create(change2)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	change1.Status.State = change.State_RUNNING
	err = store.Update(change1)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
}

/*
func TestDeviceWatcher(t *testing.T) {
	ctrl := gomock.NewController(t)

	stream := NewMockDeviceService_ListClient(ctrl)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: &device.Device{ID: device.ID("device-1")}}, nil)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: &device.Device{ID: device.ID("device-2")}}, nil)
	stream.EXPECT().Recv().Return(nil, io.EOF)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().List(gomock.Any(), gomock.Any()).Return(stream, nil).AnyTimes()

	deviceStore, err := devicestore.NewStore(client)
	assert.NoError(t, err)

	changeStore, err := devicechangestore.NewLocalStore()
	assert.NoError(t, err)
	defer changeStore.Close()

	watcher := &DeviceWatcher{
		DeviceStore: deviceStore,
		ChangeStore: changeStore,
	}

	ch := make(chan types.ID)
	err = watcher.Start(ch)
	assert.NoError(t, err)

	change1 := &devicechange.Change{
		NetworkChangeID: types.ID("network:1"),
		DeviceID:        device.ID("device-1"),
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
	}

	err = changeStore.Create(change1)
	assert.NoError(t, err)

	select {
	case id := <-ch:
		assert.Equal(t, change1.NetworkChangeID, id)
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	change2 := &devicechange.Change{
		NetworkChangeID: types.ID("network:2"),
		DeviceID:        device.ID("device-2"),
		Values: []*devicechange.Value{
			{
				Path:  "baz",
				Value: []byte("Goodbye world!"),
				Type:  devicechange.ValueType_STRING,
			},
		},
	}

	err = changeStore.Create(change2)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	change1.Status.State = change.State_RUNNING
	err = changeStore.Update(change1)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
}
*/
