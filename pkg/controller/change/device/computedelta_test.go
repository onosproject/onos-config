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
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	types "github.com/onosproject/onos-api/go/onos/config"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	devicechanges "github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReconciler_computeRollback_singleUpdate(t *testing.T) {
	t.Skip()
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	assert.NoError(t, test.Start())
	defer test.Stop()

	devices, deviceChangesStore := NewStores(t, test)
	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChangesStore,
	}
	err := setUpComputeDelta(reconciler, deviceChangesStore, 1, 1)
	assert.NoError(t, err)
	defer deviceChangesStore.Close()

	deviceChangeIf1Change := &devicechange.DeviceChange{
		Index: 1,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-change", device1, 1)),
			Index: types.Index(1),
		},
		Change: &devicechange.Change{
			DeviceID:      device1,
			DeviceVersion: v1,
			DeviceType:    stratumType,
			Values: []*devicechange.ChangeValue{
				{
					Path:    eth1Enabled,
					Value:   devicechange.NewTypedValueBool(true),
					Removed: false,
				},
			},
		},
	}

	deltas, err := reconciler.computeRollback(deviceChangeIf1Change)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(deltas.Values))
	// Was originally false, change set it to true, now that change is being
	// rolled back should be set to false again
	assert.NotNil(t, deltas.Values[0].Value)
	assert.Equal(t, "false", deltas.Values[0].Value.ValueToString())
}

func TestReconciler_computeRollback_mixedUpdate(t *testing.T) {
	t.Skip()
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	assert.NoError(t, test.Start())
	defer test.Stop()

	devices, deviceChangesStore := NewStores(t, test)
	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChangesStore,
	}
	err := setUpComputeDelta(reconciler, deviceChangesStore, 1, 1)
	assert.NoError(t, err)
	defer deviceChangesStore.Close()

	deviceChangeIf1Change := &devicechange.DeviceChange{
		Index: 1,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-change", device1, 1)),
			Index: types.Index(1),
		},
		Change: &devicechange.Change{
			DeviceID:      device1,
			DeviceVersion: v1,
			DeviceType:    stratumType,
			Values: []*devicechange.ChangeValue{
				{
					Path:    eth1Enabled,
					Value:   devicechange.NewTypedValueBool(true),
					Removed: false,
				},
				{
					Path:    eth1Hi,
					Removed: true,
				},
				{
					Path:  eth1Desc,
					Value: devicechange.NewTypedValueString("This is a test"),
				},
			},
		},
	}

	deltas, err := reconciler.computeRollback(deviceChangeIf1Change)
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

func TestReconciler_computeRollback_addInterface(t *testing.T) {
	t.Skip()
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	assert.NoError(t, test.Start())
	defer test.Stop()

	devices, deviceChangesStore := NewStores(t, test)
	defer deviceChangesStore.Close()
	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChangesStore,
	}
	err := setUpComputeDelta(reconciler, deviceChangesStore, 1, 1)
	assert.NoError(t, err)

	deviceChangeIf2Add := &devicechange.DeviceChange{
		Index: 1,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-add", device1, 2)),
			Index: types.Index(1),
		},
		Change: &devicechange.Change{
			DeviceID:      device1,
			DeviceVersion: v1,
			DeviceType:    stratumType,
			Values: []*devicechange.ChangeValue{
				{
					Path:  eth2Name,
					Value: devicechange.NewTypedValueString(eth2),
				},
				{
					Path:  eth2Enabled,
					Value: devicechange.NewTypedValueBool(true),
				},
				{
					Path:  eth2Hi,
					Value: devicechange.NewTypedValueString(healthUp),
				},
			},
		},
	}

	deltas, err := reconciler.computeRollback(deviceChangeIf2Add)
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

func TestReconciler_computeRollback_removeInterface(t *testing.T) {
	t.Skip()
	test := test.NewTest(
		rsm.NewProtocol(),
		test.WithReplicas(1),
		test.WithPartitions(1),
	)
	assert.NoError(t, test.Start())
	defer test.Stop()

	devices, deviceChangesStore := NewStores(t, test)
	defer deviceChangesStore.Close()
	reconciler := &Reconciler{
		devices: devices,
		changes: deviceChangesStore,
	}
	err := setUpComputeDelta(reconciler, deviceChangesStore, 1, 1)
	assert.NoError(t, err)

	err = setUpComputeDelta(reconciler, deviceChangesStore, 2, 2)
	assert.NoError(t, err)

	deviceChangeIf2Delete := &devicechange.DeviceChange{
		Index: 1,
		NetworkChange: devicechange.NetworkChangeRef{
			ID:    types.ID(fmt.Sprintf("%s-if%d-delete", device1, 2)),
			Index: types.Index(1),
		},
		Change: &devicechange.Change{
			DeviceID:      device1,
			DeviceVersion: v1,
			DeviceType:    stratumType,
			Values: []*devicechange.ChangeValue{
				{
					Path:    eth2Base,
					Removed: true,
				},
			},
		},
	}

	deltas, err := reconciler.computeRollback(deviceChangeIf2Delete)
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

func setUpComputeDelta(reconciler *Reconciler, deviceChangesStore devicechanges.Store, index devicechange.Index, iface int) error {
	deviceChangeIf := newChangeInterface(index, device1, v1, iface)
	err := deviceChangesStore.Create(deviceChangeIf)
	if err != nil {
		return err
	}

	_, err = reconciler.Reconcile(controller.NewID(string(deviceChangeIf.ID)))
	if err != nil {
		return err
	}
	deviceChangeIf.Status.Incarnation++
	err = deviceChangesStore.Update(deviceChangeIf)
	if err != nil {
		return err
	}
	_, err = reconciler.Reconcile(controller.NewID(string(deviceChangeIf.ID)))
	if err != nil {
		return err
	}
	return nil
}
