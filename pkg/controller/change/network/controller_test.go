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
	types "github.com/onosproject/onos-api/go/onos/config"
	"github.com/onosproject/onos-api/go/onos/config/change"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	"github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-api/go/onos/topo"
	devicetopo "github.com/onosproject/onos-config/pkg/device"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchanges "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/test/mocks"
	"github.com/stretchr/testify/assert"
	"testing"
)

const (
	device1 = device.ID("device-1")
	device2 = device.ID("device-2")
)

const (
	change1 = networkchange.ID("change-1")
)

// TestReconcilerChangeRollback tests applying and then rolling back a change
func TestReconcilerChangeRollback(t *testing.T) {
	networkChanges, deviceChanges, devices := newStores(t)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		networkChanges: networkChanges,
		deviceChanges:  deviceChanges,
		devices:        devices,
	}

	// Create a network change
	networkChange := newChange(change1, device1, device2)
	err := networkChanges.Create(networkChange)
	assert.NoError(t, err)

	// Reconcile the network change
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

	// The reconciler should have changed its state to RUNNING
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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

	// Verify that device change states were changed to RUNNING
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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

	// Verify that the rollback is running
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_ROLLBACK, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// Reconcile the rollback again
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

	// Verify that device change states were changed to RUNNING
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
	networkChanges, deviceChanges, devices := newStores(t)
	defer networkChanges.Close()
	defer deviceChanges.Close()

	reconciler := &Reconciler{
		networkChanges: networkChanges,
		deviceChanges:  deviceChanges,
		devices:        devices,
	}

	// Create a network change
	networkChange := newChange(change1, device1, device2)
	err := networkChanges.Create(networkChange)
	assert.NoError(t, err)

	// Reconcile the network change
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

	// The reconciler should have changed its state to RUNNING
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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

	// Verify that device change states were changed to RUNNING
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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

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
	err = deviceChanges.Update(deviceChange2)
	assert.NoError(t, err)

	// Reconcile the network change again
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

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
	_, err = reconciler.Reconcile(types.ID(networkChange.ID))
	assert.NoError(t, err)

	// Verify that the network change returned to PENDING
	networkChange, err = networkChanges.Get(change1)
	assert.NoError(t, err)
	assert.Equal(t, change.Phase_CHANGE, networkChange.Status.Phase)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)
}

func newStores(t *testing.T) (networkchanges.Store, devicechanges.Store, devicestore.Store) {
	networkChanges, err := networkchanges.NewLocalStore()
	assert.NoError(t, err)
	deviceChanges, err := devicechanges.NewLocalStore()
	assert.NoError(t, err)
	ctrl := gomock.NewController(t)
	client := mocks.NewMockTopoClient(ctrl)
	client.EXPECT().Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, request *topo.GetRequest) (*topo.GetResponse, error) {
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
	devices, err := devicestore.NewStore(client)
	assert.NoError(t, err)
	return networkChanges, deviceChanges, devices
}

func newChange(id networkchange.ID, devices ...device.ID) *networkchange.NetworkChange {
	changes := make([]*devicechange.Change, len(devices))
	for i, device := range devices {
		changes[i] = &devicechange.Change{
			DeviceID:      device,
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
		}
	}
	return &networkchange.NetworkChange{
		ID:      id,
		Changes: changes,
	}
}
