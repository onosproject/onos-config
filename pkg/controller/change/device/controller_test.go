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
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/golang/mock/gomock"
	types "github.com/onosproject/onos-api/go/onos/config"
	changetypes "github.com/onosproject/onos-api/go/onos/config/change"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-api/go/onos/topo"
	configmodel "github.com/onosproject/onos-config-model/pkg/model"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	devicechangeutils "github.com/onosproject/onos-config/pkg/store/change/device/utils"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/test/mocks"
	southboundmock "github.com/onosproject/onos-config/pkg/test/mocks/southbound"
	storemock "github.com/onosproject/onos-config/pkg/test/mocks/store"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io"
	"sync"
	"testing"
	"time"
)

const (
	device1     = devicetype.ID("device-1")
	device1Addr = "device-1:5150"
	device2     = devicetype.ID("device-2")
	device2Addr = "device-2:5150"
	dcDevice    = devicetype.ID("disconnected")
	v1          = "1.0.0"
	stratumType = "Stratum"
)

const (
	change1 = devicechange.ID("device-1:device-1:1.0.0")
	change2 = devicechange.ID("device-2:device-2:1.0.0")
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
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
		test.WithDebugLogs())
	assert.NoError(t, test.Start())
	defer test.Stop()

	devices, deviceChanges := newStores(t, test)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	// Create a device-1 change 1
	deviceChange1 := newChange(1, device1, v1)
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-2 change 1
	deviceChange2 := newChange(2, device2, v1)
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	// Apply change 1 to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange1.ID)))
	assert.NoError(t, err)

	// Apply change 2 to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// No changes should have been made
	deviceChange1, err = deviceChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_PENDING, deviceChange1.Status.State)

	deviceChange2, err = deviceChanges.Get(change2)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_PENDING, deviceChange2.Status.State)

	// Increment the incarnation number for device-1 change 1
	deviceChange1.Status.Incarnation++
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Increment the incarnation number for device-2 change 1
	deviceChange2.Status.Incarnation++
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply the changes to the reconciler again
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange1.ID)))
	assert.NoError(t, err)

	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Both change should be applied successfully
	deviceChange1, err = deviceChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange1.Status.State)

	deviceChange2, err = deviceChanges.Get(change2)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange2.Status.State)
}

func TestReconcilerRollbackSuccess(t *testing.T) {
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
		test.WithDebugLogs())
	assert.NoError(t, test.Start())
	defer test.Stop()

	devices, deviceChanges := newStores(t, test)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	// Create a device-1 change 1
	deviceChange1 := newChange(1, device1, v1)
	deviceChange1.Status.Phase = changetypes.Phase_ROLLBACK
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Create a device-2 change 1
	deviceChange2 := newChange(2, device2, v1)
	deviceChange2.Status.Phase = changetypes.Phase_ROLLBACK
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	// Apply change 1 to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange1.ID)))
	assert.NoError(t, err)

	// Apply change 2 to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// No changes should have been made
	deviceChange1, err = deviceChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_PENDING, deviceChange1.Status.State)

	deviceChange2, err = deviceChanges.Get(change2)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_PENDING, deviceChange2.Status.State)

	// Increment the incarnation number for device-1 change 1
	deviceChange1.Status.Incarnation++
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Increment the incarnation number for device-2 change 1
	deviceChange2.Status.Incarnation++
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply the changes to the reconciler again
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange1.ID)))
	assert.NoError(t, err)

	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Both change should be applied successfully
	deviceChange1, err = deviceChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange1.Status.State)

	deviceChange2, err = deviceChanges.Get(change2)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.Phase_ROLLBACK, deviceChange2.Status.Phase)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange2.Status.State)
}

