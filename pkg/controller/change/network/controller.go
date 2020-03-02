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
	"github.com/onosproject/onos-config/api/types"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	"github.com/onosproject/onos-config/pkg/controller"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	devicetopo "github.com/onosproject/onos-topo/api/device"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = logging.GetLogger("controller", "change", "network")

// NewController returns a new config controller
func NewController(leadership leadershipstore.Store, deviceCache cache.Cache, devices devicestore.Store, networkChanges networkchangestore.Store, deviceChanges devicechangestore.Store) *controller.Controller {
	c := controller.NewController("NetworkChange")
	c.Activate(&controller.LeadershipActivator{
		Store: leadership,
	})
	c.Watch(&Watcher{
		Store: networkChanges,
	})
	c.Watch(&DeviceWatcher{
		DeviceCache: deviceCache,
		DeviceStore: devices,
		ChangeStore: deviceChanges,
	})
	c.Reconcile(&Reconciler{
		networkChanges: networkChanges,
		deviceChanges:  deviceChanges,
		devices:        devices,
	})
	return c
}

// Reconciler is a config reconciler
type Reconciler struct {
	networkChanges networkchangestore.Store
	deviceChanges  devicechangestore.Store
	devices        devicestore.Store
}

// Reconcile reconciles the state of a network configuration
func (r *Reconciler) Reconcile(id types.ID) (controller.Result, error) {
	change, err := r.networkChanges.Get(networkchange.ID(id))
	if err != nil {
		log.Warnf("Could not get NetworkChange %s", id)
		return controller.Result{}, err
	}

	log.Infof("Reconciling NetworkChange %v", change)

	// Handle the change for each phase
	if change != nil {
		switch change.Status.Phase {
		case changetypes.Phase_CHANGE:
			return r.reconcileChange(change)
		case changetypes.Phase_ROLLBACK:
			return r.reconcileRollback(change)
		}
	}
	return controller.Result{}, nil
}

// reconcileChange reconciles a change in the CHANGE phase
func (r *Reconciler) reconcileChange(change *networkchange.NetworkChange) (controller.Result, error) {
	switch change.Status.State {
	case changetypes.State_PENDING:
		return r.reconcilePendingChange(change)
	case changetypes.State_COMPLETE:
		return r.reconcileCompleteChange(change)
	}
	return controller.Result{}, nil
}

