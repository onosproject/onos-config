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
	gogotypes "github.com/gogo/protobuf/types"
	types "github.com/onosproject/onos-api/go/onos/config"
	changetypes "github.com/onosproject/onos-api/go/onos/config/change"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	"github.com/onosproject/onos-api/go/onos/topo"
	configcontroller "github.com/onosproject/onos-config/pkg/controller"
	devicetopo "github.com/onosproject/onos-config/pkg/device"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("controller", "change", "network")

// NewController returns a new config controller
func NewController(leadership leadershipstore.Store, devices devicestore.Store, networkChanges networkchangestore.Store,
	deviceChanges devicechangestore.Store) *controller.Controller {
	c := controller.NewController("NetworkChange")
	c.Activate(&configcontroller.LeadershipActivator{
		Store: leadership,
	})
	c.Watch(&Watcher{
		Store: networkChanges,
	})
	c.Watch(&DeviceWatcher{
		DeviceStore: devices,
		ChangeStore: networkChanges,
	})
	c.Watch(&DeviceChangeWatcher{
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
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	log.Infof("Reconciling NetworkChange '%s'", id.String())
	change, err := r.networkChanges.Get(networkchange.ID(id.String()))
	if err != nil {
		if errors.IsNotFound(err) {
			return controller.Result{}, nil
		}
		log.Errorf("Error fetching NetworkChange '%s'", id.String(), err)
		return controller.Result{}, err
	}

	log.Debug(change)

	// If the change is not in the PENDING state it must be either COMPLETE or FAILED
	// It does not need to be reconciled.
	if change.Status.State != changetypes.State_PENDING {
		log.Debugf("Skipping reconciliation for NetworkChange '%s': Change is already %s", change.ID, change.Status.State)
		return controller.Result{}, nil
	}

	// Ensure a bi-directional link between the change and its dependency (if any)
	if linked, err := r.linkDependencies(change); err != nil {
		return controller.Result{}, err
	} else if linked {
		return controller.Result{}, nil
	}

	// Reconcile the change based on whether it's a CHANGE or ROLLBACK
	switch change.Status.Phase {
	case changetypes.Phase_CHANGE:
		return r.reconcileChange(change)
	case changetypes.Phase_ROLLBACK:
		return r.reconcileRollback(change)
	default:
		log.Errorf("Unexpected phase '%s' for NetworkChange '%s'", change.Status.Phase, change.ID)
	}
	return controller.Result{}, nil
}

// reconcileChange reconciles a change in the CHANGE phase
func (r *Reconciler) reconcileChange(networkChange *networkchange.NetworkChange) (controller.Result, error) {
	// Determine whether the network change is enabled (its dependencies have been completed)
	enabled, err := r.isChangeEnabled(networkChange)
	if err != nil {
		return controller.Result{}, err
	} else if !enabled {
		return controller.Result{}, nil
	}

	// Create device changes if necessary
	if created, err := r.createDeviceChanges(networkChange); err != nil {
		return controller.Result{}, err
	} else if created {
		return controller.Result{}, nil
	}

	log.Debugf("Reconciling devices changes for NetworkChange '%s'", networkChange.ID)
	deviceChanges, err := r.getDeviceChanges(networkChange)
	if err != nil {
		return controller.Result{}, err
	}

	// If all device changes are complete, complete the network change
	if r.isDeviceChangesComplete(deviceChanges) {
		log.Infof("Completing NetworkChange '%s'", networkChange.ID)
		networkChange.Status.State = changetypes.State_COMPLETE
		log.Debug(networkChange)
		if err := r.networkChanges.Update(networkChange); err != nil {
			if errors.IsNotFound(err) {
				return controller.Result{}, nil
			}
			log.Errorf("Error updating NetworkChange '%s'", networkChange.ID, err)
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// If any device change has failed, fail the network change
	if failed, msg := r.isDeviceChangesFailed(deviceChanges); failed {
		log.Infof("Failing NetworkChange '%s'", networkChange.ID)
		networkChange.Status.State = changetypes.State_FAILED
		networkChange.Status.Reason = changetypes.Reason_ERROR
		networkChange.Status.Message = msg
		log.Debug(networkChange)
		if err := r.networkChanges.Update(networkChange); err != nil {
			if errors.IsNotFound(err) {
				return controller.Result{}, nil
			}
			log.Errorf("Error updating NetworkChange '%s'", networkChange.ID, err)
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}

// isChangeEnabled returns a bool indicating whether the change can be applied
func (r *Reconciler) isChangeEnabled(networkChange *networkchange.NetworkChange) (bool, error) {
	if networkChange.Dependency == nil {
		log.Errorf("Missing dependency info for NetworkChange '%s'", networkChange.ID)
		return false, nil
	}

	if networkChangeID, ok := networkChange.Dependency.Id.(*networkchange.NetworkChangeRef_NetworkChangeID); ok {
		dependencyChange, err := r.networkChanges.Get(networkChangeID.NetworkChangeID)
		if err != nil {
			// If the dependency is not found, it must have been compacted into a snapshot. This means
			// the dependency is permanently complete so this change is enabled.
			if errors.IsNotFound(err) {
				return true, nil
			}
			log.Errorf("Error fetching NetworkChange '%s'", networkChangeID.NetworkChangeID, err)
			return false, err
		}
		if dependencyChange.Status.State == changetypes.State_PENDING {
			log.Debugf("Cannot apply NetworkChange '%s': waiting for dependency '%s'", networkChange.ID, dependencyChange.ID)
			return false, nil
		}
		return true, nil
	}
	return true, nil
}

// createDeviceChanges creates device changes
func (r *Reconciler) createDeviceChanges(networkChange *networkchange.NetworkChange) (bool, error) {
	if len(networkChange.Refs) > 0 {
		return false, nil
	}

	log.Infof("Applying NetworkChange '%s'", networkChange.ID)
	updated := false
	refs := make([]*networkchange.DeviceChangeRef, len(networkChange.Changes))
	for i, change := range networkChange.Changes {
		deviceChange := &devicechange.DeviceChange{
			ID:    devicechange.NewID(types.ID(networkChange.ID), change.DeviceID, change.DeviceVersion),
			Index: devicechange.Index(networkChange.Index),
			NetworkChange: devicechange.NetworkChangeRef{
				ID:    types.ID(networkChange.ID),
				Index: types.Index(networkChange.Index),
			},
			Change: change,
			Status: changetypes.Status{
				Phase: changetypes.Phase_CHANGE,
				State: changetypes.State_PENDING,
			},
		}

		// Create the device change if it doesn't exist
		log.Infof("Creating DeviceChange '%s'", deviceChange.ID)
		log.Debug(deviceChange)
		if err := r.deviceChanges.Create(deviceChange); err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Errorf("Error creating DeviceChange %+v", deviceChange, err)
				return false, err
			}
		} else {
			updated = true
		}
		refs[i] = &networkchange.DeviceChangeRef{
			DeviceChangeID: deviceChange.ID,
		}
	}

	if !updated {
		return false, nil
	}

	log.Infof("Binding %d device changes to NetworkChange '%s'", len(refs), networkChange.ID)
	networkChange.Refs = refs
	log.Debug(networkChange)
	if err := r.networkChanges.Update(networkChange); err != nil {
		if errors.IsNotFound(err) {
			return true, nil
		}
		log.Errorf("Error updating NetworkChange %+v", networkChange, err)
		return false, err
	}
	return true, nil
}

// isDeviceChangesComplete checks whether the device changes are complete
func (r *Reconciler) isDeviceChangesComplete(deviceChanges []*devicechange.DeviceChange) bool {
	for _, deviceChange := range deviceChanges {
		if deviceChange.Status.Phase != changetypes.Phase_CHANGE ||
			deviceChange.Status.State != changetypes.State_COMPLETE {
			return false
		}
	}
	return true
}

// isDeviceChangesFailed checks whether any device change has failed
func (r *Reconciler) isDeviceChangesFailed(deviceChanges []*devicechange.DeviceChange) (bool, string) {
	for _, deviceChange := range deviceChanges {
		if deviceChange.Status.State == changetypes.State_FAILED {
			log.Warnf("DeviceChange '%s' failed: '%s'", deviceChange.ID, deviceChange.Status.Message)
			return true, deviceChange.Status.Message
		}
	}
	return false, ""
}

// reconcileRollback reconciles a change in the ROLLBACK phase
func (r *Reconciler) reconcileRollback(networkChange *networkchange.NetworkChange) (controller.Result, error) {
	// Get the device changes associated with the network change
	deviceChanges, err := r.getDeviceChanges(networkChange)
	if err != nil {
		return controller.Result{}, err
	}

	// Determine whether the rollback is enabled
	enabled, err := r.isRollbackEnabled(networkChange)
	if err != nil {
		return controller.Result{}, err
	} else if !enabled {
		return controller.Result{}, nil
	}

	// Ensure the device changes are in the ROLLBACK phase
	if rolledBack, err := r.rollbackDeviceChanges(deviceChanges); err != nil {
		return controller.Result{}, err
	} else if rolledBack {
		return controller.Result{}, nil
	}

	log.Debugf("Reconciling devices changes for NetworkChange '%s'", networkChange.ID)

	// If all device rollbacks are complete, complete the network change rollback
	if r.isDeviceRollbacksComplete(deviceChanges) {
		log.Infof("Completing NetworkChange '%s'", networkChange.ID)
		networkChange.Status.State = changetypes.State_COMPLETE
		log.Debug(networkChange)
		if err := r.networkChanges.Update(networkChange); err != nil {
			if errors.IsNotFound(err) {
				return controller.Result{}, nil
			}
			log.Warnf("Error updating NetworkChange '%s'", networkChange.ID, err)
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// If any device rollback has failed, fail the network change rollback
	if failed, msg := r.isDeviceRollbacksFailed(deviceChanges); failed {
		log.Infof("Failing NetworkChange '%s'", networkChange.ID)
		networkChange.Status.State = changetypes.State_FAILED
		networkChange.Status.Reason = changetypes.Reason_ERROR
		networkChange.Status.Message = msg
		log.Debug(networkChange)
		if err := r.networkChanges.Update(networkChange); err != nil {
			if errors.IsNotFound(err) {
				return controller.Result{}, nil
			}
			log.Warnf("Error updating NetworkChange '%s'", networkChange.ID, err)
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}
	return controller.Result{}, nil
}

// isRollbackEnabled returns a bool indicating whether the rollback can be attempted
func (r *Reconciler) isRollbackEnabled(networkChange *networkchange.NetworkChange) (bool, error) {
	for _, dependent := range networkChange.Dependents {
		if networkChangeID, ok := dependent.Id.(*networkchange.NetworkChangeRef_NetworkChangeID); ok {
			dependentChange, err := r.networkChanges.Get(networkChangeID.NetworkChangeID)
			if err != nil {
				if errors.IsNotFound(err) {
					log.Errorf("NetworkChange '%s' dependent '%s' is missing from the store!", networkChange.ID, networkChangeID.NetworkChangeID, err)
				} else {
					log.Errorf("Error fetching '%s' dependent NetworkChange '%s'", networkChange.ID, networkChangeID.NetworkChangeID, err)
					return false, err
				}
			} else if dependentChange.Status.Phase != changetypes.Phase_ROLLBACK ||
				dependentChange.Status.State == changetypes.State_PENDING {
				log.Debugf("Cannot rollback NetworkChange '%s': waiting for dependent '%s'", networkChange.ID, networkChangeID.NetworkChangeID)
				return false, nil
			}
		}
	}
	return true, nil
}

// rollbackDeviceChanges changes the phase of the given device changes to ROLLBACK
func (r *Reconciler) rollbackDeviceChanges(deviceChanges []*devicechange.DeviceChange) (bool, error) {
	updated := false
	for _, deviceChange := range deviceChanges {
		if deviceChange.Status.Phase != changetypes.Phase_ROLLBACK {
			log.Infof("Rolling back DeviceChange %s", deviceChange.ID)
			deviceChange.Status.Phase = changetypes.Phase_ROLLBACK
			deviceChange.Status.State = changetypes.State_PENDING
			log.Debug(deviceChange)
			if err := r.deviceChanges.Update(deviceChange); err != nil {
				if !errors.IsNotFound(err) {
					log.Errorf("Error updating DeviceChange %+v", deviceChange, err)
					return false, err
				}
			} else {
				updated = true
			}
		}
	}
	return updated, nil
}

// isDeviceRollbacksComplete checks whether the device rollbacks are complete
func (r *Reconciler) isDeviceRollbacksComplete(deviceChanges []*devicechange.DeviceChange) bool {
	for _, deviceChange := range deviceChanges {
		if deviceChange.Status.Phase != changetypes.Phase_ROLLBACK ||
			deviceChange.Status.State != changetypes.State_COMPLETE {
			return false
		}
	}
	return true
}

// isDeviceRollbacksFailed checks whether the device rollbacks are failed
func (r *Reconciler) isDeviceRollbacksFailed(deviceChanges []*devicechange.DeviceChange) (bool, string) {
	for _, deviceChange := range deviceChanges {
		if deviceChange.Status.Phase == changetypes.Phase_ROLLBACK &&
			deviceChange.Status.State == changetypes.State_FAILED {
			log.Warnf("DeviceChange '%s' failed: '%s'", deviceChange.ID, deviceChange.Status.Message)
			return true, deviceChange.Status.Message
		}
	}
	return false, ""
}

func (r *Reconciler) linkDependencies(change *networkchange.NetworkChange) (bool, error) {
	if change.Dependency != nil {
		return false, nil
	}

	for index := change.Index - 1; index > 0; index-- {
		log.Debugf("Fetching previous NetworkChange at index %d", index)
		prevChange, err := r.networkChanges.GetByIndex(index)
		if err != nil {
			// If the change is not found, break out of the loop to set an empty dependency
			if errors.IsNotFound(err) {
				break
			}
			log.Errorf("Error fetching NetworkChange at index %d", index, err)
			return false, err
		}

		// If the change intersects this change, verify it's complete
		if isIntersectingChange(change, prevChange) {
			// Check if the change is already in the dependents list on the dependency change
			dependentExists := false
			for _, dependent := range prevChange.Dependents {
				if networkChangeID, ok := dependent.Id.(*networkchange.NetworkChangeRef_NetworkChangeID); ok &&
					networkChangeID.NetworkChangeID == change.ID {
					dependentExists = true
					break
				}
			}

			// If the change is not already in the dependents list, add it to the dependency change
			if !dependentExists {
				log.Debugf("Adding NetworkChange dependents '%s'->'%s'", prevChange.ID, change.ID)
				prevChange.Dependents = append(prevChange.Dependents, networkchange.NetworkChangeRef{
					Id: &networkchange.NetworkChangeRef_NetworkChangeID{
						NetworkChangeID: change.ID,
					},
				})
				log.Debug(prevChange)
				if err := r.networkChanges.Update(prevChange); err != nil && !errors.IsNotFound(err) {
					log.Errorf("Error updating NetworkChange '%s'", prevChange.ID, err)
					return false, err
				}
			}

			// Note it's critical that this be done *after* adding dependents so the existence
			// of a dependency field implies dependents were previously configured.
			log.Debugf("Adding NetworkChange dependency '%s'->'%s'", prevChange.ID, change.ID)
			change.Dependency = &networkchange.NetworkChangeRef{
				Id: &networkchange.NetworkChangeRef_NetworkChangeID{
					NetworkChangeID: prevChange.ID,
				},
			}
			log.Debug(change)
			if err := r.networkChanges.Update(change); err != nil && !errors.IsNotFound(err) {
				log.Errorf("Error updating NetworkChange '%s'", change.ID, err)
				return false, err
			}
			return true, nil
		}
	}

	// If no prior overlapping entry was found, the change has no dependency
	log.Debugf("Adding null NetworkChange dependency ->'%s'", change.ID)
	change.Dependency = &networkchange.NetworkChangeRef{
		Id: &networkchange.NetworkChangeRef_None{
			None: &gogotypes.Empty{},
		},
	}
	log.Debug(change)
	if err := r.networkChanges.Update(change); err != nil && !errors.IsNotFound(err) {
		log.Errorf("Error updating NetworkChange '%s'", change.ID, err)
		return false, err
	}
	return true, nil
}

// getDeviceChanges gets the device changes for the given network change
func (r *Reconciler) getDeviceChanges(networkChange *networkchange.NetworkChange) ([]*devicechange.DeviceChange, error) {
	deviceChanges := make([]*devicechange.DeviceChange, 0, len(networkChange.Changes))
	for _, changeRef := range networkChange.Refs {
		deviceChange, err := r.deviceChanges.Get(changeRef.DeviceChangeID)
		if err != nil {
			if !errors.IsNotFound(err) {
				log.Errorf("Error fetching DeviceChange '%s'", changeRef.DeviceChangeID, err)
				return nil, err
			}
		} else {
			deviceChanges = append(deviceChanges, deviceChange)
		}
	}
	return deviceChanges, nil
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

func getProtocolState(device *devicetopo.Device) topo.ChannelState {
	// Find the gNMI protocol state for the device
	var protocol *topo.ProtocolState
	for _, p := range device.Protocols {
		if p.Protocol == topo.Protocol_GNMI {
			protocol = p
			break
		}
	}
	if protocol == nil {
		return topo.ChannelState_UNKNOWN_CHANNEL_STATE
	}
	return protocol.ChannelState
}

var _ controller.Reconciler = &Reconciler{}
