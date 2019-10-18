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
	log "k8s.io/klog"
	"time"
)

// NewController returns a new network snapshot controller
func NewController(leadership leadershipstore.Store, networkChanges networkchangestore.Store, networkSnapshots networksnapstore.Store, deviceSnapshots devicesnapstore.Store) *controller.Controller {
	c := controller.NewController("NetworkSnapshot")
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
}

// Reconcile reconciles the state of a network configuration
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	snapshot, err := r.networkSnapshots.Get(networksnaptypes.ID(id))
	if err != nil {
		return false, err
	}

	// Handle the snapshot for each phase
	if snapshot != nil {
		switch snapshot.Status.Phase {
		case snaptypes.Phase_MARK:
			return r.reconcileMark(snapshot)
		case snaptypes.Phase_DELETE:
			return r.reconcileDelete(snapshot)
		}
	}
	return true, nil
}

// reconcileMark reconciles a snapshot in the MARK phase
func (r *Reconciler) reconcileMark(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Handle each possible state of the phase
	switch snapshot.Status.State {
	case snaptypes.State_PENDING:
		return r.reconcilePendingMark(snapshot)
	case snaptypes.State_RUNNING:
		return r.reconcileRunningMark(snapshot)
	default:
		return true, nil
	}
}

// reconcilePendingMark reconciles a snapshot in the PENDING state during the MARK phase
func (r *Reconciler) reconcilePendingMark(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Determine whether the snapshot can be applied
	canApply, err := r.canApplySnapshot(snapshot)
	if err != nil {
		return false, err
	} else if !canApply {
		return false, nil
	}

	// If the snapshot can be applied, update the snapshot state to RUNNING
	snapshot.Status.State = snaptypes.State_RUNNING
	log.Infof("Running MARK phase for NetworkSnapshot %v", snapshot)
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return false, err
	}
	return true, nil
}

// canApplySnapshot returns a bool indicating whether the snapshot can be taken
func (r *Reconciler) canApplySnapshot(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	prevSnapshot, err := r.networkSnapshots.GetByIndex(snapshot.Index - 1)
	if err != nil {
		return false, err
	} else if prevSnapshot != nil && (prevSnapshot.Status.Phase != snaptypes.Phase_DELETE || prevSnapshot.Status.State != snaptypes.State_COMPLETE) {
		return false, nil
	}
	return true, nil
}

// hasDeviceSnapshots indicates whether the given snapshot has device snapshots
func hasDeviceSnapshots(snapshot *networksnaptypes.NetworkSnapshot) bool {
	return snapshot.Refs != nil && len(snapshot.Refs) > 0
}

// reconcileRunningMark reconciles a snapshot in the RUNNING state during the MARK phase
func (r *Reconciler) reconcileRunningMark(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// If device snapshots have not been created, run the mark phase
	if !hasDeviceSnapshots(snapshot) {
		return r.createDeviceSnapshots(snapshot)
	}
	return r.completeRunningMark(snapshot)
}

