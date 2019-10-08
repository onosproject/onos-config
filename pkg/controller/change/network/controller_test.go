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
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchanges "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechange "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchange "github.com/onosproject/onos-config/pkg/types/change/network"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestNetworkControllerSuccess verifies that the network controller applies device changes
func TestNetworkControllerSuccess(t *testing.T) {
	leadershipStore, err := leadership.NewLocalStore("TestNetworkController", cluster.NodeID("node-1"))
	assert.NoError(t, err)
	networkChanges, err := networkchanges.NewLocalStore()
	assert.NoError(t, err)
	deviceChanges, err := devicechanges.NewLocalStore()
	assert.NoError(t, err)

	// Create and start the controller
	controller := NewController(leadershipStore, networkChanges, deviceChanges)
	err = controller.Start()
	assert.NoError(t, err)

	// Create watches on device and network changes
	deviceCh := make(chan *devicechange.Change)
	err = deviceChanges.Watch(deviceCh)
	assert.NoError(t, err)

	networkCh := make(chan *networkchange.NetworkChange)
	err = networkChanges.Watch(networkCh)
	assert.NoError(t, err)

	device1 := device.ID("device-1")
	device2 := device.ID("device-2")

	// Create a network change
	change1 := &networkchange.NetworkChange{
		Changes: []*devicechange.Change{
			{

				DeviceID: device1,
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
				DeviceID: device2,
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

	err = networkChanges.Create(change1)
	assert.NoError(t, err)

	// Verify the change is propagated to devices
	deviceChange := nextDeviceEvent(t, deviceCh)
	assert.Equal(t, devicechange.ID("device-1:1"), deviceChange.ID)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	deviceChange = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, devicechange.ID("device-2:1"), deviceChange.ID)
	assert.Equal(t, types.ID(change1.ID), deviceChange.NetworkChangeID)
	assert.Equal(t, change.State_PENDING, deviceChange.Status.State)

	// After the device change have been created, the controller should apply
	// the network change since no changes on the devices are in progress
	networkChange := nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	// The controller will also update change information in the parent
	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_PENDING, networkChange.Status.State)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Applying the network change should result in the device changes being applied
	deviceChange = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, devicechange.ID("device-1:1"), deviceChange.ID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	deviceChange = nextDeviceEvent(t, deviceCh)
	assert.Equal(t, devicechange.ID("device-2:1"), deviceChange.ID)
	assert.Equal(t, change.State_APPLYING, deviceChange.Status.State)

	// Marking one device change successful should not result in the network change update
	deviceChange, err = deviceChanges.Get("device-2:1")
	assert.NoError(t, err)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	// Check that the network change state has not changed
	networkChange, err = networkChanges.Get("network:1")
	assert.NoError(t, err)
	assert.Equal(t, change.State_APPLYING, networkChange.Status.State)

	// Marking the other device change successful should result in the network change being successful
	deviceChange, err = deviceChanges.Get("device-1:1")
	assert.NoError(t, err)
	deviceChange.Status.State = change.State_SUCCEEDED
	err = deviceChanges.Update(deviceChange)
	assert.NoError(t, err)

	networkChange = nextNetworkEvent(t, networkCh)
	assert.Equal(t, change1.ID, networkChange.ID)
	assert.Equal(t, change.State_SUCCEEDED, networkChange.Status.State)

	networkChange, err = networkChanges.Get("network:1")
	assert.NoError(t, err)
	assert.Equal(t, change.State_SUCCEEDED, networkChange.Status.State)
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

func nextNetworkEvent(t *testing.T, ch chan *networkchange.NetworkChange) *networkchange.NetworkChange {
	select {
	case e := <-ch:
		return e
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	return nil
}
