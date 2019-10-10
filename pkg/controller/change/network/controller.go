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
		// For all phases, ensure device changes have been created in the device change store
		succeeded, err := r.ensureDeviceChanges(change)
		if succeeded || err != nil {
			return succeeded, err
		}

		switch change.Status.Phase {
		case changetypes.Phase_CHANGE:
			return r.reconcileChange(change)
		case changetypes.Phase_ROLLBACK:
			return r.reconcileRollback(change)
		}
	}
	return true, nil
}

// ensureDeviceChanges ensures device changes have been created for all changes in the network change
func (r *Reconciler) ensureDeviceChanges(config *networktypes.NetworkChange) (bool, error) {
	// Loop through changes and create if necessary
	updated := false
	for _, change := range config.Changes {
		if change.ID == "" {
			deviceChange := &devicetypes.Change{
				NetworkChangeID: types.ID(config.ID),
				DeviceID:        change.DeviceID,
				DeviceVersion:   change.DeviceVersion,
				Values:          change.Values,
			}
			if err := r.deviceChanges.Create(deviceChange); err != nil {
				return false, err
			}
			change.ID = deviceChange.ID
			change.Index = deviceChange.Index
			updated = true
		}
	}

	// If indexes have been updated, store the indexes first in the network change
	if updated {
		if err := r.networkChanges.Update(config); err != nil {
			return false, err
		}
	}
	return updated, nil
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

// canApplyChange returns a bool indicating whether the change can be applied
func (r *Reconciler) canApplyChange(change *networktypes.NetworkChange) (bool, error) {
	sequential := true
	for index := r.changeIndex; index < change.Index; index++ {
		priorChange, err := r.networkChanges.GetByIndex(index)
		if err != nil {
			return false, err
		} else if priorChange != nil {
			if priorChange.Status.State == changetypes.State_PENDING || priorChange.Status.State == changetypes.State_RUNNING {
				if intersecting(change, priorChange) {
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
	// Ensure the device changes are being applied
	succeeded, err := r.ensureDeviceChangesRunning(change)
	if succeeded || err != nil {
		return succeeded, err
	}

	// Get the current state of all device changes for the change
	deviceChanges := make([]*devicetypes.Change, len(change.Changes))
	for i, changeReq := range change.Changes {
		deviceChange, err := r.deviceChanges.Get(changeReq.ID)
		if err != nil {
			return false, err
		}
		deviceChanges[i] = deviceChange
	}

	// Check all device changes for completion or failures
	complete := 0
	var failure *changetypes.Reason
	for _, deviceChange := range deviceChanges {
		// If a device change is complete, count the change
		// If a device change fails, determine whether the failure allows for retry
		if deviceChange.Status.State == changetypes.State_COMPLETE {
			complete++
		} else if deviceChange.Status.State == changetypes.State_FAILED {
			failure = &deviceChange.Status.Reason
		}
	}

	// If all the device changes have completed successfully, complete the change
	// If one of the device changes failed, ensure remaining changes are rolled back
	// and then fail the change
	if failure != nil {
		// After a failure, we need to roll back any pending or successful device changes.
		// Iterate through device changes and revert any that has failed.
		rollbackStarted := false
		rollbackComplete := true
		for _, deviceChange := range deviceChanges {
			if deviceChange.Status.Phase == changetypes.Phase_CHANGE && deviceChange.Status.State != changetypes.State_FAILED {
				deviceChange.Status.Phase = changetypes.Phase_ROLLBACK
				deviceChange.Status.State = changetypes.State_RUNNING
				if err := r.deviceChanges.Update(deviceChange); err != nil {
					return false, err
				}
				rollbackStarted = true
			} else if deviceChange.Status.Phase == changetypes.Phase_ROLLBACK && deviceChange.Status.State != changetypes.State_COMPLETE {
				rollbackComplete = false
			}
		}

		// If device changes were rolled back, discard this change to wait for rollbacks to complete
		if rollbackStarted {
			return true, nil
		} else if rollbackComplete {
			// If the failure was due to a loss of availability, revert the change back to the PENDING state
			// Otherwise, fail the change
			if *failure == changetypes.Reason_UNAVAILABLE {
				change.Status.State = changetypes.State_PENDING
				if err := r.networkChanges.Update(change); err != nil {
					return false, err
				}
			} else {
				change.Status.State = changetypes.State_FAILED
				change.Status.Reason = *failure
				if err := r.networkChanges.Update(change); err != nil {
					return false, err
				}
			}
			return true, nil
		}
	} else if complete == len(change.Changes) {
		change.Status.State = changetypes.State_COMPLETE
		if err := r.networkChanges.Update(change); err != nil {
			return false, err
		}
	}
	return true, nil
}

// ensureDeviceChangesRunning ensures device changes are in the running state
func (r *Reconciler) ensureDeviceChangesRunning(change *networktypes.NetworkChange) (bool, error) {
	// Ensure all device changes are being applied
	updated := false
	for _, changeReq := range change.Changes {
		deviceChange, err := r.deviceChanges.Get(changeReq.ID)
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
func (r *Reconciler) ensureDeviceRollbacks(change *networktypes.NetworkChange) (bool, error) {
	// Ensure all device changes are being rolled back
	updated := false
	for _, changeReq := range change.Changes {
		deviceChange, err := r.deviceChanges.Get(changeReq.ID)
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
func (r *Reconciler) canApplyRollback(change *networktypes.NetworkChange) (bool, error) {
	lastIndex, err := r.networkChanges.LastIndex()
	if err != nil {
		return false, err
	}

	for index := change.Index + 1; index <= lastIndex; index++ {
		futureChange, err := r.networkChanges.GetByIndex(index)
		if err != nil {
			return false, err
		} else if futureChange != nil && intersecting(change, futureChange) && futureChange.Status.State != changetypes.State_COMPLETE && futureChange.Status.State != changetypes.State_FAILED {
			return false, err
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
func (r *Reconciler) ensureDeviceRollbacksRunning(change *networktypes.NetworkChange) (bool, error) {
	updated := false
	for _, changeReq := range change.Changes {
		deviceChange, err := r.deviceChanges.Get(changeReq.ID)
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
func (r *Reconciler) isRollbackComplete(change *networktypes.NetworkChange) (bool, error) {
	complete := 0
	for _, changeReq := range change.Changes {
		deviceChange, err := r.deviceChanges.Get(changeReq.ID)
		if err != nil {
			return false, err
		}

		if deviceChange.Status.State == changetypes.State_COMPLETE {
			complete++
		}
	}
	return complete == len(change.Changes), nil
}

// intersecting indicates whether the changes from the two given NetworkChanges intersect
func intersecting(config *networktypes.NetworkChange, history *networktypes.NetworkChange) bool {
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