// createDeviceSnapshots marks NetworkChanges for deletion and creates device snapshots
func (r *Reconciler) createDeviceSnapshots(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Iterate through network changes
	deviceChanges := make(map[device.ID]map[string]networkchangetypes.Index)
	deviceMaxChanges := make(map[device.ID]map[string]networkchangetypes.Index)

	// List network changes
	changes := make(chan *networkchangetypes.NetworkChange)
	if err := r.networkChanges.List(changes); err != nil {
		return false, err
	}

	// Compute the maximum timestamp for changes to be deleted from the change store
	var maxTimestamp *time.Time
	if snapshot.Retention.RetainWindow != nil {
		t := time.Now().Add(*snapshot.Retention.RetainWindow * -1)
		maxTimestamp = &t
	}

	// Iterate through network changes in chronological order
	for change := range changes {
		// If the change was created after the retention period, break out of the loop
		if maxTimestamp != nil && change.Created.After(*maxTimestamp) {
			break
		}

		// If the change is still pending, ensure snapshots are not taken of devices following this change
		if change.Status.State == changetypes.State_PENDING || change.Status.State == changetypes.State_RUNNING {
			// Record max device changes if necessary
			for _, device := range change.Refs {
				if _, ok := deviceMaxChanges[device.DeviceChangeID.GetDeviceID()]; !ok {
					deviceVersions, ok := deviceChanges[device.DeviceChangeID.GetDeviceID()]
					if !ok {
						deviceVersions = make(map[string]networkchangetypes.Index)
						deviceChanges[device.DeviceChangeID.GetDeviceID()] = deviceVersions
					}
					prevChangeIndex := deviceVersions[device.DeviceChangeID.GetDeviceVersion()]

					deviceMaxVersions, ok := deviceMaxChanges[device.DeviceChangeID.GetDeviceID()]
					if !ok {
						deviceMaxVersions = make(map[string]networkchangetypes.Index)
						deviceMaxChanges[device.DeviceChangeID.GetDeviceID()] = deviceMaxVersions
					}
					deviceMaxVersions[device.DeviceChangeID.GetDeviceVersion()] = prevChangeIndex
				}
			}
		} else {
			// Mark the change deleted
			change.Deleted = true
			if err := r.networkChanges.Update(change); err != nil {
				return false, err
			}

			// Record the change ID for each device in the change
			for _, device := range change.Refs {
				deviceVersions, ok := deviceChanges[device.DeviceChangeID.GetDeviceID()]
				if !ok {
					deviceVersions = make(map[string]networkchangetypes.Index)
					deviceChanges[device.DeviceChangeID.GetDeviceID()] = deviceVersions
				}
				deviceVersions[device.DeviceChangeID.GetDeviceVersion()] = change.Index
			}
		}
	}

	// Ensure max device changes are populated for all devices
	for device, changeID := range deviceChanges {
		if _, ok := deviceMaxChanges[device]; !ok {
			deviceMaxChanges[device] = changeID
		}
	}

	// Create device snapshots for each device
	refs := make([]*networksnaptypes.DeviceSnapshotRef, 0, len(deviceMaxChanges))
	for deviceID, maxVersions := range deviceMaxChanges {
		for deviceVersion, maxChangeIndex := range maxVersions {
			deviceSnapshot := &devicesnaptypes.DeviceSnapshot{
				DeviceID:      deviceID,
				DeviceVersion: deviceVersion,
				NetworkSnapshot: devicesnaptypes.NetworkSnapshotRef{
					ID:    types.ID(snapshot.ID),
					Index: types.Index(snapshot.Index),
				},
				MaxNetworkChangeIndex: types.Index(maxChangeIndex),
				Status: snaptypes.Status{
					Phase: snaptypes.Phase_MARK,
					State: snaptypes.State_RUNNING,
				},
			}
			if err := r.deviceSnapshots.Create(deviceSnapshot); err != nil {
				return false, err
			}
			refs = append(refs, &networksnaptypes.DeviceSnapshotRef{
				DeviceSnapshotID: deviceSnapshot.ID,
			})
		}
	}

	// Once the device snapshots have been created, update the network snapshot
	snapshot.Refs = refs
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return false, err
	}
	return true, nil
}

// completeRunningMark attempts to complete the MARK phase
func (r *Reconciler) completeRunningMark(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	for _, ref := range snapshot.Refs {
		deviceSnapshot, err := r.deviceSnapshots.Get(ref.DeviceSnapshotID)
		if err != nil {
			return false, err
		} else if deviceSnapshot != nil && deviceSnapshot.Status.State != snaptypes.State_COMPLETE {
			return true, nil
		}
	}

	// If we've made it this far, move to the DELETE phase
	snapshot.Status.Phase = snaptypes.Phase_DELETE
	snapshot.Status.State = snaptypes.State_PENDING
	log.Infof("Completing MARK phase for NetworkSnapshot %v", snapshot)
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return false, err
	}
	return true, nil
}

