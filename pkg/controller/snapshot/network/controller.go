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
	"github.com/onosproject/onos-config/pkg/controller"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	"github.com/onosproject/onos-config/pkg/types"
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	snaptypes "github.com/onosproject/onos-config/pkg/types/snapshot"
	devicesnaptypes "github.com/onosproject/onos-config/pkg/types/snapshot/device"
	networksnaptypes "github.com/onosproject/onos-config/pkg/types/snapshot/network"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
)

// NewController returns a new network snapshot controller
func NewController(leadership leadershipstore.Store, networkChanges networkchangestore.Store, networkSnapshots networksnapstore.Store, deviceSnapshots devicesnapstore.Store) *controller.Controller {
	c := controller.NewController()
	c.Activate(&controller.LeadershipActivator{
		Store: leadership,
	})
	c.Watch(&Watcher{
		Store: networkSnapshots,
	})
	c.Watch(&DeviceWatcher{
		Store: deviceSnapshots,
	})
	c.Reconcile(&Reconciler{
		networkChanges:   networkChanges,
		networkSnapshots: networkSnapshots,
		deviceSnapshots:  deviceSnapshots,
	})
	return c
}

// Reconciler is a network snapshot reconciler
type Reconciler struct {
	networkChanges   networkchangestore.Store
	networkSnapshots networksnapstore.Store
	deviceSnapshots  devicesnapstore.Store
	snapshotIndex    networksnaptypes.Index
}

// Reconcile reconciles the state of a network snapshot request
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	snapshot, err := r.networkSnapshots.Get(networksnaptypes.ID(id))
	if err != nil {
		return false, err
	}

	// If the snapshot is nil, skip it
	if snapshot == nil {
		return true, nil
	}

	// Ensure device snapshots exist for the network snapshot
	if ok, err := r.ensureDeviceSnapshots(snapshot); ok || err != nil {
		return ok, err
	}

	// Handle the snapshot based on its state
	switch snapshot.Status.State {
	case snaptypes.State_PENDING:
		return r.reconcilePendingSnapshot(snapshot)
	case snaptypes.State_RUNNING:
		return r.reconcileRunningSnapshot(snapshot)
	default:
		return true, nil
	}
}

// ensureDeviceSnapshots ensures device snapshots have been created for all devices in the network snapshot
func (r *Reconciler) ensureDeviceSnapshots(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// If the snapshot does not specify devices to snapshot, determine a set of devices to snapshot
	if snapshot.Devices == nil || len(snapshot.Devices) == 0 {
		devices, err := r.getDevices(snapshot)
		if err != nil {
			return false, err
		}

		deviceRefs := make([]*networksnaptypes.DeviceSnapshotRef, len(devices))
		for i, deviceID := range devices {
			deviceRefs[i] = &networksnaptypes.DeviceSnapshotRef{
				DeviceID: deviceID,
			}
		}

		snapshot.Devices = deviceRefs
		if err := r.networkSnapshots.Update(snapshot); err != nil {
			return false, err
		}
		return true, nil
	}

	// Loop through devices and ensure device snapshots have been created
	updated := false
	for _, deviceRef := range snapshot.Devices {
		if deviceRef.DeviceSnapshotId == "" {
			deviceSnapshot := &devicesnaptypes.DeviceSnapshot{
				DeviceID:          deviceRef.DeviceID,
				NetworkSnapshotID: types.ID(snapshot.ID),
				Timestamp:         snapshot.Timestamp,
			}
			if err := r.deviceSnapshots.Create(deviceSnapshot); err != nil {
				return false, err
			}
			deviceRef.DeviceSnapshotId = deviceSnapshot.ID
			updated = true
		}
	}

	// If device snapshots have been updated, store the indexes first in the network snapshot
	if updated {
		if err := r.networkSnapshots.Update(snapshot); err != nil {
			return false, err
		}
	}
	return updated, nil
}

// getDevices gets the set of devices that can be snapshot for the given snapshot request
func (r *Reconciler) getDevices(snapshot *networksnaptypes.NetworkSnapshot) ([]device.ID, error) {
	lastIndex, err := r.networkChanges.LastIndex()
	if err != nil {
		return nil, err
	}

	devices := make(map[device.ID]bool)
	for index := networkchangetypes.Index(0); index <= lastIndex; index++ {
		networkChange, err := r.networkChanges.GetByIndex(index)
		if err != nil {
			return nil, err
		} else if networkChange != nil {
			if networkChange.Created.After(snapshot.Timestamp) {
				break
			}

			if networkChange.Status.Phase == changetypes.Phase_CHANGE {
				for _, deviceChange := range networkChange.Changes {
					devices[deviceChange.DeviceID] = true
				}
			}
		}
	}

	deviceIDs := make([]device.ID, 0, len(devices))
	for deviceID := range devices {
		deviceIDs = append(deviceIDs, deviceID)
	}
	return deviceIDs, nil
}

