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

package rollback

import (
	"github.com/onosproject/onos-config/pkg/controller"
	devicecontroller "github.com/onosproject/onos-config/pkg/controller/device/rollback"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/device/change"
	devicerollbackstore "github.com/onosproject/onos-config/pkg/store/device/rollback"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/network/change"
	networkrollbackstore "github.com/onosproject/onos-config/pkg/store/network/rollback"
	"github.com/onosproject/onos-config/pkg/types"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/device/change"
	devicerollbacktypes "github.com/onosproject/onos-config/pkg/types/device/rollback"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/network/change"
	networkrollbacktypes "github.com/onosproject/onos-config/pkg/types/network/rollback"
)

// NewController returns a new config controller
func NewController(leadership leadershipstore.Store, networkChanges networkchangestore.Store, deviceChanges devicechangestore.Store, networkRollbacks networkrollbackstore.Store, deviceRollbacks devicerollbackstore.Store) *controller.Controller {
	c := controller.NewController()
	c.Activate(&controller.LeadershipActivator{
		Store: leadership,
	})
	c.Watch(&Watcher{
		Store: networkRollbacks,
	})
	c.Watch(&devicecontroller.OwnerWatcher{
		Store: deviceRollbacks,
	})
	c.Reconcile(&Reconciler{
		networkChanges:   networkChanges,
		deviceChanges:    deviceChanges,
		networkRollbacks: networkRollbacks,
		deviceRollbacks:  deviceRollbacks,
	})
	return c
}

// Reconciler is a config reconciler
type Reconciler struct {
	networkChanges   networkchangestore.Store
	deviceChanges    devicechangestore.Store
	networkRollbacks networkrollbackstore.Store
	deviceRollbacks  devicerollbackstore.Store
}

// Reconcile reconciles the state of a network configuration
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	// Get the rollback configuration
	config, err := r.networkRollbacks.Get(networkrollbacktypes.ID(id))
	if err != nil {
		return false, err
	}

	// If the revision number is 0, the rollback has been removed
	if config.Revision == 0 {
		return r.deleteDeviceRollbacks(config)
	}

	// Get the network change
	networkChange, err := r.networkChanges.Get(networkchangetypes.ID(config.ChangeID))
	if err != nil {
		return false, err
	}

	// If the network change does not exist, fail the rollback
	if networkChange == nil {
		return r.failNetworkRollback(config)
	}

	// If the network change has not yet started, cancel it
	if networkChange.Status == networkchangetypes.Status_PENDING {
		return r.failNetworkChange(networkChange)
	}

	// Ensure rollbacks have been created in the device rollbacks store
	succeeded, err := r.createDeviceRollbacks(config)
	if !succeeded || err != nil {
		return succeeded, err
	}

	// If the change is in the PENDING state, check if it can be applied and apply it if so.
	if config.Status == networkrollbacktypes.Status_PENDING {
		apply, err := r.canApplyNetworkRollback(config, networkChange)
		if err != nil {
			return false, err
		} else if !apply {
			return true, nil
		}
		return r.applyNetworkRollback(config)
	}

	// If the change is the APPLYING state, ensure all device changes are in the APPLYING
	// state and check results
	if config.Status == networkrollbacktypes.Status_APPLYING {
		return r.applyDeviceChanges(config)
	}
	return true, nil
}

func (r *Reconciler) failNetworkRollback(rollback *networkrollbacktypes.NetworkRollback) (bool, error) {
	rollback.Status = networkrollbacktypes.Status_FAILED
	rollback.Reason = networkrollbacktypes.Reason_ERROR
	rollback.Message = "change does not exist"
	if err := r.networkRollbacks.Update(rollback); err != nil {
		return false, err
	}
	return true, nil
}

func (r *Reconciler) failNetworkChange(change *networkchangetypes.NetworkChange) (bool, error) {
	change.Status = networkchangetypes.Status_FAILED
	change.Reason = networkchangetypes.Reason_ERROR
	change.Message = "change cancelled"
	if err := r.networkChanges.Update(change); err != nil {
		return false, err
	}
	return true, nil
}

// canApplyNetworkRollback determines whether the rollback can be applied by checking whether all changes
// can be rolled back
func (r *Reconciler) canApplyNetworkRollback(rollback *networkrollbacktypes.NetworkRollback, networkChange *networkchangetypes.NetworkChange) (bool, error) {
	for _, change := range networkChange.Changes {
		succeeded, err := r.canApplyDeviceRollback(rollback, change)
		if !succeeded || err != nil {
			return succeeded, err
		}
	}
	return true, nil
}

