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
	"fmt"
	"github.com/golang/mock/gomock"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	devicechangeutils "github.com/onosproject/onos-config/pkg/store/change/device/utils"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io"
	"testing"
)

const (
	device1  = devicetopo.ID("device-1")
	device2  = devicetopo.ID("device-2")
	dcDevice = devicetopo.ID("disconnected")
)

const (
	change1 = devicechangetypes.ID("device-1:device-1")
	change2 = devicechangetypes.ID("device-2:device-2")
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

func TestReconcilerChangeThenRollback(t *testing.T) {
	devices, deviceChanges := newStores(t)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	// Create a device-1 change 1
	deviceChange1 := newChangeInterface(device1, 1)
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err := reconciler.Reconcile(types.ID(deviceChange1.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// No changes should have been made
	deviceChange1, err = deviceChanges.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_PENDING, deviceChange1.Status.State)

	// Change the state of device-1 change 1 to RUNNING
	deviceChange1.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange1.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Should be complete by now
	deviceChange1, err = deviceChanges.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_COMPLETE, deviceChange1.Status.State)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)

	//**********************************************************
	// Create eth2 in a second change
	//**********************************************************
	deviceChange2 := newChangeInterface(device1, 2)
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Change the state of device-1 change 1 to RUNNING
	deviceChange2.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Should be complete by now
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)

	paths, err := devicechangeutils.ExtractFullConfig(device1, nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 4, len(paths))
	for _, p := range paths {
		switch p.Path {
		case "/interfaces/interface[name=eth1]/config/name":
			assert.Equal(t, "eth1", p.Value.ValueToString())
		case "/interfaces/interface[name=eth1]/config/enabled":
			assert.Equal(t, "UP", p.Value.ValueToString())
		case "/interfaces/interface[name=eth2]/config/name":
			assert.Equal(t, "eth2", p.Value.ValueToString())
		case "/interfaces/interface[name=eth2]/config/enabled":
			assert.Equal(t, "DOWN", p.Value.ValueToString())
		default:
			t.Errorf("Unexpected path %s", p.Path)
		}
	}

	//**********************************************************
	// Change devicechange2 to rollback
	//**********************************************************
	deviceChange2.Status.State = change.State_PENDING
	deviceChange2.Status.Phase = change.Phase_ROLLBACK
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Change the state of device-1 change 1 to RUNNING
	deviceChange2.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Should be complete by now
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange2.Status.Phase)

	paths, err = devicechangeutils.ExtractFullConfig(device1, nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 2, len(paths))
	for _, p := range paths {
		switch p.Path {
		case "/interfaces/interface[name=eth1]/config/name":
			assert.Equal(t, "eth1", p.Value.ValueToString())
		case "/interfaces/interface[name=eth1]/config/enabled":
			assert.Equal(t, "UP", p.Value.ValueToString())
		default:
			t.Errorf("Unexpected path %s", p.Path)
		}
	}
}

// TestReconcilerRemoveThenRollback creates an eth1 with 2 attribs. Then the
// interface is removed (at root). Then this delete is rolled back and the 2
// attributes become visible again
func TestReconcilerRemoveThenRollback(t *testing.T) {
	devices, deviceChanges := newStores(t)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	//**********************************************
	// First create an interface eth1
	//**********************************************
	deviceChange1 := newChangeInterface(device1, 1)
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err := reconciler.Reconcile(types.ID(deviceChange1.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Change the state of device-1 change 1 to RUNNING
	deviceChange1.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange1.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Should be complete by now
	deviceChange1, err = deviceChanges.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_COMPLETE, deviceChange1.Status.State)
	assert.Equal(t, change.Phase_CHANGE, deviceChange1.Status.Phase)

	paths, err := devicechangeutils.ExtractFullConfig(device1, nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 2, len(paths))
	for _, p := range paths {
		switch p.Path {
		case "/interfaces/interface[name=eth1]/config/name":
			assert.Equal(t, "eth1", p.Value.ValueToString())
		case "/interfaces/interface[name=eth1]/config/enabled":
			assert.Equal(t, "UP", p.Value.ValueToString())
		default:
			t.Errorf("Unexpected path %s", p.Path)
		}
	}

	//**********************************************
	// Then remove the interface eth1
	//**********************************************
	deviceChange2 := newChangeInterfaceRemove(device1, 1)
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Change the state of device-1 change 1 to RUNNING
	deviceChange2.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Should be complete by now
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, change.Phase_CHANGE, deviceChange2.Status.Phase)

	paths, err = devicechangeutils.ExtractFullConfig(device1, nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 0, len(paths))

	//**********************************************************
	// Now rollback the remove
	//**********************************************************
	deviceChange2.Status.State = change.State_PENDING
	deviceChange2.Status.Phase = change.Phase_ROLLBACK
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Change the state of device-1 change 1 to RUNNING
	deviceChange2.Status.State = change.State_RUNNING
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	ok, err = reconciler.Reconcile(types.ID(deviceChange2.ID))
	assert.NoError(t, err)
	assert.True(t, ok)

	// Should be complete by now
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Equal(t, change.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, change.Phase_ROLLBACK, deviceChange2.Status.Phase)

	paths, err = devicechangeutils.ExtractFullConfig(device1, nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 2, len(paths))
	for _, p := range paths {
		switch p.Path {
		case "/interfaces/interface[name=eth1]/config/name":
			assert.Equal(t, "eth1", p.Value.ValueToString())
		case "/interfaces/interface[name=eth1]/config/enabled":
			assert.Equal(t, "UP", p.Value.ValueToString())
		default:
			t.Errorf("Unexpected path %s", p.Path)
		}
	}

}