func TestReconcilerChangeThenRollback(t *testing.T) {
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
		test.WithDebugLogs())
	assert.NoError(t, test.Start())
	defer test.Stop()

	devices, deviceChanges := newStores(t, test)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	// Create a device-1 change 1
	deviceChange1 := newChangeInterface(1, device1, v1, 1)
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange1.ID)))
	assert.NoError(t, err)

	// No changes should have been made
	deviceChange1, err = deviceChanges.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_PENDING, deviceChange1.Status.State)

	// Increment the incarnation number for device-1 change 1
	deviceChange1.Status.Incarnation++
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange1.ID)))
	assert.NoError(t, err)

	// Should be complete by now
	deviceChange1, err = deviceChanges.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange1.Status.State)
	assert.Equal(t, changetypes.Phase_CHANGE, deviceChange1.Status.Phase)

	//**********************************************************
	// Create eth2 in a second change
	//**********************************************************
	deviceChange2 := newChangeInterface(2, device1, v1, 2)
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Increment the incarnation number for device-1 change 2
	deviceChange2.Status.Incarnation++
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Should be complete by now
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, changetypes.Phase_CHANGE, deviceChange2.Status.Phase)

	paths, err := devicechangeutils.ExtractFullConfig(devicetype.NewVersionedID(device1, v1), nil, deviceChanges, 0)
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
	deviceChange2.Status.State = changetypes.State_PENDING
	deviceChange2.Status.Phase = changetypes.Phase_ROLLBACK
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Increment the incarnation number for device-1 change 2
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	deviceChange2.Status.Incarnation++
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Should be complete by now
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, changetypes.Phase_ROLLBACK, deviceChange2.Status.Phase)

	paths, err = devicechangeutils.ExtractFullConfig(devicetype.NewVersionedID(device1, v1), nil, deviceChanges, 0)
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
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
		test.WithDebugLogs())
	assert.NoError(t, test.Start())
	defer test.Stop()

	devices, deviceChanges := newStores(t, test)
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChanges,
	}

	//**********************************************
	// First create an interface eth1
	//**********************************************
	deviceChange1 := newChangeInterface(1, device1, v1, 1)
	err := deviceChanges.Create(deviceChange1)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange1.ID)))
	assert.NoError(t, err)

	// Increment the incarnation number for device-1 change 1
	deviceChange1.Status.Incarnation++
	err = deviceChanges.Update(deviceChange1)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange1.ID)))
	assert.NoError(t, err)

	// Should be complete by now
	deviceChange1, err = deviceChanges.Get(deviceChange1.ID)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange1.Status.State)
	assert.Equal(t, changetypes.Phase_CHANGE, deviceChange1.Status.Phase)

	paths, err := devicechangeutils.ExtractFullConfig(devicetype.NewVersionedID(device1, v1), nil, deviceChanges, 0)
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
	deviceChange2 := newChangeInterfaceRemove(2, device1, v1, 1)
	err = deviceChanges.Create(deviceChange2)
	assert.NoError(t, err)

	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Increment the incarnation number for device-1 change 2
	deviceChange2.Status.Incarnation++
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Should be complete by now
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, changetypes.Phase_CHANGE, deviceChange2.Status.Phase)

	paths, err = devicechangeutils.ExtractFullConfig(devicetype.NewVersionedID(device1, v1), nil, deviceChanges, 0)
	assert.NoError(t, err, "problem extracting full config")
	assert.Equal(t, 0, len(paths))

	//**********************************************************
	// Now rollback the remove
	//**********************************************************
	deviceChange2.Status.State = changetypes.State_PENDING
	deviceChange2.Status.Phase = changetypes.Phase_ROLLBACK
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Increment the incarnation number for device-1 change 2
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	deviceChange2.Status.Incarnation++
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Apply change to the reconciler
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChange2.ID)))
	assert.NoError(t, err)

	// Should be complete by now
	deviceChange2, err = deviceChanges.Get(deviceChange2.ID)
	assert.NoError(t, err)
	assert.Equal(t, changetypes.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, changetypes.Phase_ROLLBACK, deviceChange2.Status.Phase)

	paths, err = devicechangeutils.ExtractFullConfig(devicetype.NewVersionedID(device1, v1), nil, deviceChanges, 0)
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

