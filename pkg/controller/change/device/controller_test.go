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
	"github.com/onosproject/onos-config/api/types"
	"github.com/onosproject/onos-config/api/types/change"
	devicechangetypes "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	devicechangeutils "github.com/onosproject/onos-config/pkg/store/change/device/utils"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	southboundmock "github.com/onosproject/onos-config/pkg/test/mocks/southbound"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io"
	log "k8s.io/klog"
	"os"
	"testing"
	"time"
)

const (
	device1     = device.ID("device-1")
	device2     = device.ID("device-2")
	dcDevice    = device.ID("disconnected")
	v1          = "1.0.0"
	stratumType = "Stratum"
)

const (
	change1 = devicechangetypes.ID("device-1:device-1:1.0.0")
	change2 = devicechangetypes.ID("device-2:device-2:1.0.0")
)

const (
	healthUp        = "UP"
	healthDown      = "DOWN"
	ethPrefix       = "eth"
	eth1            = ethPrefix + "1"
	eth2            = ethPrefix + "2"
	ifHealthAttr    = "health-indicator"
	ifNameAttr      = "name"
	ifEnabledAttr   = "enabled"
	ifConfigNameFmt = "/interfaces/interface[name=eth%d]/config/"
	eth1Name        = "/interfaces/interface[name=eth1]/config/name"
	eth1Enabled     = "/interfaces/interface[name=eth1]/config/enabled"
	eth1Hi          = "/interfaces/interface[name=eth1]/config/health-indicator"
	eth1Desc        = "/interfaces/interface[name=eth1]/config/description"
	eth2Base        = "/interfaces/interface[name=eth2]"
	eth2Name        = "/interfaces/interface[name=eth2]/config/name"
	eth2Enabled     = "/interfaces/interface[name=eth2]/config/enabled"
	eth2Hi          = "/interfaces/interface[name=eth2]/config/health-indicator"
)

