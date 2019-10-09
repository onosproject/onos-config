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
	"github.com/onosproject/onos-config/pkg/controller"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io"
	"testing"
	"time"
)

const (
	device1  = device.ID("device-1")
	device2  = device.ID("device-2")
	dcDevice = device.ID("disconnected")
)

const (
	change1  = devicechange.ID("device-1:1")
	change2  = devicechange.ID("device-2:1")
	dcChange = devicechange.ID("disconnected:1")
)

func TestDeviceChangeApply(t *testing.T) {
	leaderships, devices, deviceChanges := newStores(t)
	defer leaderships.Close()
	defer deviceChanges.Close()

	controller := newController(t, leaderships, devices, deviceChanges)
	defer controller.Stop()

	// Listen for events for device-1 and device-2 changes
	device1Ch := make(chan *devicechange.Change)
	err := deviceChanges.Watch(device1, device1Ch)
	assert.NoError(t, err)

	device2Ch := make(chan *devicechange.Change)
	err = deviceChanges.Watch(device2, device2Ch)
	assert.NoError(t, err)

	// Create a device-1 change 1
	deviceChange1 := newChange(device1)
	err = deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	event := nextDeviceEvent(t, device1Ch)
	assert.Equal(t, change1, event.ID)
	assert.Equal(t, device1, event.DeviceID)

	// Create a device-1 change 1
	deviceChange2 := newChange(device2)
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	event = nextDeviceEvent(t, device2Ch)
	assert.Equal(t, change2, event.ID)
	assert.Equal(t, device2, event.DeviceID)

	// Change the state of device-1 change 1 to APPLYING
	deviceChange1.Status.State = change.State_APPLYING
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Change the state of device-2 change 1 to APPLYING
	deviceChange2.Status.State = change.State_APPLYING
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Events for both device changes should be received
	event = nextDeviceEvent(t, device1Ch)
	assert.Equal(t, change1, event.ID)
	assert.Equal(t, device1, event.DeviceID)
	assert.Equal(t, change.State_APPLYING, event.Status.State)

	event = nextDeviceEvent(t, device2Ch)
	assert.Equal(t, change2, event.ID)
	assert.Equal(t, device2, event.DeviceID)
	assert.Equal(t, change.State_APPLYING, event.Status.State)

	// Both change should be applied successfully
	event = nextDeviceEvent(t, device1Ch)
	assert.Equal(t, change1, event.ID)
	assert.Equal(t, device1, event.DeviceID)
	assert.Equal(t, change.State_SUCCEEDED, event.Status.State)

	event = nextDeviceEvent(t, device2Ch)
	assert.Equal(t, change2, event.ID)
	assert.Equal(t, device2, event.DeviceID)
	assert.Equal(t, change.State_SUCCEEDED, event.Status.State)
}

func TestDeviceChangeFailError(t *testing.T) {
	// TODO
}

func TestDeviceChangeFailAvailability(t *testing.T) {
	leaderships, devices, deviceChanges := newStores(t)
	defer leaderships.Close()
	defer deviceChanges.Close()

	controller := newController(t, leaderships, devices, deviceChanges)
	defer controller.Stop()

	// Listen for events for disconnected device changes
	deviceCh := make(chan *devicechange.Change)
	err := deviceChanges.Watch(dcDevice, deviceCh)
	assert.NoError(t, err)

	// Create a disconnected device change 1
	deviceChange := newChange(dcDevice)
	err = deviceChanges.Create(deviceChange)
	assert.NoError(t, err)

	event := nextDeviceEvent(t, deviceCh)
	assert.Equal(t, dcChange, event.ID)
	assert.Equal(t, dcDevice, event.DeviceID)

	// Change the state of disconnected device change to APPLYING
	deviceChange.Status.State = change.State_APPLYING
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	// An event should be received for the APPLYING state change
	event = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, dcChange, event.ID)
	assert.Equal(t, dcDevice, event.DeviceID)
	assert.Equal(t, change.State_APPLYING, event.Status.State)

	// The controller should fail the change with an UNAVAILABLE reason
	event = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, dcChange, event.ID)
	assert.Equal(t, dcDevice, event.DeviceID)
	assert.Equal(t, change.State_FAILED, event.Status.State)
	assert.Equal(t, change.Reason_UNAVAILABLE, event.Status.Reason)
}

