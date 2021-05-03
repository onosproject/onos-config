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
	"github.com/golang/mock/gomock"
	types "github.com/onosproject/onos-api/go/onos/config/change"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	devicechangecontroller "github.com/onosproject/onos-config/pkg/controller/change/device"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/southbound"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	southboundtest "github.com/onosproject/onos-config/pkg/test/mocks/southbound"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
	"time"
)

func Test_NewController2Devices(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	networkChanges, deviceChanges, devices := newStores(t, ctrl)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	deviceCache := newDeviceCache(ctrl)
	defer deviceCache.Close()

	leadershipStore, err := leadershipstore.NewLocalStore("cluster-1", "node-1")
	assert.NoError(t, err)
	defer leadershipStore.Close()

	mastershipStore, err := mastershipstore.NewLocalStore("cluster-1", "node-1")
	assert.NoError(t, err)
	defer mastershipStore.Close()

	networkChangeController := NewController(leadershipStore, deviceCache, devices, networkChanges, deviceChanges)
	assert.NotNil(t, networkChangeController)

	deviceChangeController := devicechangecontroller.NewController(mastershipStore, devices, deviceCache, deviceChanges)
	assert.NotNil(t, deviceChangeController)

	mockTargetDevice1 := southboundtest.NewMockTargetIf(ctrl)
	assert.NotNil(t, mockTargetDevice1)
	southbound.Targets[topodevice.ID(device1)] = mockTargetDevice1
	device1Context, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	mockTargetDevice1.EXPECT().Context().Return(&device1Context).AnyTimes()
	mockTargetDevice1.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

	mockTargetDevice2 := southboundtest.NewMockTargetIf(ctrl)
	assert.NotNil(t, mockTargetDevice2)
	southbound.Targets[topodevice.ID(device2)] = mockTargetDevice2
	device2Context, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	mockTargetDevice2.EXPECT().Context().Return(&device2Context).AnyTimes()
	mockTargetDevice2.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

	err = networkChangeController.Start()
	assert.NoError(t, err)
	defer networkChangeController.Stop()

	err = deviceChangeController.Start()
	assert.NoError(t, err)
	defer deviceChangeController.Stop()

	// Create a network change
	networkChange1 := &networkchange.NetworkChange{
		ID: "change-1",
		Changes: []*devicechange.Change{
			&deviceChange1,
			&deviceChange2,
		},
	}

	err = networkChanges.Create(networkChange1)
	assert.NoError(t, err)

	// Should cause an event to be sent to the Watcher
	// Watcher should pass it to the Reconciler (if not filtered)
	// Reconciler should process it
	// Verify that device changes were created
	time.Sleep(300 * time.Millisecond)
	networkChange1, err = networkChanges.Get(networkChange1.GetID())
	assert.NoError(t, err)
	assert.Equal(t, types.Phase_CHANGE, networkChange1.Status.Phase)
	assert.Equal(t, types.State_COMPLETE, networkChange1.Status.State)
	assert.Equal(t, uint64(1), networkChange1.Status.Incarnation)

	deviceChange1, err := deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange1)
	assert.Equal(t, types.Phase_CHANGE, deviceChange1.Status.Phase)
	assert.Equal(t, types.State_COMPLETE, deviceChange1.Status.State)
	assert.Equal(t, types.Reason_NONE, deviceChange1.Status.Reason)
	assert.Equal(t, "", deviceChange1.Status.Message)
	assert.Equal(t, uint64(1), deviceChange1.Status.Incarnation)
	deviceChange2, err := deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange2)
	assert.Equal(t, types.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, types.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, types.Reason_NONE, deviceChange2.Status.Reason)
	assert.Equal(t, "", deviceChange2.Status.Message)
	assert.Equal(t, uint64(1), deviceChange2.Status.Incarnation)
}