func newStores(t *testing.T, test *test.Test) (devicestore.Store, devicechanges.Store) {
	ctrl := gomock.NewController(t)

	devices := map[topodevice.ID]*topodevice.Device{
		topodevice.ID(device1): {
			ID:      topodevice.ID(device1),
			Version: v1,
			Type:    stratumType,
			Address: device1Addr,
			Protocols: []*topo.ProtocolState{
				{
					Protocol:          topo.Protocol_GNMI,
					ConnectivityState: topo.ConnectivityState_REACHABLE,
					ChannelState:      topo.ChannelState_CONNECTED,
				},
			},
		},
		topodevice.ID(device2): {
			ID:      topodevice.ID(device2),
			Version: v1,
			Type:    stratumType,
			Address: device2Addr,
			Protocols: []*topo.ProtocolState{
				{
					Protocol:          topo.Protocol_GNMI,
					ConnectivityState: topo.ConnectivityState_REACHABLE,
					ChannelState:      topo.ChannelState_CONNECTED,
				},
			},
		},
		topodevice.ID(dcDevice): {
			ID:      topodevice.ID(dcDevice),
			Version: v1,
			Type:    stratumType,
			Address: "",
			Protocols: []*topo.ProtocolState{
				{
					Protocol:          topo.Protocol_GNMI,
					ConnectivityState: topo.ConnectivityState_UNREACHABLE,
					ChannelState:      topo.ChannelState_DISCONNECTED,
				},
			},
		},
	}

	stream := mocks.NewMockTopo_WatchClient(ctrl)
	stream.EXPECT().Recv().Return(&topo.WatchResponse{Event: topo.Event{Object: *topodevice.ToObject(devices[topodevice.ID(device1)])}}, nil)
	stream.EXPECT().Recv().Return(&topo.WatchResponse{Event: topo.Event{Object: *topodevice.ToObject(devices[topodevice.ID(device2)])}}, nil)
	stream.EXPECT().Recv().Return(&topo.WatchResponse{Event: topo.Event{Object: *topodevice.ToObject(devices[topodevice.ID(dcDevice)])}}, nil)
	stream.EXPECT().Recv().Return(nil, io.EOF)

	client := mocks.NewMockTopoClient(ctrl)
	client.EXPECT().Watch(gomock.Any(), gomock.Any()).Return(stream, nil).AnyTimes()
	client.EXPECT().Get(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, in *topo.GetRequest, opts ...grpc.CallOption) (*topo.GetResponse, error) {
			return &topo.GetResponse{
				Object: topodevice.ToObject(devices[topodevice.ID(in.ID)]),
			}, nil
		}).AnyTimes()

	deviceStore, err := devicestore.NewStore(client)
	assert.NoError(t, err)

	atomixClient, err := test.NewClient("test")
	assert.NoError(t, err)

	deviceChanges, err := devicechanges.NewAtomixStore(atomixClient)
	assert.NoError(t, err)

	mockTargetDevice(t, device1, ctrl)
	mockTargetDevice(t, device2, ctrl)

	return deviceStore, deviceChanges
}