func TestReconcilerChangeSuccess(t *testing.T) {
	devices, deviceChanges := newStores(t)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	// Create a device-1 change 1
	deviceChange1 := newChange(device1, v1)
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-2 change 1
	deviceChange2 := newChange(device2, v1)
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
	deviceChange1 := newChange(device1, v1)
	deviceChange1.Status.Phase = change.Phase_ROLLBACK
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-2 change 1
	deviceChange2 := newChange(device2, v1)
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
	deviceChange1 := newChangeInterface(device1, v1, 1)
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
	deviceChange2 := newChangeInterface(device1, v1, 2)
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

	paths, err := devicechangeutils.ExtractFullConfig(device.NewVersionedID(device1, v1), nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 6, len(paths))
	for _, p := range paths {
		switch p.Path {
		case eth1Name:
			assert.Equal(t, eth1, p.Value.ValueToString())
		case eth1Enabled:
			assert.Equal(t, "false", p.Value.ValueToString())
		case eth1Hi:
			assert.Equal(t, healthUp, p.Value.ValueToString())
		case eth2Name:
			assert.Equal(t, eth2, p.Value.ValueToString())
		case eth2Enabled:
			assert.Equal(t, "true", p.Value.ValueToString())
		case eth2Hi:
			assert.Equal(t, healthDown, p.Value.ValueToString())
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

	paths, err = devicechangeutils.ExtractFullConfig(device.NewVersionedID(device1, v1), nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 3, len(paths))
	for _, p := range paths {
		switch p.Path {
		case eth1Name:
			assert.Equal(t, eth1, p.Value.ValueToString())
		case eth1Enabled:
			assert.Equal(t, "false", p.Value.ValueToString())
		case eth1Hi:
			assert.Equal(t, healthUp, p.Value.ValueToString())
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
	deviceChange1 := newChangeInterface(device1, v1, 1)
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

	paths, err := devicechangeutils.ExtractFullConfig(device.NewVersionedID(device1, v1), nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 3, len(paths))
	for _, p := range paths {
		switch p.Path {
		case eth1Name:
			assert.Equal(t, eth1, p.Value.ValueToString())
		case eth1Enabled:
			assert.Equal(t, "false", p.Value.ValueToString())
		case eth1Hi:
			assert.Equal(t, healthUp, p.Value.ValueToString())
		default:
			t.Errorf("Unexpected path %s", p.Path)
		}
	}

	//**********************************************
	// Then remove the interface eth1
	//**********************************************
	deviceChange2 := newChangeInterfaceRemove(device1, v1, 1)
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

	paths, err = devicechangeutils.ExtractFullConfig(device.NewVersionedID(device1, v1), nil, deviceChanges, 0)
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

	paths, err = devicechangeutils.ExtractFullConfig(device.NewVersionedID(device1, v1), nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 3, len(paths))
	for _, p := range paths {
		switch p.Path {
		case eth1Name:
			assert.Equal(t, eth1, p.Value.ValueToString())
		case eth1Enabled:
			assert.Equal(t, "false", p.Value.ValueToString())
		case eth1Hi:
			assert.Equal(t, healthUp, p.Value.ValueToString())
		default:
			t.Errorf("Unexpected path %s", p.Path)
		}
	}

}

func newStores(t *testing.T) (devicestore.Store, devicechanges.Store) {
	log.SetOutput(os.Stdout)

	ctrl := gomock.NewController(t)

	devices := map[devicetopo.ID]*devicetopo.Device{
		devicetopo.ID(device1): {
			ID: devicetopo.ID(device1),
			Protocols: []*devicetopo.ProtocolState{
				{
					Protocol:          devicetopo.Protocol_GNMI,
					ConnectivityState: devicetopo.ConnectivityState_REACHABLE,
					ChannelState:      devicetopo.ChannelState_CONNECTED,
				},
			},
		},
		devicetopo.ID(device2): {
			ID: devicetopo.ID(device2),
			Protocols: []*devicetopo.ProtocolState{
				{
					Protocol:          devicetopo.Protocol_GNMI,
					ConnectivityState: devicetopo.ConnectivityState_REACHABLE,
					ChannelState:      devicetopo.ChannelState_CONNECTED,
				},
			},
		},
		devicetopo.ID(dcDevice): {
			ID: devicetopo.ID(dcDevice),
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
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: devices[devicetopo.ID(device1)]}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: devices[devicetopo.ID(device2)]}, nil)
	stream.EXPECT().Recv().Return(&devicetopo.ListResponse{Device: devices[devicetopo.ID(dcDevice)]}, nil)
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

	mockTargetDevice(t, device1, ctrl)
	mockTargetDevice(t, device2, ctrl)

	return deviceStore, deviceChanges
}

func mockTargetDevice(t *testing.T, name device.ID, ctrl *gomock.Controller) {
	modelData := gnmi.ModelData{
		Name:         "test1",
		Organization: "Open Networking Foundation",
		Version:      "2018-02-20",
	}
	timeout := time.Millisecond * 200
	mockDevice := devicetopo.Device{
		ID:          devicetopo.ID(name),
		Address:     "1.2.3.4:11161",
		Version:     v1,
		Timeout:     &timeout,
		Credentials: devicetopo.Credentials{},
		TLS:         devicetopo.TlsConfig{},
		Type:        "TestDevice",
		Role:        "leaf",
	}

	mockTargetDevice := southboundmock.NewMockTargetIf(ctrl)
	mockTargetDeviceCtx := context.TODO()

	mockTargetDevice.EXPECT().ConnectTarget(
		gomock.Any(),
		mockDevice,
	).Return(devicetopo.ID(name), nil)

	mockTargetDevice.EXPECT().CapabilitiesWithString(
		gomock.Any(),
		"",
	).Return(&gnmi.CapabilityResponse{
		SupportedModels:    []*gnmi.ModelData{&modelData},
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_JSON},
		GNMIVersion:        "0.7.0",
	}, nil)

	mockTargetDevice.EXPECT().Context().Return(&mockTargetDeviceCtx).AnyTimes()

	mockTargetDevice.EXPECT().Set(gomock.Any(), gomock.Any()).Return(&gnmi.SetResponse{
		Response: []*gnmi.UpdateResult{},
	}, nil).AnyTimes()

	//topoChannel := make(chan *devicetopo.ListResponse)
	//dispatcher := dispatcher.NewDispatcher()
	//modelregistry := new(modelregistry.ModelRegistry)
	opStateCache := make(devicechangetypes.TypedValueMap)
	roPathMap := make(modelregistry.ReadOnlyPathMap)

	_, err := synchronizer.New(context.Background(), &mockDevice,
		make(chan<- events.OperationalStateEvent), make(chan<- events.DeviceResponse),
		opStateCache, roPathMap, mockTargetDevice,
		modelregistry.GetStateExplicitRoPaths)
	assert.NoError(t, err, "Unable to create new synchronizer for", mockDevice.ID)

	// Finally to make it visible to tests - add it to `Targets`
	southbound.Targets[devicetopo.ID(name)] = mockTargetDevice
}

func newChange(device device.ID, version device.Version) *devicechangetypes.DeviceChange {
	return &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(device),
			Index: 1,
		},
		Change: &devicechangetypes.Change{
			DeviceID:      device,
			DeviceVersion: version,
			DeviceType:    stratumType,
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
func newChangeInterface(device device.ID, version device.Version, iface int) *devicechangetypes.DeviceChange {
	ifaceID := fmt.Sprintf("%s%d", ethPrefix, iface)
	ifacePath := fmt.Sprintf(ifConfigNameFmt, iface)
	healthValue := healthUp
	if iface%2 == 0 {
		healthValue = healthDown
	}

	return &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-added", device, iface)),
			Index: types.Index(iface),
		},
		Change: &devicechangetypes.Change{
			DeviceID:      device,
			DeviceVersion: version,
			DeviceType:    stratumType,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:    ifacePath + ifNameAttr,
					Value:   devicechangetypes.NewTypedValueString(ifaceID),
					Removed: false,
				},
				{
					Path:    ifacePath + ifEnabledAttr,
					Value:   devicechangetypes.NewTypedValueBool(iface%2 == 0),
					Removed: false,
				},
				{
					Path:    ifacePath + ifHealthAttr,
					Value:   devicechangetypes.NewTypedValueString(healthValue),
					Removed: false,
				},
			},
		},
	}
}

// newChangeInterfaceRemove removes an interface eth<n> of the OpenConfig model style
func newChangeInterfaceRemove(device device.ID, version device.Version, iface int) *devicechangetypes.DeviceChange {
	ifacePath := fmt.Sprintf(ifConfigNameFmt, iface)

	return &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-removed", device, iface)),
			Index: types.Index(iface),
		},
		Change: &devicechangetypes.Change{
			DeviceID:      device,
			DeviceVersion: version,
			DeviceType:    stratumType,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:    ifacePath,
					Removed: true,
				},
			},
		},
	}
}