// a NetworkChange is made to 2 devices. One of the devices returns an error on Set
// and so the NetworkChange is rolled back, rolling back the devicechange on both devices,
// in the end leaving both devices unchanged.
// The Network and Device changes sit there in COMPLETED state in the ROLLBACK phase.
func Test_NewController1FailsGnmiSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkChanges, deviceChanges, devices := newStores(t, ctrl)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	deviceCache := newDeviceCache(ctrl)
	defer deviceCache.Close()

	leadershipStore, err := leadershipstore.NewLocalStore("cluster-1", "node-1")
	assert.NoError(t, err)
	defer leadershipStore.Close()

	mastershipStore, err := mastershipstore.NewLocalStore("cluster-1", "node-1")
	assert.NoError(t, err)
	defer mastershipStore.Close()

	networkChangeController := NewController(leadershipStore, deviceCache, devices, networkChanges, deviceChanges)
	assert.NotNil(t, networkChangeController)

	deviceChangeController := devicechangecontroller.NewController(mastershipStore, devices, deviceCache, deviceChanges)
	assert.NotNil(t, deviceChangeController)

	mockTargetDevice1 := southboundtest.NewMockTargetIf(ctrl)
	assert.NotNil(t, mockTargetDevice1)
	southbound.Targets[topodevice.ID(device1)] = mockTargetDevice1
	device1Context, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	mockTargetDevice1.EXPECT().Context().Return(&device1Context).AnyTimes()
	mockTargetDevice1.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)

	mockTargetDevice2 := southboundtest.NewMockTargetIf(ctrl)
	assert.NotNil(t, mockTargetDevice2)
	southbound.Targets[topodevice.ID(device2)] = mockTargetDevice2
	device2Context, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	mockTargetDevice2.EXPECT().Context().Return(&device2Context).AnyTimes()
	// First time will return an error
	mockTargetDevice2.EXPECT().Set(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, request *gnmi.SetRequest) (*gnmi.SetResponse, error) {
			return nil, status.Errorf(codes.Internal, "simulated error in device-2 %s", request)
		}).Times(1)
	// Second time will be a rollback when SET is not possible - no error
	mockTargetDevice2.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

	err = networkChangeController.Start()
	assert.NoError(t, err)
	defer networkChangeController.Stop()

	err = deviceChangeController.Start()
	assert.NoError(t, err)
	defer deviceChangeController.Stop()

	// Create a network change
	networkChange1 := &networkchange.NetworkChange{
		ID: "change-1",
		Changes: []*devicechange.Change{
			&deviceChange1,
			&deviceChange2,
		},
	}

	err = networkChanges.Create(networkChange1)
	assert.NoError(t, err)

	// Should cause an event to be sent to the Watcher
	// Watcher should pass it to the Reconciler (if not filtered)
	// Reconciler should process it
	// Verify that device changes were created
	time.Sleep(500 * time.Millisecond) // Should give 5 attempts 20+40+80+160+320 ms
	networkChange1, err = networkChanges.Get(networkChange1.GetID())
	assert.NoError(t, err)
	assert.Equal(t, types.Phase_CHANGE, networkChange1.Status.Phase)
	assert.Equal(t, types.State_PENDING, networkChange1.Status.State)
	assert.Equal(t, types.Reason_ERROR, networkChange1.Status.Reason)
	assert.Equal(t, "change rejected by device", networkChange1.Status.Message)
	assert.Equal(t, uint64(1), networkChange1.Status.Incarnation)

	deviceChange1, err := deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange1)
	assert.Equal(t, types.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, types.State_COMPLETE, deviceChange1.Status.State)
	assert.Equal(t, types.Reason_NONE, deviceChange1.Status.Reason)
	assert.Equal(t, "", deviceChange1.Status.Message)
	assert.Equal(t, uint64(1), deviceChange1.Status.Incarnation)
	deviceChange2, err := deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange2)
	assert.Equal(t, types.Phase_ROLLBACK, deviceChange2.Status.Phase)
	assert.Equal(t, types.State_COMPLETE, deviceChange2.Status.State)
	assert.Equal(t, types.Reason_ERROR, deviceChange2.Status.Reason)
	assert.Equal(t, `rpc error: code = Internal desc = simulated error in device-2 update:{path:{elem:{name:"baz"}} val:{string_val:"Goodbye world!"}}`,
		strings.ReplaceAll(deviceChange2.Status.Message, "  ", " "))
	assert.Equal(t, uint64(1), deviceChange2.Status.Incarnation)
}

