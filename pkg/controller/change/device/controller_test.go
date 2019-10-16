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
	"context"
	"github.com/golang/mock/gomock"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io"
	"testing"
)

const (
	device1  = device.ID("device-1")
	device2  = device.ID("device-2")
	dcDevice = device.ID("disconnected")
)

const (
	change1 = devicechange.ID("device-1:device-1")
	change2 = devicechange.ID("device-2:device-2")
)

func TestReconcilerChangeSuccess(t *testing.T) {
	devices, deviceChanges := newStores(t)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	// Create a device-1 change 1
	deviceChange1 := newChange(device1)
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-2 change 1
	deviceChange2 := newChange(device2)
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	// Apply change 1 to the reconciler
	ok, err := reconciler.Reconcile(types.ID(deviceChange1.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Apply change 2 to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// No changes should have been made
	deviceChange1, err = deviceChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)

	deviceChange2, err = deviceChanges.Get(change2)
	assert.NoError(t, err)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Change the state of device-1 change 1 to RUNNING
	deviceChange1.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Change the state of device-2 change 1 to RUNNING
	deviceChange2.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply the changes to the reconciler again
	ok, err = reconciler.Reconcile(types.ID(deviceChange1.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Both change should be applied successfully
	deviceChange1, err = deviceChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.State_COMPLETE, deviceChange1.Status.State)

	deviceChange2, err = deviceChanges.Get(change2)
	assert.NoError(t, err)
	assert.Equal(t, change.State_COMPLETE, deviceChange2.Status.State)
}

func TestReconcilerRollbackSuccess(t *testing.T) {
	devices, deviceChanges := newStores(t)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	// Create a device-1 change 1
	deviceChange1 := newChange(device1)
	deviceChange1.Status.Phase = change.Phase_ROLLBACK
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-2 change 1
	deviceChange2 := newChange(device2)
	deviceChange2.Status.Phase = change.Phase_ROLLBACK
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	// Apply change 1 to the reconciler
	ok, err := reconciler.Reconcile(types.ID(deviceChange1.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Apply change 2 to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// No changes should have been made
	deviceChange1, err = deviceChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)

	deviceChange2, err = deviceChanges.Get(change2)
	assert.NoError(t, err)
	assert.Equal(t, change.State_PENDING, deviceChange2.Status.State)

	// Change the state of device-1 change 1 to RUNNING
	deviceChange1.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Change the state of device-2 change 1 to RUNNING
	deviceChange2.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply the changes to the reconciler again
	ok, err = reconciler.Reconcile(types.ID(deviceChange1.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Both change should be applied successfully
	deviceChange1, err = deviceChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, change.State_COMPLETE, deviceChange1.Status.State)

	deviceChange2, err = deviceChanges.Get(change2)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange2.Status.Phase)
	assert.Equal(t, change.State_COMPLETE, deviceChange2.Status.State)
}

func newStores(t *testing.T) (devicestore.Store, devicechanges.Store) {
	ctrl := gomock.NewController(t)

	devices := map[device.ID]*device.Device{
		device1: {
			ID: device1,
			Protocols: []*device.ProtocolState{
				{
					Protocol:          device.Protocol_GNMI,
					ConnectivityState: device.ConnectivityState_REACHABLE,
					ChannelState:      device.ChannelState_CONNECTED,
				},
			},
		},
		device2: {
			ID: device2,
			Protocols: []*device.ProtocolState{
				{
					Protocol:          device.Protocol_GNMI,
					ConnectivityState: device.ConnectivityState_REACHABLE,
					ChannelState:      device.ChannelState_CONNECTED,
				},
			},
		},
		dcDevice: {
			ID: dcDevice,
			Protocols: []*device.ProtocolState{
				{
					Protocol:          device.Protocol_GNMI,
					ConnectivityState: device.ConnectivityState_UNREACHABLE,
					ChannelState:      device.ChannelState_DISCONNECTED,
				},
			},
		},
	}

	stream := NewMockDeviceService_ListClient(ctrl)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: devices[device1]}, nil)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: devices[device2]}, nil)
	stream.EXPECT().Recv().Return(&device.ListResponse{Device: devices[dcDevice]}, nil)
	stream.EXPECT().Recv().Return(nil, io.EOF)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().List(gomock.Any(), gomock.Any()).Return(stream, nil).AnyTimes()
	client.EXPECT().Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, in *device.GetRequest, opts ...grpc.CallOption) (*device.GetResponse, error) {
			return &device.GetResponse{
				Device: devices[in.ID],
			}, nil
		}).AnyTimes()

	deviceStore, err := devicestore.NewStore(client)
	assert.NoError(t, err)
	deviceChanges, err := devicechanges.NewLocalStore()
	assert.NoError(t, err)
	return deviceStore, deviceChanges
}

func newChange(device device.ID) *devicechange.DeviceChange {
	return &devicechange.DeviceChange{
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(device),
			Index: 1,
		},
		Change: &devicechange.Change{
			DeviceID: device,
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
	}
}