func (r *Reconciler) reconcilePendingChange(change *networkchange.NetworkChange) (controller.Result, error) {
	// Create device changes if necessary
	if !hasDeviceChanges(change) {
		return r.createDeviceChanges(change)
	}

	// Get the current state of all device changes for the change
	deviceChanges, err := r.getDeviceChanges(change)
	if err != nil {
		return controller.Result{}, err
	}

	// Ensure device changes are pending for the current incarnation
	changed, err := r.ensureDeviceChangesPending(change, deviceChanges)
	if changed || err != nil {
		return controller.Result{}, err
	}

	// If the network change can be applied, apply it by incrementing the incarnation number
	apply, err := r.canTryChange(change, deviceChanges)
	if err != nil {
		return controller.Result{}, err
	} else if apply {
		change.Status.Incarnation++
		change.Status.State = changetypes.State_PENDING
		change.Status.Reason = changetypes.Reason_NONE
		change.Status.Message = ""
		log.Infof("Applying NetworkChange %v", change)
		if err := r.networkChanges.Update(change); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// If all device changes are complete, complete the network change
	if r.isDeviceChangesComplete(change, deviceChanges) {
		change.Status.State = changetypes.State_COMPLETE
		log.Infof("Completing NetworkChange %v", change)
		if err := r.networkChanges.Update(change); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// If any device change has failed, roll back all device changes
	if r.isDeviceChangesFailed(change, deviceChanges) {
		return r.ensureDeviceChangeRollbacks(change, deviceChanges)
	}
	return controller.Result{}, nil
}

// reconcileCompleteChange reconciles a change in the COMPLETE state during the CHANGE phase
func (r *Reconciler) reconcileCompleteChange(change *networkchange.NetworkChange) (controller.Result, error) {
	nextChange, err := r.networkChanges.GetNext(change.Index)
	if err != nil {
		return controller.Result{}, err
	}

	for nextChange != nil {
		if isIntersectingChange(change, nextChange) {
			if nextChange.Status.State == changetypes.State_PENDING {
				return controller.Result{Requeue: types.ID(nextChange.ID)}, nil
			}
			return controller.Result{}, nil
		}

		nextChange, err = r.networkChanges.GetNext(nextChange.Index)
		if err != nil {
			return controller.Result{}, err
		}
	}
	return controller.Result{}, nil
}

// hasDeviceChanges indicates whether the given change has created device changes
func hasDeviceChanges(change *networkchange.NetworkChange) bool {
	return change.Refs != nil && len(change.Refs) > 0
}

// createDeviceChanges creates device changes in sequential order
func (r *Reconciler) createDeviceChanges(networkChange *networkchange.NetworkChange) (controller.Result, error) {
	// If the previous network change has not created device changes, requeue to wait for changes to be propagated
	// TODO devices changes should be written to stores by index to avoid having to manage index order
	prevChange, err := r.networkChanges.GetByIndex(networkChange.Index - 1)
	if err != nil {
		return controller.Result{}, err
	} else if prevChange != nil && !hasDeviceChanges(prevChange) {
		return controller.Result{Requeue: types.ID(networkChange.ID)}, nil
	}

	// Loop through changes and create device changes
	refs := make([]*networkchange.DeviceChangeRef, len(networkChange.Changes))
	for i, change := range networkChange.Changes {
		deviceChange := &devicechange.DeviceChange{
			Index: devicechange.Index(networkChange.Index),
			NetworkChange: devicechange.NetworkChangeRef{
				ID:    types.ID(networkChange.ID),
				Index: types.Index(networkChange.Index),
			},
			Change: change,
		}
		log.Infof("Creating DeviceChange %v for %v", deviceChange, networkChange.ID)
		if err := r.deviceChanges.Create(deviceChange); err != nil {
			return controller.Result{}, err
		}
		refs[i] = &networkchange.DeviceChangeRef{
			DeviceChangeID: deviceChange.ID,
		}
	}

	// If references have been updated, store the refs and succeed the reconciliation
	networkChange.Refs = refs
	if err := r.networkChanges.Update(networkChange); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{Requeue: types.ID(networkChange.ID)}, nil
}

// canTryChange returns a bool indicating whether the change can be attempted
func (r *Reconciler) canTryChange(change *networkchange.NetworkChange, deviceChanges []*devicechange.DeviceChange) (bool, error) {
	// If the incarnation number is positive, verify all device changes have been rolled back
	if change.Status.Incarnation > 0 {
		for _, deviceChange := range deviceChanges {
			if deviceChange.Status.Incarnation != change.Status.Incarnation ||
				deviceChange.Status.Phase != changetypes.Phase_ROLLBACK ||
				deviceChange.Status.State != changetypes.State_COMPLETE {
				return false, nil
			}
		}
	}

	// First, check if the devices affected by the change are available
	for _, deviceChange := range change.Changes {
		device, err := r.devices.Get(devicetopo.ID(deviceChange.DeviceID))
		if err != nil && status.Code(err) != codes.NotFound {
			return false, err
		} else if device == nil {
			return false, nil
		}
		state := getProtocolState(device)
		if state != devicetopo.ChannelState_CONNECTED {
			log.Infof("Cannot apply NetworkChange %v: %v is offline", change.ID, deviceChange.DeviceID)
			return false, nil
		}
	}

	// If the devices are available, ensure the change does not intersect prior changes
	prevChange, err := r.networkChanges.GetPrev(change.Index)
	if err != nil {
		return false, err
	}

	for prevChange != nil {
		// If the change intersects this change, verify it's complete
		if isIntersectingChange(change, prevChange) {
			// If the change is in the CHANGE phase, verify it's complete
			// If the change is in the ROLLBACK phase, verify it's complete but continue iterating
			// back to the last CHANGE phase change
			if prevChange.Status.Phase == changetypes.Phase_CHANGE {
				return prevChange.Status.State != changetypes.State_PENDING, nil
			} else if prevChange.Status.Phase == changetypes.Phase_ROLLBACK {
				if prevChange.Status.State == changetypes.State_PENDING {
					return false, nil
				}
			}
		}

		prevChange, err = r.networkChanges.GetPrev(prevChange.Index)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// ensureDeviceChangesPending ensures device changes are pending
func (r *Reconciler) ensureDeviceChangesPending(networkChange *networkchange.NetworkChange, changes []*devicechange.DeviceChange) (bool, error) {
	// Ensure all device changes are being applied
	updated := false
	for _, deviceChange := range changes {
		if deviceChange.Status.Incarnation < networkChange.Status.Incarnation {
			deviceChange.Status.Incarnation = networkChange.Status.Incarnation
			deviceChange.Status.Phase = changetypes.Phase_CHANGE
			deviceChange.Status.State = changetypes.State_PENDING
			deviceChange.Status.Reason = changetypes.Reason_NONE
			log.Infof("Running DeviceChange %v", deviceChange)
			if err := r.deviceChanges.Update(deviceChange); err != nil {
				return false, err
			}
			updated = true
		}
	}
	return updated, nil
}

// getDeviceChanges gets the device changes for the given network change
func (r *Reconciler) getDeviceChanges(networkChange *networkchange.NetworkChange) ([]*devicechange.DeviceChange, error) {
	deviceChanges := make([]*devicechange.DeviceChange, len(networkChange.Changes))
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
func (r *Reconciler) isDeviceChangesComplete(networkChange *networkchange.NetworkChange, changes []*devicechange.DeviceChange) bool {
	for _, change := range changes {
		if change.Status.Incarnation != networkChange.Status.Incarnation ||
			change.Status.Phase != changetypes.Phase_CHANGE ||
			change.Status.State != changetypes.State_COMPLETE {
			return false
		}
	}
	return true
}

// isDeviceChangesFailed checks whether any device change has failed for the current incarnation
func (r *Reconciler) isDeviceChangesFailed(networkChange *networkchange.NetworkChange, changes []*devicechange.DeviceChange) bool {
	for _, change := range changes {
		if change.Status.Incarnation == networkChange.Status.Incarnation && change.Status.State == changetypes.State_FAILED {
			return true
		}
	}
	return false
}

// ensureDeviceChangeRollbacks ensures device changes are being rolled back
func (r *Reconciler) ensureDeviceChangeRollbacks(networkChange *networkchange.NetworkChange, changes []*devicechange.DeviceChange) (controller.Result, error) {
	for _, deviceChange := range changes {
		if deviceChange.Status.Incarnation != networkChange.Status.Incarnation ||
			deviceChange.Status.Phase != changetypes.Phase_ROLLBACK ||
			deviceChange.Status.State == changetypes.State_FAILED {
			deviceChange.Status.Incarnation = networkChange.Status.Incarnation
			deviceChange.Status.Phase = changetypes.Phase_ROLLBACK
			deviceChange.Status.State = changetypes.State_PENDING
			log.Infof("Rolling back DeviceChange %v", deviceChange)
			if err := r.deviceChanges.Update(deviceChange); err != nil {
				return controller.Result{}, err
			}
		}
	}
	return controller.Result{}, nil
}

// reconcileRollback reconciles a change in the ROLLBACK phase
func (r *Reconciler) reconcileRollback(change *networkchange.NetworkChange) (controller.Result, error) {
	switch change.Status.State {
	case changetypes.State_PENDING:
		return r.reconcilePendingRollback(change)
	case changetypes.State_COMPLETE:
		return r.reconcileCompleteRollback(change)
	}
	return controller.Result{}, nil
}

// reconcilePendingRollback reconciles a change in the ROLLBACK phase
func (r *Reconciler) reconcilePendingRollback(change *networkchange.NetworkChange) (controller.Result, error) {
	// Ensure the device changes are in the ROLLBACK phase
	updated, err := r.ensureDeviceRollbacks(change)
	if updated || err != nil {
		return controller.Result{}, err
	}

	// If the change is not pending, skip it
	if change.Status.State != changetypes.State_PENDING {
		return controller.Result{}, nil
	}

	// Get the current state of all device changes for the change
	deviceChanges, err := r.getDeviceChanges(change)
	if err != nil {
		return controller.Result{}, err
	}

	// If the network rollback can be applied, apply it by incrementing the incarnation number
	apply, err := r.canTryRollback(change, deviceChanges)
	if err != nil {
		return controller.Result{}, err
	} else if apply {
		change.Status.Incarnation++
		change.Status.State = changetypes.State_PENDING
		change.Status.Reason = changetypes.Reason_NONE
		change.Status.Message = ""
		log.Infof("Rolling back NetworkChange %v", change)
		if err := r.networkChanges.Update(change); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// If all device rollbacks are complete, complete the network change
	if r.isDeviceRollbacksComplete(change, deviceChanges) {
		change.Status.State = changetypes.State_COMPLETE
		log.Infof("Completing NetworkChange %v", change)
		if err := r.networkChanges.Update(change); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// If any device change has failed, roll back all device changes
	if r.isDeviceChangesFailed(change, deviceChanges) {
		return r.ensureDeviceChangeRollbacks(change, deviceChanges)
	}
	return controller.Result{}, nil
}

// reconcileCompleteRollback reconciles a change in the COMPLETE state during the CHANGE phase
func (r *Reconciler) reconcileCompleteRollback(change *networkchange.NetworkChange) (controller.Result, error) {
	prevChange, err := r.networkChanges.GetPrev(change.Index)
	if err != nil {
		return controller.Result{}, err
	}

	for prevChange != nil {
		if isIntersectingChange(change, prevChange) {
			if prevChange.Status.State == changetypes.State_PENDING {
				return controller.Result{Requeue: types.ID(prevChange.ID)}, nil
			}
			return controller.Result{}, nil
		}

		prevChange, err = r.networkChanges.GetPrev(prevChange.Index)
		if err != nil {
			return controller.Result{}, err
		}
	}
	return controller.Result{}, nil
}

// ensureDeviceRollbacks ensures device rollbacks are pending
func (r *Reconciler) ensureDeviceRollbacks(networkChange *networkchange.NetworkChange) (bool, error) {
	// Ensure all device changes are being rolled back
	updated := false
	for _, changeRef := range networkChange.Refs {
		deviceChange, err := r.deviceChanges.Get(changeRef.DeviceChangeID)
		if err != nil {
			return false, err
		}

		if deviceChange.Status.Incarnation < networkChange.Status.Incarnation ||
			deviceChange.Status.Phase != changetypes.Phase_ROLLBACK {
			deviceChange.Status.Incarnation = networkChange.Status.Incarnation
			deviceChange.Status.Phase = changetypes.Phase_ROLLBACK
			deviceChange.Status.State = changetypes.State_PENDING
			log.Infof("Rolling back DeviceChange %v", deviceChange)
			if err := r.deviceChanges.Update(deviceChange); err != nil {
				return false, err
			}
			updated = true
		}
	}
	return updated, nil
}

// isDeviceRollbacksComplete checks whether the device rollbacks are complete
func (r *Reconciler) isDeviceRollbacksComplete(networkChange *networkchange.NetworkChange, changes []*devicechange.DeviceChange) bool {
	for _, change := range changes {
		if change.Status.Incarnation != networkChange.Status.Incarnation ||
			change.Status.Phase != changetypes.Phase_ROLLBACK ||
			change.Status.State != changetypes.State_COMPLETE {
			return false
		}
	}
	return true
}

// canTryRollback returns a bool indicating whether the rollback can be attempted
func (r *Reconciler) canTryRollback(change *networkchange.NetworkChange, deviceChanges []*devicechange.DeviceChange) (bool, error) {
	// Verify all device changes are being rolled back
	if change.Status.Incarnation > 0 {
		for _, deviceChange := range deviceChanges {
			if deviceChange.Status.Incarnation != change.Status.Incarnation ||
				deviceChange.Status.Phase != changetypes.Phase_ROLLBACK ||
				deviceChange.Status.State != changetypes.State_FAILED {
				return false, nil
			}
		}
	}

	nextChange, err := r.networkChanges.GetNext(change.Index)
	if err != nil {
		return false, err
	}

	for nextChange != nil {
		// If the change intersects this change, verify it has been rolled back
		if isIntersectingChange(change, nextChange) {
			return nextChange.Status.Phase == changetypes.Phase_ROLLBACK &&
				(nextChange.Status.State == changetypes.State_COMPLETE ||
					nextChange.Status.State == changetypes.State_FAILED), nil
		}

		nextChange, err = r.networkChanges.GetNext(nextChange.Index)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

// isIntersectingChange indicates whether the changes from the two given NetworkChanges intersect
func isIntersectingChange(config *networkchange.NetworkChange, history *networkchange.NetworkChange) bool {
	for _, configChange := range config.Changes {
		for _, historyChange := range history.Changes {
			if configChange.DeviceID == historyChange.DeviceID {
				return true
			}
		}
	}
	return false
}

func getProtocolState(device *devicetopo.Device) devicetopo.ChannelState {
	// Find the gNMI protocol state for the device
	var protocol *devicetopo.ProtocolState
	for _, p := range device.Protocols {
		if p.Protocol == devicetopo.Protocol_GNMI {
			protocol = p
			break
		}
	}
	if protocol == nil {
		return devicetopo.ChannelState_UNKNOWN_CHANNEL_STATE
	}
	return protocol.ChannelState
}

var _ controller.Reconciler = &Reconciler{}
