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
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	"github.com/onosproject/onos-api/go/onos/topo"
	devicetopo "github.com/onosproject/onos-config/pkg/device"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/test/mocks"
	mockcache "github.com/onosproject/onos-config/pkg/test/mocks/store/cache"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestNetworkWatcher(t *testing.T) {
	store, err := networkchangestore.NewLocalStore()
	assert.NoError(t, err)
	defer store.Close()

	watcher := &Watcher{
		Store: store,
	}

	ch := make(chan controller.ID)
	err = watcher.Start(ch)
	assert.NoError(t, err)

	change1 := &networkchange.NetworkChange{
		ID: "change-1",
		Changes: []*devicechange.Change{
			{

				DeviceID:      device1,
				DeviceVersion: v1,
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
			{
				DeviceID:      device2,
				DeviceVersion: v1,
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
		},
	}

	err = store.Create(change1)
	assert.NoError(t, err)

	select {
	case id := <-ch:
		assert.Equal(t, string(change1.ID), id.String())
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	change2 := &networkchange.NetworkChange{
		ID: "change-2",
		Changes: []*devicechange.Change{
			{
				DeviceID:      device1,
				DeviceVersion: v1,
				Values: []*devicechange.ChangeValue{
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

	change1.Status.Incarnation++
	err = store.Update(change1)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
}

func TestDeviceWatcher(t *testing.T) {
	ctrl := gomock.NewController(t)

	t.Log("Testing")
	cachedDevices := []*cache.Info{
		{
			DeviceID: device1,
			Type:     v1,
			Version:  v1,
		},
		{
			DeviceID: device2,
			Type:     v1,
			Version:  v1,
		},
	}

	deviceClient := mocks.NewMockTopoClient(ctrl)
	deviceClient.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, request *topo.WatchRequest) (topo.Topo_WatchClient, error) {
		listClient := mocks.NewMockTopo_WatchClient(ctrl)
		listClient.EXPECT().Recv().Return(&topo.WatchResponse{
			Event: topo.Event{
				Type: topo.EventType_NONE,
				Object: *devicetopo.ToObject(&devicetopo.Device{
					ID:      devicetopo.ID(device1),
					Type:    stratumType,
					Version: v1,
					Address: fmt.Sprintf("%s:1234", device1),
					Protocols: []*topo.ProtocolState{
						{
							Protocol:          topo.Protocol_GNMI,
							ChannelState:      topo.ChannelState_CONNECTED,
							ConnectivityState: topo.ConnectivityState_REACHABLE,
						},
					},
				}),
			},
		}, nil)
		listClient.EXPECT().Recv().Return(&topo.WatchResponse{
			Event: topo.Event{
				Type: topo.EventType_NONE,
				Object: *devicetopo.ToObject(&devicetopo.Device{
					ID:      devicetopo.ID(device2),
					Type:    stratumType,
					Version: v1,
					Address: fmt.Sprintf("%s:1234", device2),
					Protocols: []*topo.ProtocolState{
						{
							Protocol:          topo.Protocol_GNMI,
							ChannelState:      topo.ChannelState_CONNECTED,
							ConnectivityState: topo.ConnectivityState_REACHABLE,
						},
					},
				}),
			},
		}, nil)
		listClient.EXPECT().Recv().Return(nil, io.EOF)
		return listClient, nil
	}).AnyTimes()
	deviceClient.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, request *topo.GetRequest) (*topo.GetResponse, error) {
		return &topo.GetResponse{
			Object: devicetopo.ToObject(&devicetopo.Device{
				ID: devicetopo.ID(request.ID),
				Protocols: []*topo.ProtocolState{
					{
						Protocol:          topo.Protocol_GNMI,
						ChannelState:      topo.ChannelState_CONNECTED,
						ConnectivityState: topo.ConnectivityState_REACHABLE,
					},
				},
			}),
		}, nil
	}).AnyTimes()
	deviceStore, err := devicestore.NewStore(deviceClient)
	assert.NoError(t, err)

	deviceCache := mockcache.NewMockCache(ctrl)
	deviceCache.EXPECT().Watch(gomock.Any(), true).DoAndReturn(
		func(ch chan<- stream.Event, replay bool) (stream.Context, error) {
			go func() {
				for _, di := range cachedDevices {
					event := stream.Event{
						Type:   stream.Created,
						Object: di,
					}
					ch <- event
				}
				close(ch)
			}()
			return stream.NewContext(func() {}), nil
		})

	changeStore, err := devicechangestore.NewLocalStore()
	assert.NoError(t, err)
	defer changeStore.Close()

	watcher := &DeviceWatcher{
		DeviceStore: deviceStore,
		DeviceCache: deviceCache,
		ChangeStore: changeStore,
	}

	ch := make(chan controller.ID)
	err = watcher.Start(ch)
	assert.NoError(t, err)

	change1 := &devicechange.DeviceChange{
		Index: 1,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    "network-change-1",
			Index: 1,
		},
		Change: &devicechange.Change{
			DeviceID:      "device-1",
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

	err = changeStore.Create(change1)
	assert.NoError(t, err)

	select {
	case id := <-ch:
		assert.Equal(t, string(change1.NetworkChange.ID), id.String())
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	change2 := &devicechange.DeviceChange{
		Index: 2,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    "network-change-2",
			Index: 2,
		},
		Change: &devicechange.Change{
			DeviceID:      "device-2",
			DeviceVersion: "1.0.0",
			DeviceType:    "Stratum",
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

	err = changeStore.Create(change2)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	change1.Status.Incarnation++
	err = changeStore.Update(change1)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
}
