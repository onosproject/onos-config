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
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/types"
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	devicetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	networktypes "github.com/onosproject/onos-config/pkg/types/change/network"
)

// NewController returns a new config controller
func NewController(leadership leadershipstore.Store, deviceStore devicestore.Store, networkChanges networkchangestore.Store, deviceChanges devicechangestore.Store) *controller.Controller {
	c := controller.NewController()
	c.Activate(&controller.LeadershipActivator{
		Store: leadership,
	})
	c.Watch(&Watcher{
		Store: networkChanges,
	})
	c.Watch(&DeviceWatcher{
		DeviceStore: deviceStore,
		ChangeStore: deviceChanges,
	})
	c.Reconcile(&Reconciler{
		networkChanges: networkChanges,
		deviceChanges:  deviceChanges,
	})
	return c
}

// Reconciler is a config reconciler
type Reconciler struct {
	networkChanges networkchangestore.Store
	deviceChanges  devicechangestore.Store
	// changeIndex is the index of the highest sequential network change applied
	changeIndex networktypes.Index
}

// Reconcile reconciles the state of a network configuration
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	change, err := r.networkChanges.Get(networktypes.ID(id))
	if err != nil {
		return false, err
	}

	// Handle the change for each phase
	if change != nil {
		switch change.Status.Phase {
		case changetypes.Phase_CHANGE:
			return r.reconcileChange(change)
		case changetypes.Phase_ROLLBACK:
			return r.reconcileRollback(change)
		}
	}
	return true, nil
}

// reconcileChange reconciles a change in the CHANGE phase
func (r *Reconciler) reconcileChange(change *networktypes.NetworkChange) (bool, error) {
	// Handle each possible state of the phase
	switch change.Status.State {
	case changetypes.State_PENDING:
		return r.reconcilePendingChange(change)
	case changetypes.State_RUNNING:
		return r.reconcileRunningChange(change)
	default:
		return true, nil
	}
}

// reconcilePendingChange reconciles a change in the PENDING state during the CHANGE phase
func (r *Reconciler) reconcilePendingChange(change *networktypes.NetworkChange) (bool, error) {
	// Create device changes if the prior change has been propagated
	if !hasDeviceChanges(change) {
		return r.createDeviceChanges(change)
	}

	// Determine whether the change can be applied
	canApply, err := r.canApplyChange(change)
	if err != nil {
		return false, err
	} else if !canApply {
		return false, nil
	}

	// If the change can be applied, update the change state to RUNNING
	change.Status.State = changetypes.State_RUNNING
	if err := r.networkChanges.Update(change); err != nil {
		return false, err
	}
	return true, nil
}

// hasDeviceChanges indicates whether the given change has created device changes
func hasDeviceChanges(change *networktypes.NetworkChange) bool {
	return change.Refs != nil && len(change.Refs) > 0
}

// createDeviceChanges creates device changes in sequential order
func (r *Reconciler) createDeviceChanges(networkChange *networktypes.NetworkChange) (bool, error) {
	// If the previous network change has not created device changes, requeue to wait for changes to be propagated
	prevChange, err := r.networkChanges.GetByIndex(networkChange.Index - 1)
	if err != nil {
		return false, err
	} else if prevChange != nil && !hasDeviceChanges(prevChange) {
		return false, nil
	}

	// Loop through changes and create device changes
	refs := make([]*networktypes.DeviceChangeRef, len(networkChange.Changes))
	for i, change := range networkChange.Changes {
		deviceChange := &devicetypes.DeviceChange{
			NetworkChange: devicetypes.NetworkChangeRef{
				ID:    types.ID(networkChange.ID),
				Index: types.Index(networkChange.Index),
			},
			Change: change,
		}
		if err := r.deviceChanges.Create(deviceChange); err != nil {
			return false, err
		}
		refs[i] = &networktypes.DeviceChangeRef{
			DeviceChangeID: deviceChange.ID,
		}
	}

	// If references have been updated, store the refs and succeed the reconciliation
	networkChange.Refs = refs
	if err := r.networkChanges.Update(networkChange); err != nil {
		return false, err
	}
	return true, nil
}

// canApplyChange returns a bool indicating whether the change can be applied
func (r *Reconciler) canApplyChange(change *networktypes.NetworkChange) (bool, error) {
	sequential := true
	for index := r.changeIndex; index < change.Index; index++ {
		priorChange, err := r.networkChanges.GetByIndex(index)
		if err != nil {
			return false, err
		} else if priorChange != nil {
			if priorChange.Status.State == changetypes.State_PENDING || priorChange.Status.State == changetypes.State_RUNNING {
				if isIntersectingChange(change, priorChange) {
					return false, nil
				}
				sequential = false
			} else {
				if sequential {
					r.changeIndex++
				}
			}
		}
	}
	return true, nil
}

