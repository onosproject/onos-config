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
	"fmt"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReconciler_computeNewRollback_singleUpdate(t *testing.T) {
	devices, deviceChangesStore := newStores(t)
	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChangesStore,
	}
	err := setUpComputeDelta(reconciler, deviceChangesStore, 1)
	assert.NoError(t, err)
	defer deviceChangesStore.Close()

	deviceChangeIf1Change := &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-change", device1, 1)),
			Index: types.Index(1),
		},
		Change: &devicechangetypes.Change{
			DeviceID: device1,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:    eth1Enabled,
					Value:   devicechangetypes.NewTypedValueBool(true),
					Removed: false,
				},
			},
		},
	}

	deltas, err := reconciler.computeNewRollback(deviceChangeIf1Change)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(deltas.Values))
	// Was originally false, change set it to true, now that change is being
	// rolled back should be set to false again
	assert.Equal(t, "false", deltas.Values[0].Value.ValueToString())
}

func TestReconciler_computeNewRollback_mixedUpdate(t *testing.T) {
	devices, deviceChangesStore := newStores(t)
	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChangesStore,
	}
	err := setUpComputeDelta(reconciler, deviceChangesStore, 1)
	assert.NoError(t, err)
	defer deviceChangesStore.Close()

	deviceChangeIf1Change := &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-change", device1, 1)),
			Index: types.Index(1),
		},
		Change: &devicechangetypes.Change{
			DeviceID: device1,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:    eth1Enabled,
					Value:   devicechangetypes.NewTypedValueBool(true),
					Removed: false,
				},
				{
					Path:    eth1Hi,
					Removed: true,
				},
				{
					Path:  eth1Desc,
					Value: devicechangetypes.NewTypedValueString("This is a test"),
				},
			},
		},
	}

	deltas, err := reconciler.computeNewRollback(deviceChangeIf1Change)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(deltas.Values))
	// enabled was originally false, change set it to true, should be set to false on rollback
	// health was originally UP, removed it, and should be visible again now as UP
	// description was never in the original so should be removed here
	for _, d := range deltas.Values {
		switch d.Path {
		case eth1Enabled:
			assert.Equal(t, "false", d.Value.ValueToString())
			assert.Equal(t, false, d.Removed)
		case eth1Hi:
			assert.Equal(t, "UP", d.Value.ValueToString())
			assert.Equal(t, false, d.Removed)
		case eth1Desc:
			assert.Nil(t, d.Value)
			assert.Equal(t, true, d.Removed)
		default:
			t.Errorf("Unexpected path %s", d.Path)
		}
	}
}

func TestReconciler_computeNewRollback_addInterface(t *testing.T) {
	devices, deviceChangesStore := newStores(t)
	defer deviceChangesStore.Close()
	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChangesStore,
	}
	err := setUpComputeDelta(reconciler, deviceChangesStore, 1)
	assert.NoError(t, err)

	deviceChangeIf2Add := &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-add", device1, 2)),
			Index: types.Index(1),
		},
		Change: &devicechangetypes.Change{
			DeviceID: device1,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:  eth2Name,
					Value: devicechangetypes.NewTypedValueString(eth2),
				},
				{
					Path:  eth2Enabled,
					Value: devicechangetypes.NewTypedValueBool(true),
				},
				{
					Path:  eth2Hi,
					Value: devicechangetypes.NewTypedValueString(healthUp),
				},
			},
		},
	}

	deltas, err := reconciler.computeNewRollback(deviceChangeIf2Add)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(deltas.Values))
	// Should be deleting everything of eth2
	for _, d := range deltas.Values {
		switch d.Path {
		case eth2Name:
			assert.Nil(t, d.Value)
			assert.Equal(t, true, d.Removed)
		case eth2Enabled:
			assert.Nil(t, d.Value)
			assert.Equal(t, true, d.Removed)
		case eth2Hi:
			assert.Nil(t, d.Value)
			assert.Equal(t, true, d.Removed)
		default:
			t.Errorf("Unexpected path %s", d.Path)
		}
	}
}

func TestReconciler_computeNewRollback_removeInterface(t *testing.T) {
	devices, deviceChangesStore := newStores(t)
	defer deviceChangesStore.Close()
	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChangesStore,
	}
	err := setUpComputeDelta(reconciler, deviceChangesStore, 1)
	assert.NoError(t, err)

	err = setUpComputeDelta(reconciler, deviceChangesStore, 2)
	assert.NoError(t, err)

	deviceChangeIf2Delete := &devicechangetypes.DeviceChange{
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-delete", device1, 2)),
			Index: types.Index(1),
		},
		Change: &devicechangetypes.Change{
			DeviceID: device1,
			Values: []*devicechangetypes.ChangeValue{
				{
					Path:    eth2Base,
					Removed: true,
				},
			},
		},
	}

	deltas, err := reconciler.computeNewRollback(deviceChangeIf2Delete)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(deltas.Values))
	// Should be restoring everything of eth2
	for _, d := range deltas.Values {
		switch d.Path {
		case eth2Name:
			assert.Equal(t, eth2, d.GetValue().ValueToString())
			assert.Equal(t, false, d.Removed)
		case eth2Enabled:
			assert.Equal(t, "true", d.GetValue().ValueToString())
			assert.Equal(t, false, d.Removed)
		case eth2Hi:
			assert.Equal(t, healthDown, d.GetValue().ValueToString())
			assert.Equal(t, false, d.Removed)
		default:
			t.Errorf("Unexpected path %s", d.Path)
		}
	}
}

func setUpComputeDelta(reconciler *Reconciler, deviceChangesStore devicechanges.Store, iface int) error {
	deviceChangeIf := newChangeInterface(device1, iface)

	err := deviceChangesStore.Create(deviceChangeIf)
	if err != nil {
		return err
	}

	ok, err := reconciler.Reconcile(types.ID(deviceChangeIf.ID))
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("failed to reconcile")
	}
	deviceChangeIf.Status.State = change.State_RUNNING
	err = deviceChangesStore.Update(deviceChangeIf)
	if err != nil {
		return err
	}
	ok, err = reconciler.Reconcile(types.ID(deviceChangeIf.ID))
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("failed to reconcile")
	}
	return nil
}