func newStores(t *testing.T) (devicestore.Store, devicechanges.Store) {
	ctrl := gomock.NewController(t)

	devices := map[devicetopo.ID]*devicetopo.Device{
		device1: {
			ID: device1,
			Protocols: []*devicetopo.ProtocolState{
				{
					Protocol:          devicetopo.Protocol_GNMI,
					ConnectivityState: devicetopo.ConnectivityState_REACHABLE,
					ChannelState:      devicetopo.ChannelState_CONNECTED,
				},
			},
		},
		device2: {
			ID: device2,
			Protocols: []*devicetopo.ProtocolState{
				{
					Protocol:          devicetopo.Protocol_GNMI,
					ConnectivityState: devicetopo.ConnectivityState_REACHABLE,
					ChannelState:      devicetopo.ChannelState_CONNECTED,
				},
			},
		},
		dcDevice: {
			ID: dcDevice,
			Protocols: []*devicetopo.ProtocolState{
				{
					Protocol:          devicetopo.Protocol_GNMI,
					ConnectivityState: devicetopo.ConnectivityState_UNREACHABLE,
					ChannelState:      devicetopo.ChannelState_DISCONNECTED,
				},
			},
		},
	}

	stream := NewMockDeviceService_ListClient(ctrl)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: devices[device1]}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: devices[device2]}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: devices[dcDevice]}, nil)
	stream.EXPECT().Recv().Return(nil, io.EOF)

	client := NewMockDeviceServiceClient(ctrl)
	client.EXPECT().List(gomock.Any(), gomock.Any()).Return(stream, nil).AnyTimes()
	client.EXPECT().Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, in *devicetopo.GetRequest, opts ...grpc.CallOption) (*devicetopo.GetResponse, error) {
			return &devicetopo.GetResponse{
				Device: devices[in.ID],
			}, nil
		}).AnyTimes()

	deviceStore, err := devicestore.NewStore(client)
	assert.NoError(t, err)
	deviceChanges, err := devicechanges.NewLocalStore()
	assert.NoError(t, err)
	return deviceStore, deviceChanges
}

func newChange(device devicetopo.ID) *devicechangetypes.DeviceChange {
	return &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(device),
			Index: 1,
		},
		Change: &devicechangetypes.Change{
			DeviceID: device,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:  "foo",
					Value: devicechangetypes.NewTypedValueString("Hello world!"),
				},
			},
		},
	}
}

// newChangeInterface creates a new interface eth<n> in the OpenConfig model style
func newChangeInterface(device devicetopo.ID, iface int) *devicechangetypes.DeviceChange {
	ifaceID := fmt.Sprintf("eth%d", iface)
	ifacePath := fmt.Sprintf("/interfaces/interface[name=%s]/config/", ifaceID)
	enabled := "UP"
	if iface%2 == 0 {
		enabled = "DOWN"
	}

	return &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-added", device, iface)),
			Index: types.Index(iface),
		},
		Change: &devicechangetypes.Change{
			DeviceID: device,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:    ifacePath + "name",
					Value:   devicechangetypes.NewTypedValueString(ifaceID),
					Removed: false,
				},
				{
					Path:    ifacePath + "enabled",
					Value:   devicechangetypes.NewTypedValueString(enabled),
					Removed: false,
				},
			},
		},
	}
}

// newChangeInterfaceRemove removes an interface eth<n> of the OpenConfig model style
func newChangeInterfaceRemove(device devicetopo.ID, iface int) *devicechangetypes.DeviceChange {
	ifaceID := fmt.Sprintf("eth%d", iface)
	ifacePath := fmt.Sprintf("/interfaces/interface[name=%s]", ifaceID)

	return &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-removed", device, iface)),
			Index: types.Index(iface),
		},
		Change: &devicechangetypes.Change{
			DeviceID: device,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:    ifacePath,
					Removed: true,
				},
			},
		},
	}
}