// reconcileDelete reconciles a snapshot in the DELETE phase
func (r *Reconciler) reconcileDelete(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Handle each possible state of the phase
	switch snapshot.Status.State {
	case snaptypes.State_PENDING:
		return r.reconcilePendingDelete(snapshot)
	case snaptypes.State_RUNNING:
		return r.reconcileRunningDelete(snapshot)
	default:
		return true, nil
	}
}

// reconcilePendingDelete reconciles a snapshot in the PENDING state during the DELETE phase
func (r *Reconciler) reconcilePendingDelete(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Ensure device snapshots are in the DELETE phase
	if ok, err := r.ensureDeviceSnapshotsDelete(snapshot); ok || err != nil {
		return ok, err
	}

	snapshot.Status.State = snaptypes.State_RUNNING
	log.Infof("Running DELETE phase for NetworkSnapshot %v", snapshot)
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return false, err
	}
	return true, nil
}

// ensureDeviceSnapshotsDelete ensures device device snapshots are pending in the DELETE phase
func (r *Reconciler) ensureDeviceSnapshotsDelete(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Ensure all device snapshots are in the DELETE phase
	updated := false
	for _, ref := range snapshot.Refs {
		deviceSnapshot, err := r.deviceSnapshots.Get(ref.DeviceSnapshotID)
		if err != nil {
			return false, err
		}

		if deviceSnapshot.Status.Phase != snaptypes.Phase_DELETE {
			deviceSnapshot.Status.Phase = snaptypes.Phase_DELETE
			deviceSnapshot.Status.State = snaptypes.State_PENDING
			if err := r.deviceSnapshots.Update(deviceSnapshot); err != nil {
				return false, err
			}
			updated = true
		}
	}
	return updated, nil
}

// reconcileRunningDelete reconciles a snapshot in the RUNNING state during the DELETE phase
func (r *Reconciler) reconcileRunningDelete(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// Ensure device snapshots are in the RUNNING state
	if ok, err := r.ensureDeleteRunning(snapshot); ok || err != nil {
		return ok, err
	}

	// Delete all network changes marked for deletion
	if ok, err := r.deleteNetworkChanges(snapshot); !ok || err != nil {
		return ok, err
	}

	// If device snapshots have finished DELETE, complete the snapshot
	complete, err := r.isDeleteComplete(snapshot)
	if err != nil {
		return false, err
	} else if !complete {
		return true, nil
	}

	snapshot.Status.State = snaptypes.State_COMPLETE
	log.Infof("Completing DELETE phase for NetworkSnapshot %v", snapshot)
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return false, nil
	}
	return true, nil
}

// deleteNetworkChanges deletes network changes marked for deletion
func (r *Reconciler) deleteNetworkChanges(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	// List network changes
	changes := make(chan *networkchangetypes.NetworkChange)
	if err := r.networkChanges.List(changes); err != nil {
		return false, err
	}

	// Iterate through network changes and mark changes marked for deletion
	for change := range changes {
		if change.Deleted {
			if err := r.networkChanges.Delete(change); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// ensureDeleteRunning ensures device rollbacks are in the running state
func (r *Reconciler) ensureDeleteRunning(snapshot *networksnaptypes.NetworkSnapshot) (bool, error) {
	updated := false
	for _, ref := range snapshot.Refs {
		deviceSnapshot, err := r.deviceSnapshots.Get(ref.DeviceSnapshotID)
		if err != nil {
			return false, err
		}

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

// isDeleteComplete determines whether device deletes are complete
func (r *Reconciler) isDeleteComplete(networkChange *networksnaptypes.NetworkSnapshot) (bool, error) {
	for _, changeRef := range networkChange.Refs {
		deviceChange, err := r.deviceSnapshots.Get(changeRef.DeviceSnapshotID)
		if err != nil {
			return false, err
		}
		if deviceChange.Status.State != snaptypes.State_COMPLETE {
			return false, nil
		}
	}
	return true, nil
}

var _ controller.Reconciler = &Reconciler{}