func TestDeviceChangeRollback(t *testing.T) {
	leaderships, devices, deviceChanges := newStores(t)
	defer leaderships.Close()
	defer deviceChanges.Close()

	controller := newController(t, leaderships, devices, deviceChanges)
	defer controller.Stop()

	// Listen for events for disconnected device changes
	deviceCh := make(chan *devicechange.Change)
	err := deviceChanges.Watch(device1, deviceCh)
	assert.NoError(t, err)

	// Create a disconnected device change 1
	deviceChange := newChange(device1)
	err = deviceChanges.Create(deviceChange)
	assert.NoError(t, err)

	event := nextDeviceEvent(t, deviceCh)
	assert.Equal(t, change1, event.ID)
	assert.Equal(t, device1, event.DeviceID)

	// Change the state of disconnected device change to APPLYING with the deleted flag set
	deviceChange.Status.State = change.State_APPLYING
	deviceChange.Status.Deleted = true
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	// An event should be received for the APPLYING state change
	event = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, change1, event.ID)
	assert.Equal(t, device1, event.DeviceID)
	assert.Equal(t, change.State_APPLYING, event.Status.State)

	// The controller should succeed the change
	event = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, change1, event.ID)
	assert.Equal(t, device1, event.DeviceID)
	assert.Equal(t, change.State_SUCCEEDED, event.Status.State)
}

func TestDeviceChangeRollbackFailAvailability(t *testing.T) {
	leaderships, devices, deviceChanges := newStores(t)
	defer leaderships.Close()
	defer deviceChanges.Close()

	controller := newController(t, leaderships, devices, deviceChanges)
	defer controller.Stop()

	// Listen for events for disconnected device changes
	deviceCh := make(chan *devicechange.Change)
	err := deviceChanges.Watch(dcDevice, deviceCh)
	assert.NoError(t, err)

	// Create a disconnected device change 1
	deviceChange := newChange(dcDevice)
	err = deviceChanges.Create(deviceChange)
	assert.NoError(t, err)

	event := nextDeviceEvent(t, deviceCh)
	assert.Equal(t, dcChange, event.ID)
	assert.Equal(t, dcDevice, event.DeviceID)

	// Change the state of disconnected device change to APPLYING with the deleted flag set
	deviceChange.Status.State = change.State_APPLYING
	deviceChange.Status.Deleted = true
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	// An event should be received for the APPLYING state change
	event = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, dcChange, event.ID)
	assert.Equal(t, dcDevice, event.DeviceID)
	assert.Equal(t, change.State_APPLYING, event.Status.State)

	// The controller should fail the change with an UNAVAILABLE reason
	event = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, dcChange, event.ID)
	assert.Equal(t, dcDevice, event.DeviceID)
	assert.Equal(t, change.State_FAILED, event.Status.State)
	assert.Equal(t, change.Reason_UNAVAILABLE, event.Status.Reason)
}

func newStores(t *testing.T) (mastership.Store, devicestore.Store, devicechanges.Store) {
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
	mastershipStore, err := mastership.NewLocalStore(t.Name(), cluster.NodeID("node-1"))
	assert.NoError(t, err)
	deviceChanges, err := devicechanges.NewLocalStore()
	assert.NoError(t, err)
	return mastershipStore, deviceStore, deviceChanges
}

func newController(t *testing.T, mastershipStore mastership.Store, devices devicestore.Store, deviceChanges devicechanges.Store) *controller.Controller {
	controller := NewController(mastershipStore, devices, deviceChanges)
	err := controller.Start()
	assert.NoError(t, err)
	return controller
}

func newChange(device device.ID) *devicechange.Change {
	return &devicechange.Change{
		DeviceID: device,
		Values: []*devicechange.Value{
			{
				Path:  "foo",
				Value: []byte("Hello world!"),
				Type:  devicechange.ValueType_STRING,
			},
		},
	}
}

func nextDeviceEvent(t *testing.T, ch chan *devicechange.Change) *devicechange.Change {
	select {
	case e := <-ch:
		return e
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