// a NetworkChange is made to 2 devices, which succeeds
// Then rollback the network change
// The Network and Device changes sit there in COMPLETED state in the ROLLBACK phase.
func Test_NewControllerDoRollbackWhichFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	networkChanges, deviceChanges, devices := newStores(t, ctrl)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	deviceCache := newDeviceCache(ctrl)
	defer deviceCache.Close()

	leadershipStore, err := leadershipstore.NewLocalStore("cluster-1", "node-1")
	assert.NoError(t, err)
	defer leadershipStore.Close()

	mastershipStore, err := mastershipstore.NewLocalStore("cluster-1", "node-1")
	assert.NoError(t, err)
	defer mastershipStore.Close()

	networkChangeController := NewController(leadershipStore, deviceCache, devices, networkChanges, deviceChanges)
	assert.NotNil(t, networkChangeController)

	deviceChangeController := devicechangecontroller.NewController(mastershipStore, devices, deviceCache, deviceChanges)
	assert.NotNil(t, deviceChangeController)

	mockTargetDevice1 := southboundtest.NewMockTargetIf(ctrl)
	assert.NotNil(t, mockTargetDevice1)
	southbound.Targets[topodevice.ID(device1)] = mockTargetDevice1
	device1Context, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	mockTargetDevice1.EXPECT().Context().Return(&device1Context).AnyTimes()
	mockTargetDevice1.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil, nil).Times(2)

	mockTargetDevice2 := southboundtest.NewMockTargetIf(ctrl)
	assert.NotNil(t, mockTargetDevice2)
	southbound.Targets[topodevice.ID(device2)] = mockTargetDevice2
	device2Context, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	mockTargetDevice2.EXPECT().Context().Return(&device2Context).AnyTimes()
	// First time is the SET - no error
	mockTargetDevice2.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
	// Second time will be a rollback but SET returns error
	mockTargetDevice2.EXPECT().Set(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, request *gnmi.SetRequest) (*gnmi.SetResponse, error) {
			return nil, status.Errorf(codes.Internal, "simulated error on rollback in device-2 %s", request)
		}).Times(1)
	// Third time will be a roll forward but here too SET returns error
	//mockTargetDevice2.EXPECT().Set(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)

	mockTargetDevice2.EXPECT().Set(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, request *gnmi.SetRequest) (*gnmi.SetResponse, error) {
			return nil, status.Errorf(codes.Internal, "simulated error on undoing rollback in device-2 %s", request)
		}).Times(1)

	err = networkChangeController.Start()
	assert.NoError(t, err)
	defer networkChangeController.Stop()

	err = deviceChangeController.Start()
	assert.NoError(t, err)
	defer deviceChangeController.Stop()

	// Create a network change
	networkChange1 := &networkchange.NetworkChange{
		ID: "change-1",
		Changes: []*devicechange.Change{
			&deviceChange1,
			&deviceChange2,
		},
	}
	err = networkChanges.Create(networkChange1)
	assert.NoError(t, err)

	// Should cause an event to be sent to the Watcher
	// Watcher should pass it to the Reconciler (if not filtered)
	// Reconciler should process it
	// Verify that device changes were created
	time.Sleep(50 * time.Millisecond)

	// Now try a rollback
	changeRollback, errGet := networkChanges.Get(change1)
	assert.NoError(t, errGet)
	assert.NotNil(t, changeRollback)
	changeRollback.Status.Incarnation++
	changeRollback.Status.Phase = types.Phase_ROLLBACK
	changeRollback.Status.State = types.State_PENDING
	changeRollback.Status.Reason = types.Reason_NONE
	changeRollback.Status.Message = "Administratively requested rollback"
	err = networkChanges.Update(changeRollback)
	assert.NoError(t, err)
	time.Sleep(500 * time.Millisecond) // Should give 5 attempts 20+40+80+160+320 ms

	networkChange1, err = networkChanges.Get(networkChange1.GetID())
	assert.NoError(t, err)
	assert.Equal(t, types.Phase_ROLLBACK, networkChange1.Status.Phase)
	assert.Equal(t, types.State_PENDING, networkChange1.Status.State)
	assert.Equal(t, types.Reason_ERROR, networkChange1.Status.Reason)
	assert.Equal(t, "rollback rejected by device", networkChange1.Status.Message)
	assert.Equal(t, uint64(2), networkChange1.Status.Incarnation)

	deviceChange1, err := deviceChanges.Get("change-1:device-1:1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange1)
	assert.Equal(t, types.Phase_ROLLBACK, deviceChange1.Status.Phase)
	assert.Equal(t, types.State_COMPLETE, deviceChange1.Status.State)
	assert.Equal(t, types.Reason_NONE, deviceChange1.Status.Reason)
	assert.Equal(t, "", deviceChange1.Status.Message)
	assert.Equal(t, uint64(2), deviceChange1.Status.Incarnation)
	deviceChange2, err := deviceChanges.Get("change-1:device-2:1.0.0")
	assert.NoError(t, err)
	assert.NotNil(t, deviceChange2)
	assert.Equal(t, types.Phase_CHANGE, deviceChange2.Status.Phase)
	assert.Equal(t, types.State_FAILED, deviceChange2.Status.State)
	assert.Equal(t, types.Reason_ERROR, deviceChange2.Status.Reason)
	assert.Equal(t, `rpc error: code = Internal desc = simulated error on undoing rollback in device-2 update:{path:{elem:{name:"baz"}} val:{string_val:"Goodbye world!"}}`,
		strings.ReplaceAll(deviceChange2.Status.Message, "  ", " "))
	assert.Equal(t, uint64(2), deviceChange2.Status.Incarnation)
}