// canApplyDeviceRollback determines whether the rollback can be applied for a specific device
func (r *Reconciler) canApplyDeviceRollback(rollback *networkrollbacktypes.NetworkRollback, deviceChange *devicechangetypes.Change) (bool, error) {
	ch := make(chan *devicechangetypes.Change)
	if err := r.deviceChanges.Replay(deviceChange.DeviceID, deviceChange.Index+1, ch); err != nil {
		return false, err
	}

	// Iterate through device changes that occurred after this change and determine whether any succeeded
	for change := range ch {
		if change.Status == devicechangetypes.Status_SUCCEEDED {
			return false, nil
		}
	}
	return true, nil
}

func (r *Reconciler) applyNetworkRollback(config *networkrollbacktypes.NetworkRollback) (bool, error) {
	config.Status = networkrollbacktypes.Status_APPLYING
	err := r.networkRollbacks.Update(config)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Reconciler) createDeviceRollbacks(config *networkrollbacktypes.NetworkRollback) (bool, error) {
	networkChange, err := r.networkChanges.Get(networkchangetypes.ID(config.ChangeID))
	if err != nil {
		return false, err
	}

	for _, change := range networkChange.Changes {
		deviceRollback, err := r.deviceRollbacks.Get(devicerollbacktypes.ID(change.ID))
		if err != nil {
			return false, err
		} else if deviceRollback == nil {
			deviceRollback = &devicerollbacktypes.Rollback{
				ID:       devicerollbacktypes.ID(change.ID),
				ChangeID: types.ID(config.ID),
				Status:   devicerollbacktypes.Status_PENDING,
			}
			err = r.deviceRollbacks.Create(deviceRollback)
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (r *Reconciler) applyDeviceChanges(config *networkrollbacktypes.NetworkRollback) (bool, error) {
	networkChange, err := r.networkChanges.Get(networkchangetypes.ID(config.ChangeID))
	if err != nil {
		return false, err
	}

	status := networkrollbacktypes.Status_SUCCEEDED
	reason := networkrollbacktypes.Reason_ERROR
	for _, change := range networkChange.Changes {
		deviceRollback, err := r.deviceRollbacks.Get(devicerollbacktypes.ID(change.ID))
		if err != nil {
			return false, err
		}

		// If the change is PENDING then change it to APPLYING
		if deviceRollback.Status == devicerollbacktypes.Status_PENDING {
			deviceRollback.Status = devicerollbacktypes.Status_APPLYING
			status = networkrollbacktypes.Status_APPLYING
			err = r.deviceRollbacks.Update(deviceRollback)
			if err != nil {
				return false, err
			}
		} else if deviceRollback.Status == devicerollbacktypes.Status_APPLYING {
			// If the change is APPLYING then ensure the network status is APPLYING
			status = networkrollbacktypes.Status_APPLYING
		} else if deviceRollback.Status == devicerollbacktypes.Status_FAILED {
			// If the change is FAILED then set the network to FAILED
			// If the change failure reason is UNAVAILABLE then all changes must be UNAVAILABLE, otherwise
			// the network must be failed with an ERROR.
			if status != networkrollbacktypes.Status_FAILED {
				switch deviceRollback.Reason {
				case devicerollbacktypes.Reason_ERROR:
					reason = networkrollbacktypes.Reason_ERROR
				case devicerollbacktypes.Reason_UNAVAILABLE:
					reason = networkrollbacktypes.Reason_UNAVAILABLE
				}
			} else if reason == networkrollbacktypes.Reason_UNAVAILABLE && deviceRollback.Reason == devicerollbacktypes.Reason_ERROR {
				reason = networkrollbacktypes.Reason_ERROR
			} else if reason == networkrollbacktypes.Reason_ERROR && deviceRollback.Reason == devicerollbacktypes.Reason_UNAVAILABLE {
				reason = networkrollbacktypes.Reason_ERROR
			}
			status = networkrollbacktypes.Status_FAILED
		}
	}

	// If the status has changed, update the network config
	if config.Status != status || config.Reason != reason {
		config.Status = status
		config.Reason = reason
		if err := r.networkRollbacks.Update(config); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *Reconciler) deleteDeviceRollbacks(config *networkrollbacktypes.NetworkRollback) (bool, error) {
	networkChange, err := r.networkChanges.Get(networkchangetypes.ID(config.ChangeID))
	if err != nil {
		return false, err
	}

	for _, change := range networkChange.Changes {
		deviceRollback, err := r.deviceRollbacks.Get(devicerollbacktypes.ID(change.ID))
		if err != nil {
			return false, err
		}
		err = r.deviceRollbacks.Delete(deviceRollback)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

var _ controller.Reconciler = &Reconciler{}