// reconcileRunningChange reconciles a change in the RUNNING state during the CHANGE phase
func (r *Reconciler) reconcileRunningChange(change *networktypes.NetworkChange) (bool, error) {
	// Get the current state of all device changes for the change
	deviceChanges, err := r.getDeviceChanges(change)
	if err != nil {
		return false, err
	}

	// Ensure the device changes are being applied
	succeeded, err := r.ensureDeviceChangesRunning(deviceChanges)
	if succeeded || err != nil {
		return succeeded, err
	}

	// If all device changes are complete, mark the network change complete
	if r.isDeviceChangesComplete(deviceChanges) {
		change.Status.State = changetypes.State_COMPLETE
		if err := r.networkChanges.Update(change); err != nil {
			return false, err
		}
		return true, nil
	}

	// If a device change failed, rollback pending changes and requeue the change
	if r.isDeviceChangesFailed(deviceChanges) {
		// Ensure changes that have not failed are being rolled back
		succeeded, err = r.ensureDeviceChangeRollbacksRunning(deviceChanges)
		if succeeded || err != nil {
			return succeeded, err
		}

		// If all device change rollbacks have completed, revert the network change to PENDING
		if r.isDeviceChangeRollbacksComplete(deviceChanges) {
			change.Status.State = changetypes.State_PENDING
			change.Status.Reason = changetypes.Reason_ERROR
			if err := r.networkChanges.Update(change); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// ensureDeviceChangesRunning ensures device changes are in the running state
func (r *Reconciler) ensureDeviceChangesRunning(changes []*devicetypes.DeviceChange) (bool, error) {
	// Ensure all device changes are being applied
	updated := false
	for _, deviceChange := range changes {
		if deviceChange.Status.State == changetypes.State_PENDING {
			deviceChange.Status.State = changetypes.State_RUNNING
			if err := r.deviceChanges.Update(deviceChange); err != nil {
				return false, err
			}
			updated = true
		}
	}
	return updated, nil
}

// getDeviceChanges gets the device changes for the given network change
func (r *Reconciler) getDeviceChanges(networkChange *networktypes.NetworkChange) ([]*devicetypes.DeviceChange, error) {
	deviceChanges := make([]*devicetypes.DeviceChange, len(networkChange.Changes))
	for i, changeRef := range networkChange.Refs {
		deviceChange, err := r.deviceChanges.Get(changeRef.DeviceChangeID)
		if err != nil {
			return nil, err
		}
		deviceChanges[i] = deviceChange
	}
	return deviceChanges, nil
}

// isDeviceChangesComplete checks whether the device changes are complete
func (r *Reconciler) isDeviceChangesComplete(changes []*devicetypes.DeviceChange) bool {
	for _, change := range changes {
		if change.Status.State != changetypes.State_COMPLETE {
			return false
		}
	}
	return true
}

// isDeviceChangesFailed checks whether the device changes are complete
func (r *Reconciler) isDeviceChangesFailed(changes []*devicetypes.DeviceChange) bool {
	for _, change := range changes {
		if change.Status.State == changetypes.State_FAILED {
			return true
		}
	}
	return false
}

// ensureDeviceChangeRollbacksRunning ensures RUNNING or COMPLETE device changes are being rolled back
func (r *Reconciler) ensureDeviceChangeRollbacksRunning(changes []*devicetypes.DeviceChange) (bool, error) {
	updated := false
	for _, deviceChange := range changes {
		if deviceChange.Status.Phase == changetypes.Phase_CHANGE && deviceChange.Status.State != changetypes.State_FAILED {
			deviceChange.Status.Phase = changetypes.Phase_ROLLBACK
			deviceChange.Status.State = changetypes.State_RUNNING
			if err := r.deviceChanges.Update(deviceChange); err != nil {
				return false, err
			}
			updated = true
		}
	}
	return updated, nil
}

// isDeviceChangeRollbacksComplete determines whether a rollback of device changes is complete
func (r *Reconciler) isDeviceChangeRollbacksComplete(changes []*devicetypes.DeviceChange) bool {
	for _, deviceChange := range changes {
		if deviceChange.Status.Phase == changetypes.Phase_ROLLBACK && deviceChange.Status.State != changetypes.State_COMPLETE {
			return false
		}
	}
	return true
}

// reconcileRollback reconciles a change in the ROLLBACK phase
func (r *Reconciler) reconcileRollback(change *networktypes.NetworkChange) (bool, error) {
	// Ensure the device changes are in the ROLLBACK phase
	updated, err := r.ensureDeviceRollbacks(change)
	if updated || err != nil {
		return updated, err
	}

	// Handle each possible state of the phase
	switch change.Status.State {
	case changetypes.State_PENDING:
		return r.reconcilePendingRollback(change)
	case changetypes.State_RUNNING:
		return r.reconcileRunningRollback(change)
	default:
		return true, nil
	}
}

// ensureDeviceRollbacks ensures device rollbacks are pending
func (r *Reconciler) ensureDeviceRollbacks(networkChange *networktypes.NetworkChange) (bool, error) {
	// Ensure all device changes are being rolled back
	updated := false
	for _, changeRef := range networkChange.Refs {
		deviceChange, err := r.deviceChanges.Get(changeRef.DeviceChangeID)
		if err != nil {
			return false, err
		}

		if deviceChange.Status.Phase != changetypes.Phase_ROLLBACK {
			deviceChange.Status.Phase = changetypes.Phase_ROLLBACK
			deviceChange.Status.State = changetypes.State_PENDING
			if err := r.deviceChanges.Update(deviceChange); err != nil {
				return false, err
			}
			updated = true
		}
	}
	return updated, nil
}

// reconcilePendingRollback reconciles a change in the PENDING state during the ROLLBACK phase
func (r *Reconciler) reconcilePendingRollback(change *networktypes.NetworkChange) (bool, error) {
	// Determine whether the rollback can be applied
	canApply, err := r.canApplyRollback(change)
	if err != nil {
		return false, err
	} else if !canApply {
		return false, nil
	}

	// If the rollback can be applied, update the change state to RUNNING
	change.Status.State = changetypes.State_RUNNING
	if err := r.networkChanges.Update(change); err != nil {
		return false, err
	}
	return true, nil
}

// canApplyRollback returns a bool indicating whether the rollback can be applied
func (r *Reconciler) canApplyRollback(networkChange *networktypes.NetworkChange) (bool, error) {
	ch := make(chan *networktypes.NetworkChange)
	if err := r.networkChanges.List(ch); err != nil {
		return false, err
	}

	for change := range ch {
		if change.Index > networkChange.Index && isIntersectingChange(networkChange, change) && change.Status.State != changetypes.State_COMPLETE && change.Status.State != changetypes.State_FAILED {
			return false, nil
		}
	}
	return true, nil
}

// reconcileRunningRollback reconciles a change in the RUNNING state during the ROLLBACK phase
func (r *Reconciler) reconcileRunningRollback(change *networktypes.NetworkChange) (bool, error) {
	// Ensure the device rollbacks are running
	succeeded, err := r.ensureDeviceRollbacksRunning(change)
	if succeeded || err != nil {
		return succeeded, err
	}

	// If the rollback is complete, update the change state. Otherwise discard the change.
	complete, err := r.isRollbackComplete(change)
	if err != nil {
		return false, err
	} else if !complete {
		return true, nil
	}

	change.Status.State = changetypes.State_COMPLETE
	if err := r.networkChanges.Update(change); err != nil {
		return false, nil
	}
	return true, nil
}

// ensureDeviceRollbacksRunning ensures device rollbacks are in the running state
func (r *Reconciler) ensureDeviceRollbacksRunning(networkChange *networktypes.NetworkChange) (bool, error) {
	updated := false
	for _, changeRef := range networkChange.Refs {
		deviceChange, err := r.deviceChanges.Get(changeRef.DeviceChangeID)
		if err != nil {
			return false, err
		}

		if deviceChange.Status.State == changetypes.State_PENDING {
			deviceChange.Status.State = changetypes.State_RUNNING
			if err := r.deviceChanges.Update(deviceChange); err != nil {
				return false, err
			}
			updated = true
		}
	}
	return updated, nil
}

// isRollbackComplete determines whether a rollback is complete
func (r *Reconciler) isRollbackComplete(networkChange *networktypes.NetworkChange) (bool, error) {
	complete := 0
	for _, changeRef := range networkChange.Refs {
		deviceChange, err := r.deviceChanges.Get(changeRef.DeviceChangeID)
		if err != nil {
			return false, err
		}

		if deviceChange.Status.State == changetypes.State_COMPLETE {
			complete++
		}
	}
	return complete == len(networkChange.Refs), nil
}

// isIntersectingChange indicates whether the changes from the two given NetworkChanges intersect
func isIntersectingChange(config *networktypes.NetworkChange, history *networktypes.NetworkChange) bool {
	for _, configChange := range config.Changes {
		for _, historyChange := range history.Changes {
			if configChange.DeviceID == historyChange.DeviceID {
				return true
			}
		}
	}
	return false
}

var _ controller.Reconciler = &Reconciler{}