func mockTargetDevice(t *testing.T, name devicetype.ID, ctrl *gomock.Controller) {
	modelData := gnmi.ModelData{
		Name:         "test1",
		Organization: "Open Networking Foundation",
		Version:      "2018-02-20",
	}
	timeout := time.Millisecond * 200
	mockDevice := topodevice.Device{
		ID:          topodevice.ID(name),
		Address:     "1.2.3.4:11161",
		Version:     v1,
		Timeout:     &timeout,
		Credentials: topodevice.Credentials{},
		TLS:         topodevice.TLSConfig{},
		Type:        "TestDevice",
		Role:        "leaf",
	}

	mockTargetDevice := southboundmock.NewMockTargetIf(ctrl)
	mockTargetDeviceCtx := context.TODO()

	mockTargetDevice.EXPECT().ConnectTarget(
		gomock.Any(),
		mockDevice,
	).Return(devicetype.NewVersionedID(name, v1), nil)

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

	//topoChannel := make(chan *topodevice.ListResponse)
	//dispatcher := dispatcher.NewDispatcher()
	//modelregistry := new(modelregistry.ModelRegistry)
	opStateCache := make(devicechange.TypedValueMap)
	roPathMap := make(modelregistry.ReadOnlyPathMap)
	deviceChangeStore := storemock.NewMockDeviceChangesStore(ctrl)
	deviceChangeStore.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(deviceID devicetype.VersionedID, c chan<- *devicechange.DeviceChange) (stream.Context, error) {
			ctx := stream.NewContext(func() {})
			return ctx, errors.NewNotFound("no Configuration found")
		}).AnyTimes()
	_, err := synchronizer.New(context.Background(), &mockDevice,
		make(chan<- events.OperationalStateEvent), make(chan<- events.DeviceResponse),
		opStateCache, roPathMap, mockTargetDevice,
		configmodel.GetStateExplicitRoPaths, &sync.RWMutex{}, deviceChangeStore)
	assert.NoError(t, err, "Unable to create new synchronizer for", mockDevice.ID)

	// Finally to make it visible to tests - add it to `targets`
	southbound.NewTargetItem(devicetype.NewVersionedID(name, v1), mockTargetDevice)
}

func newChange(index devicechange.Index, device devicetype.ID, version devicetype.Version) *devicechange.DeviceChange {
	return &devicechange.DeviceChange{
		Index: index,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(device),
			Index: 1,
		},
		Change: &devicechange.Change{
			DeviceID:      device,
			DeviceVersion: version,
			DeviceType:    stratumType,
			Values: []*devicechange.ChangeValue{
				{
					Path:  "foo",
					Value: devicechange.NewTypedValueString("Hello world!"),
				},
			},
		},
	}
}

// newChangeInterface creates a new interface eth<n> in the OpenConfig model style
func newChangeInterface(index devicechange.Index, device devicetype.ID, version devicetype.Version, iface int) *devicechange.DeviceChange {
	ifaceID := fmt.Sprintf("%s%d", ethPrefix, iface)
	ifacePath := fmt.Sprintf(ifConfigNameFmt, iface)
	healthValue := healthUp
	if iface%2 == 0 {
		healthValue = healthDown
	}

	return &devicechange.DeviceChange{
		Index: index,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-added", device, iface)),
			Index: types.Index(iface),
		},
		Change: &devicechange.Change{
			DeviceID:      device,
			DeviceVersion: version,
			DeviceType:    stratumType,
			Values: []*devicechange.ChangeValue{
				{
					Path:    ifacePath + ifNameAttr,
					Value:   devicechange.NewTypedValueString(ifaceID),
					Removed: false,
				},
				{
					Path:    ifacePath + ifEnabledAttr,
					Value:   devicechange.NewTypedValueBool(iface%2 == 0),
					Removed: false,
				},
				{
					Path:    ifacePath + ifHealthAttr,
					Value:   devicechange.NewTypedValueString(healthValue),
					Removed: false,
				},
			},
		},
	}
}

// newChangeInterfaceRemove removes an interface eth<n> of the OpenConfig model style
func newChangeInterfaceRemove(index devicechange.Index, device devicetype.ID, version devicetype.Version, iface int) *devicechange.DeviceChange {
	ifacePath := fmt.Sprintf(ifConfigNameFmt, iface)

	return &devicechange.DeviceChange{
		Index: index,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-removed", device, iface)),
			Index: types.Index(iface),
		},
		Change: &devicechange.Change{
			DeviceID:      device,
			DeviceVersion: version,
			DeviceType:    stratumType,
			Values: []*devicechange.ChangeValue{
				{
					Path:    ifacePath,
					Removed: true,
				},
			},
		},
	}
}
