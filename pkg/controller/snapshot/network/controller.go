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
	"fmt"
	"time"

	"github.com/onosproject/onos-config/api/types"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicebase "github.com/onosproject/onos-config/api/types/device"
	snaptypes "github.com/onosproject/onos-config/api/types/snapshot"
	devicesnapshot "github.com/onosproject/onos-config/api/types/snapshot/device"
	networksnapshot "github.com/onosproject/onos-config/api/types/snapshot/network"
	"github.com/onosproject/onos-config/pkg/controller"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "snapshot", "network")

// NewController returns a new network snapshot controller
func NewController(leadership leadershipstore.Store, networkChanges networkchangestore.Store,
	networkSnapshots networksnapstore.Store, deviceSnapshots devicesnapstore.Store,
	deviceChanges devicechangestore.Store) *controller.Controller {

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
		deviceChanges:    deviceChanges,
		networkSnapshots: networkSnapshots,
		deviceSnapshots:  deviceSnapshots,
	})
	return c
}

// Reconciler is a network snapshot reconciler
type Reconciler struct {
	networkChanges   networkchangestore.Store
	deviceChanges    devicechangestore.Store
	networkSnapshots networksnapstore.Store
	deviceSnapshots  devicesnapstore.Store
}

// Reconcile reconciles the state of a network configuration
func (r *Reconciler) Reconcile(id types.ID) (controller.Result, error) {
	snapshot, err := r.networkSnapshots.Get(networksnapshot.ID(id))
	if err != nil {
		return controller.Result{}, err
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
	return controller.Result{}, nil
}

// reconcileMark reconciles a snapshot in the MARK phase
func (r *Reconciler) reconcileMark(snapshot *networksnapshot.NetworkSnapshot) (controller.Result, error) {
	// Handle each possible state of the phase
	switch snapshot.Status.State {
	case snaptypes.State_PENDING:
		return r.reconcilePendingMark(snapshot)
	case snaptypes.State_RUNNING:
		return r.reconcileRunningMark(snapshot)
	default:
		return controller.Result{}, nil
	}
}

// reconcilePendingMark reconciles a snapshot in the PENDING state during the MARK phase
func (r *Reconciler) reconcilePendingMark(snapshot *networksnapshot.NetworkSnapshot) (controller.Result, error) {
	// Determine whether the snapshot can be applied
	canApply, err := r.canApplySnapshot(snapshot)
	if err != nil {
		return controller.Result{}, err
	} else if !canApply {
		return controller.Result{Requeue: types.ID(snapshot.ID)}, nil
	}

	// If the snapshot can be applied, update the snapshot state to RUNNING
	snapshot.Status.State = snaptypes.State_RUNNING
	log.Infof("Running MARK phase for NetworkSnapshot %v", snapshot)
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// canApplySnapshot returns a bool indicating whether the snapshot can be taken
func (r *Reconciler) canApplySnapshot(snapshot *networksnapshot.NetworkSnapshot) (bool, error) {
	prevSnapshot, err := r.networkSnapshots.GetByIndex(snapshot.Index - 1)
	if err != nil {
		return false, err
	} else if prevSnapshot != nil && (prevSnapshot.Status.Phase != snaptypes.Phase_DELETE || prevSnapshot.Status.State != snaptypes.State_COMPLETE) {
		return false, nil
	}
	return true, nil
}

// hasDeviceSnapshots indicates whether the given snapshot has device snapshots
func hasDeviceSnapshots(snapshot *networksnapshot.NetworkSnapshot) bool {
	return snapshot.Refs != nil && len(snapshot.Refs) > 0
}

// reconcileRunningMark reconciles a snapshot in the RUNNING state during the MARK phase
func (r *Reconciler) reconcileRunningMark(snapshot *networksnapshot.NetworkSnapshot) (controller.Result, error) {
	// If device snapshots have not been created, run the mark phase
	if !hasDeviceSnapshots(snapshot) {
		return r.createDeviceSnapshots(snapshot)
	}
	return r.completeRunningMark(snapshot)
}

// createDeviceSnapshots marks NetworkChanges for deletion and creates device snapshots
func (r *Reconciler) createDeviceSnapshots(snapshot *networksnapshot.NetworkSnapshot) (controller.Result, error) {
	// Iterate through network changes
	deviceChanges := make(map[devicebase.VersionedID]networkchange.Index)
	deviceMaxChanges := make(map[devicebase.VersionedID]networkchange.Index)
	deviceTypes := make(map[devicebase.VersionedID]devicebase.Type)

	// List network changes
	changes := make(chan *networkchange.NetworkChange)
	ctx, err := r.networkChanges.List(changes)
	if err != nil {
		return controller.Result{}, err
	}
	defer ctx.Close()

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
		if change.Status.State == changetypes.State_PENDING {
			// Record max device changes if necessary
			for _, device := range change.Refs {
				if _, ok := deviceMaxChanges[device.DeviceChangeID.GetDeviceVersionedID()]; !ok {
					prevChangeIndex := deviceChanges[device.DeviceChangeID.GetDeviceVersionedID()]
					deviceMaxChanges[device.DeviceChangeID.GetDeviceVersionedID()] = prevChangeIndex
				}
				if _, ok := deviceTypes[device.DeviceChangeID.GetDeviceVersionedID()]; !ok {
					deviceTypes[device.DeviceChangeID.GetDeviceVersionedID()] =
						r.deviceChangeType(change.ID, device.GetDeviceChangeID().GetDeviceID(), device.GetDeviceChangeID().GetDeviceVersion())
				}
			}
		} else {
			// Mark the change deleted
			change.Deleted = true
			if err := r.networkChanges.Update(change); err != nil {
				return controller.Result{}, err
			}

			// Record the change ID for each device in the change
			for _, device := range change.Refs {
				deviceChanges[device.DeviceChangeID.GetDeviceVersionedID()] = change.Index
				if _, ok := deviceTypes[device.DeviceChangeID.GetDeviceVersionedID()]; !ok {
					deviceTypes[device.DeviceChangeID.GetDeviceVersionedID()] =
						r.deviceChangeType(change.ID, device.GetDeviceChangeID().GetDeviceID(), device.GetDeviceChangeID().GetDeviceVersion())
				}
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
	refs := make([]*networksnapshot.DeviceSnapshotRef, 0, len(deviceMaxChanges))
	for device, maxChangeIndex := range deviceMaxChanges {
		deviceSnapshot := &devicesnapshot.DeviceSnapshot{
			DeviceID:      device.GetID(),
			DeviceVersion: device.GetVersion(),
			DeviceType:    deviceTypes[device],
			NetworkSnapshot: devicesnapshot.NetworkSnapshotRef{
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
			return controller.Result{}, err
		}
		refs = append(refs, &networksnapshot.DeviceSnapshotRef{
			DeviceSnapshotID: deviceSnapshot.ID,
		})
	}

	// If there are no device snapshots to take, complete the snapshot successfully.
	if len(refs) == 0 {
		snapshot.Status.Phase = snaptypes.Phase_DELETE
		snapshot.Status.State = snaptypes.State_COMPLETE
	}

	// Once the device snapshots have been created, update the network snapshot
	snapshot.Refs = refs
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// completeRunningMark attempts to complete the MARK phase
func (r *Reconciler) completeRunningMark(snapshot *networksnapshot.NetworkSnapshot) (controller.Result, error) {
	for _, ref := range snapshot.Refs {
		deviceSnapshot, err := r.deviceSnapshots.Get(ref.DeviceSnapshotID)
		if err != nil {
			return controller.Result{}, err
		} else if deviceSnapshot != nil && deviceSnapshot.Status.State != snaptypes.State_COMPLETE {
			return controller.Result{}, nil
		}
	}

	// If we've made it this far, move to the DELETE phase
	snapshot.Status.Phase = snaptypes.Phase_DELETE
	snapshot.Status.State = snaptypes.State_PENDING
	log.Infof("Completing MARK phase for NetworkSnapshot %v", snapshot)
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// reconcileDelete reconciles a snapshot in the DELETE phase
func (r *Reconciler) reconcileDelete(snapshot *networksnapshot.NetworkSnapshot) (controller.Result, error) {
	// Handle each possible state of the phase
	switch snapshot.Status.State {
	case snaptypes.State_PENDING:
		return r.reconcilePendingDelete(snapshot)
	case snaptypes.State_RUNNING:
		return r.reconcileRunningDelete(snapshot)
	default:
		return controller.Result{}, nil
	}
}

// reconcilePendingDelete reconciles a snapshot in the PENDING state during the DELETE phase
func (r *Reconciler) reconcilePendingDelete(snapshot *networksnapshot.NetworkSnapshot) (controller.Result, error) {
	// Ensure device snapshots are in the DELETE phase
	if ok, err := r.ensureDeviceSnapshotsDelete(snapshot); ok || err != nil {
		return controller.Result{}, err
	}

	snapshot.Status.State = snaptypes.State_RUNNING
	log.Infof("Running DELETE phase for NetworkSnapshot %v", snapshot)
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// ensureDeviceSnapshotsDelete ensures device device snapshots are pending in the DELETE phase
func (r *Reconciler) ensureDeviceSnapshotsDelete(snapshot *networksnapshot.NetworkSnapshot) (bool, error) {
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
func (r *Reconciler) reconcileRunningDelete(snapshot *networksnapshot.NetworkSnapshot) (controller.Result, error) {
	// Ensure device snapshots are in the RUNNING state
	if ok, err := r.ensureDeleteRunning(snapshot); ok || err != nil {
		return controller.Result{}, err
	}

	// Delete all network changes marked for deletion
	if ok, err := r.deleteNetworkChanges(snapshot); !ok || err != nil {
		return controller.Result{}, err
	}

	// If device snapshots have finished DELETE, complete the snapshot
	complete, err := r.isDeleteComplete(snapshot)
	if err != nil {
		return controller.Result{}, err
	} else if !complete {
		return controller.Result{}, nil
	}

	snapshot.Status.State = snaptypes.State_COMPLETE
	log.Infof("Completing DELETE phase for NetworkSnapshot %v", snapshot)
	if err := r.networkSnapshots.Update(snapshot); err != nil {
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}

// deleteNetworkChanges deletes network changes marked for deletion
func (r *Reconciler) deleteNetworkChanges(snapshot *networksnapshot.NetworkSnapshot) (bool, error) {
	// List network changes
	changes := make(chan *networkchange.NetworkChange)
	ctx, err := r.networkChanges.List(changes)
	if err != nil {
		return false, err
	}
	defer ctx.Close()

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
func (r *Reconciler) ensureDeleteRunning(snapshot *networksnapshot.NetworkSnapshot) (bool, error) {
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
func (r *Reconciler) isDeleteComplete(networkChange *networksnapshot.NetworkSnapshot) (bool, error) {
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

func (r *Reconciler) deviceChangeType(nwChangeID networkchange.ID, deviceID devicebase.ID, deviceVersion devicebase.Version) devicebase.Type {
	dcID := devicechange.ID(fmt.Sprintf("%s:%s:%s", nwChangeID, deviceID, deviceVersion))
	dc, err := r.deviceChanges.Get(dcID)
	if err != nil || dc.GetChange() == nil {
		log.Warnf("Unable to get Device change for %s", dcID)
		return ""
	}
	return dc.GetChange().GetDeviceType()
}

var _ controller.Reconciler = &Reconciler{}
