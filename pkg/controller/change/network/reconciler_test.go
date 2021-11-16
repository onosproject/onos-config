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
	"github.com/atomix/atomix-go-client/pkg/atomix"
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-api/go/onos/config/change"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	"github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-api/go/onos/topo"
	devicetopo "github.com/onosproject/onos-config/pkg/device"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchanges "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/test/mocks"
	mockcache "github.com/onosproject/onos-config/pkg/test/mocks/store/cache"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

const (
	device1     = device.ID("device-1")
	device2     = device.ID("device-2")
	v1          = "1.0.0"
	stratumType = "Stratum"
)

const (
	change1 = networkchange.ID("change-1")
)

// TestReconcilerChangeRollback tests applying and then rolling back a change
func TestReconcilerChangeRollback(t *testing.T) {
	t.Skip()
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	assert.NoError(t, test.Start())
	defer test.Stop()

	atomixClient, err := test.NewClient("test")
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkChanges, deviceChanges, devices := newStores(t, ctrl, atomixClient)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		networkChanges: networkChanges,
		deviceChanges:  deviceChanges,
		devices:        devices,
	}

	var requeue controller.Result

	// Create a network change
	networkChange := newChange(change1, device1, device2)
	err = networkChanges.Create(networkChange)
	assert.NoError(t, err)

	// Reconcile the network change
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.NoError(t, err)
	assert.Equal(t, "change-1", requeue.Requeue.String())

	// Verify that device changes were created
	deviceChange1, err := deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err := deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the network change again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.NoError(t, err)
	assert.Nil(t, requeue.Requeue.Value)

	// The reconciler should have changed its state to PENDING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// But device change states should remain in PENDING state
	deviceChange1, err = deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the network change again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.NoError(t, err)
	assert.Nil(t, requeue.Requeue.Value)

	// Verify that device change states were changed to PENDING
	deviceChange1, err = deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Complete one of the devices
	deviceChange1.Status.State = change.State_COMPLETE
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Reconcile the network change again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.EqualError(t, err, "waiting for device change(s) to complete change-1")
	assert.Nil(t, requeue.Requeue.Value)

	// Verify the network change was not completed
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Complete the other device
	deviceChange2.Status.State = change.State_COMPLETE
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Reconcile the network change one more time
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.NoError(t, err)
	assert.Nil(t, requeue.Requeue.Value)

	// Verify the network change is complete
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_COMPLETE, networkChange.Status.State)

	// Set the change to the ROLLBACK phase
	networkChange.Status.Phase = change.Phase_ROLLBACK
	networkChange.Status.State = change.State_PENDING
	err = networkChanges.Update(networkChange)
	assert.NoError(t, err)

	// Reconcile the rollback
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.NoError(t, err)
	assert.Nil(t, requeue.Requeue.Value)

	// Verify that the rollback is running
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Verify that device change phases were changed to ROLLBACK
	deviceChange1, err = deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the rollback again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.EqualError(t, err, "waiting for device change(s) to complete change-1")
	assert.Nil(t, requeue.Requeue.Value)

	// Verify that the rollback is running
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Reconcile the rollback again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.EqualError(t, err, "waiting for device change(s) to complete change-1")
	assert.Nil(t, requeue.Requeue.Value)

	// Verify that device change states were changed to PENDING
	deviceChange1, err = deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)
}