// reconcilePendingSnapshot reconciles a snapshot in the PENDING state
func (r *Reconciler) reconcilePendingSnapshot(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Determine whether the snapshot can be applied
	canApply, err := r.canApplySnapshot(snapshot)
	if err != nil {
		return false, err
	} else if !canApply {
		return false, nil
	}

	// If the snapshot can be applied, update the snapshot state to RUNNING
	snapshot.Status.State = snaptypes.State_RUNNING
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return false, err
	}
	return true, nil
}

// canApplySnapshot returns a bool indicating whether the snapshot can be applied
func (r *Reconciler) canApplySnapshot(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	sequential := true
	for index := r.snapshotIndex; index < snapshot.Index; index++ {
		priorSnapshot, err := r.networkSnapshots.GetByIndex(index)
		if err != nil {
			return false, err
		} else if priorSnapshot != nil {
			if priorSnapshot.Status.State == snaptypes.State_PENDING || priorSnapshot.Status.State == snaptypes.State_RUNNING {
				if isIntersectingSnapshot(snapshot, priorSnapshot) {
					return false, nil
				}
				sequential = false
			} else {
				if sequential {
					r.snapshotIndex++
				}
			}
		} else {
			if sequential {
				r.snapshotIndex++
			}
		}
	}
	return true, nil
}

// reconcileRunningSnapshot reconciles a snapshot in the RUNNING state
func (r *Reconciler) reconcileRunningSnapshot(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Get the current state of all device snapshots for the snapshot
	deviceSnapshots, err := r.getDeviceSnapshots(snapshot)
	if err != nil {
		return false, err
	}

	// Ensure the device snapshots are being applied
	succeeded, err := r.ensureDeviceSnapshotsRunning(deviceSnapshots)
	if succeeded || err != nil {
		return succeeded, err
	}

	// If all device snapshots are complete, mark the network snapshot complete
	if r.isDeviceSnapshotsComplete(deviceSnapshots) {
		snapshot.Status.State = snaptypes.State_COMPLETE
		if err := r.networkSnapshots.Update(snapshot); err != nil {
			return false, err
		}
		return true, nil
	}
	return true, nil
}

// ensureDeviceSnapshotsRunning ensures device snapshots are in the running state
func (r *Reconciler) ensureDeviceSnapshotsRunning(deviceSnapshots []*devicesnaptypes.DeviceSnapshot) (bool, error) {
	// Ensure all device snapshots are being applied
	updated := false
	for _, deviceSnapshot := range deviceSnapshots {
		if deviceSnapshot.Status.State == snaptypes.State_PENDING {
			deviceSnapshot.Status.State = snaptypes.State_RUNNING
			if err := r.deviceSnapshots.Update(deviceSnapshot); err != nil {
				return false, err
			}
			updated = true
		}
	}
	return updated, nil
}

// getDeviceSnapshots gets the device snapshots for the given network snapshot
func (r *Reconciler) getDeviceSnapshots(snapshot *networksnaptypes.NetworkSnapshot) ([]*devicesnaptypes.DeviceSnapshot, error) {
	deviceSnapshots := make([]*devicesnaptypes.DeviceSnapshot, len(snapshot.Devices))
	for i, deviceRef := range snapshot.Devices {
		deviceSnapshot, err := r.deviceSnapshots.Get(deviceRef.DeviceSnapshotId)
		if err != nil {
			return nil, err
		}
		deviceSnapshots[i] = deviceSnapshot
	}
	return deviceSnapshots, nil
}

// isDeviceSnapshotsComplete checks whether the device snapshots are complete
func (r *Reconciler) isDeviceSnapshotsComplete(snapshots []*devicesnaptypes.DeviceSnapshot) bool {
	for _, snapshot := range snapshots {
		if snapshot.Status.State != snaptypes.State_COMPLETE {
			return false
		}
	}
	return true
}

// isIntersectingSnapshot indicates whether the devices from the two given NetworkSnapshots intersect
func isIntersectingSnapshot(config *networksnaptypes.NetworkSnapshot, history *networksnaptypes.NetworkSnapshot) bool {
	for _, currentSnapshot := range config.Devices {
		for _, priorSnapshot := range history.Devices {
			if currentSnapshot.DeviceID == priorSnapshot.DeviceID {
				return true
			}
		}
	}
	return false
}

var _ controller.Reconciler = &Reconciler{}