// TestReconcilerError tests an error reverting a change to PENDING
func TestReconcilerError(t *testing.T) {
	t.Skip()
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	assert.NoError(t, test.Start())
	defer test.Stop()

	atomixClient, err := test.NewClient("test")
	assert.NoError(t, err)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkChanges, deviceChanges, devices := newStores(t, ctrl, atomixClient)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		networkChanges: networkChanges,
		deviceChanges:  deviceChanges,
		devices:        devices,
	}
	var requeue controller.Result

	// Create a network change
	networkChange := newChange(change1, device1, device2)
	err = networkChanges.Create(networkChange)
	assert.NoError(t, err)

	// Reconcile the network change
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.NoError(t, err)
	assert.Equal(t, "change-1", requeue.Requeue.String())

	// Verify that device changes were created
	deviceChange1, err := deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err := deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the network change again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.NoError(t, err)
	assert.Nil(t, requeue.Requeue.Value)

	// The reconciler should have changed its state to PENDING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// But device change states should remain in PENDING state
	deviceChange1, err = deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Reconcile the network change again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.NoError(t, err)
	assert.Nil(t, requeue.Requeue.Value)

	// Verify that device change states were changed to PENDING
	deviceChange1, err = deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Complete one of the devices
	deviceChange1.Status.State = change.State_COMPLETE
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Reconcile the network change again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.EqualError(t, err, "waiting for device change(s) to complete change-1")
	assert.Nil(t, requeue.Requeue.Value)

	// Verify the network change was not completed
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Fail the other device
	deviceChange2, err = deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	deviceChange2.Status.State = change.State_FAILED
	deviceChange2.Status.Reason = change.Reason_ERROR
	deviceChange2.Status.Message = "failed for test"
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Reconcile the network change again
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.Error(t, err) // Will trigger retry with exp backoff

	// Verify the network change is still PENDING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Verify the change to device-1 is being rolled back
	deviceChange1, err = deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)
	deviceChange2, err = deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Set the device-1 change rollback to COMPLETE
	deviceChange1.Status.State = change.State_COMPLETE
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Reconcile the network change
	requeue, err = reconciler.Reconcile(controller.NewID(string(networkChange.ID)))
	assert.Error(t, err) // Will trigger retry with exp backoff

	// Verify that the network change returned to PENDING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)
	assert.Equal(t, change.Reason_ERROR, networkChange.Status.Reason)
	assert.Equal(t, "change rejected by device", networkChange.Status.Message)
}

func newStores(t *testing.T, ctrl *gomock.Controller, atomixClient atomix.Client) (networkchanges.Store, devicechanges.Store, devicestore.Store) {
	networkChanges, err := networkchanges.NewAtomixStore(atomixClient)
	assert.NoError(t, err)
	deviceChanges, err := devicechanges.NewAtomixStore(atomixClient)
	assert.NoError(t, err)
	client := mocks.NewMockTopoClient(ctrl)
	client.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, request *topo.WatchRequest) (topo.Topo_WatchClient, error) {
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
	client.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, request *topo.GetRequest) (*topo.GetResponse, error) {
		return &topo.GetResponse{
			Object: devicetopo.ToObject(&devicetopo.Device{
				ID:      devicetopo.ID(request.ID),
				Version: v1,
				Type:    stratumType,
				Address: fmt.Sprintf("%s:5150", request.ID),
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
	devices, err := devicestore.NewStore(client)
	assert.NoError(t, err)

	return networkChanges, deviceChanges, devices
}

func newDeviceCache(ctrl *gomock.Controller, ids ...device.ID) *mockcache.MockCache {
	cachedDevices := make([]*cache.Info, 0, len(ids))
	for _, id := range ids {
		cachedDevices = append(cachedDevices, &cache.Info{
			DeviceID: id,
			Type:     stratumType,
			Version:  v1,
		})
	}

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
		}).AnyTimes()
	deviceCache.EXPECT().Close().Return(nil).AnyTimes()
	return deviceCache
}

func newChange(id networkchange.ID, devices ...device.ID) *networkchange.NetworkChange {
	changes := make([]*devicechange.Change, len(devices))
	for i, device := range devices {
		changes[i] = &devicechange.Change{
			DeviceID:      device,
			DeviceVersion: v1,
			DeviceType:    stratumType,
			Values: []*devicechange.ChangeValue{
				{
					Path:  "foo",
					Value: devicechange.NewTypedValueString(fmt.Sprintf("Hello %s!", device)),
				},
			},
		}
	}
	return &networkchange.NetworkChange{
		ID:      id,
		Changes: changes,
	}
}
