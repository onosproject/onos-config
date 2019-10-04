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

package change

import (
	"github.com/onosproject/onos-config/pkg/controller"
	devicecontroller "github.com/onosproject/onos-config/pkg/controller/device/change"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/device/change"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/network/change"
	"github.com/onosproject/onos-config/pkg/types"
	devicetypes "github.com/onosproject/onos-config/pkg/types/device/change"
	networktypes "github.com/onosproject/onos-config/pkg/types/network/change"
)

// NewController returns a new config controller
func NewController(leadership leadershipstore.Store, networkChanges networkchangestore.Store, deviceChanges devicechangestore.Store) *controller.Controller {
	c := controller.NewController()
	c.Activate(&controller.LeadershipActivator{
		Store: leadership,
	})
	c.Watch(&Watcher{
		Store: networkChanges,
	})
	c.Watch(&devicecontroller.OwnerWatcher{
		Store: deviceChanges,
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
	config, err := r.networkChanges.Get(networktypes.ID(id))
	if err != nil {
		return false, err
	}

	// If the revision number is 0, the change has been removed
	if config.Revision == 0 {
		return r.deleteDeviceChanges(config)
	}

	// Ensure changes have been created in the device change store
	succeeded, err := r.createDeviceChanges(config)
	if !succeeded || err != nil {
		return succeeded, err
	}

	// If the change is in the PENDING state, check if it can be applied and apply it if so.
	if config.Status == networktypes.Status_PENDING {
		apply, err := r.canApplyNetworkChange(config)
		if err != nil {
			return false, err
		} else if !apply {
			return true, nil
		}
		return r.applyNetworkChange(config)
	}

	// If the change is the APPLYING state, ensure all device changes are in the APPLYING
	// state and check results
	if config.Status == networktypes.Status_APPLYING {
		return r.applyDeviceChanges(config)
	}
	return true, nil
}

func (r *Reconciler) canApplyNetworkChange(config *networktypes.NetworkChange) (bool, error) {
	sequential := true
	for index := r.changeIndex; index < config.Index; index++ {
		change, err := r.networkChanges.GetByIndex(index)
		if err != nil {
			return false, err
		} else if change.Status == networktypes.Status_PENDING || change.Status == networktypes.Status_APPLYING {
			if intersecting(config, change) {
				return false, nil
			}
			sequential = false
		} else {
			if sequential {
				r.changeIndex++
			}
		}
	}
	return true, nil
}

func (r *Reconciler) applyNetworkChange(config *networktypes.NetworkChange) (bool, error) {
	config.Status = networktypes.Status_APPLYING
	err := r.networkChanges.Update(config)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Reconciler) createDeviceChanges(config *networktypes.NetworkChange) (bool, error) {
	updated := false
	for _, change := range config.Changes {
		// If the change index is not set, the change needs to be stored
		if change.Index == 0 {
			change.Status = devicetypes.Status_PENDING
			if err := r.deviceChanges.Create(change); err != nil {
				return false, err
			}
			updated = true
		}
	}

	// Update the network change if device changes have been updated
	// TODO: There's a race in this controller wherein a device change could be created
	// without the network change every being updated. To avoid this, the network change
	// should be updated with a new device change ID prior to the device change being created.
	if updated {
		if err := r.networkChanges.Update(config); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *Reconciler) applyDeviceChanges(config *networktypes.NetworkChange) (bool, error) {
	status := networktypes.Status_SUCCEEDED
	reason := networktypes.Reason_ERROR
	for _, deviceChange := range config.Changes {
		change, err := r.deviceChanges.Get(deviceChange.ID)
		if err != nil {
			return false, err
		}

		// If the change is PENDING then change it to APPLYING
		if change.Status == devicetypes.Status_PENDING {
			change.Status = devicetypes.Status_APPLYING
			status = networktypes.Status_APPLYING
			err = r.deviceChanges.Update(change)
			if err != nil {
				return false, err
			}
		} else if change.Status == devicetypes.Status_APPLYING {
			// If the change is APPLYING then ensure the network status is APPLYING
			status = networktypes.Status_APPLYING
		} else if change.Status == devicetypes.Status_FAILED {
			// If the change is FAILED then set the network to FAILED
			// If the change failure reason is UNAVAILABLE then all changes must be UNAVAILABLE, otherwise
			// the network must be failed with an ERROR.
			if status != networktypes.Status_FAILED {
				switch change.Reason {
				case devicetypes.Reason_ERROR:
					reason = networktypes.Reason_ERROR
				case devicetypes.Reason_UNAVAILABLE:
					reason = networktypes.Reason_UNAVAILABLE
				}
			} else if reason == networktypes.Reason_UNAVAILABLE && change.Reason == devicetypes.Reason_ERROR {
				reason = networktypes.Reason_ERROR
			} else if reason == networktypes.Reason_ERROR && change.Reason == devicetypes.Reason_UNAVAILABLE {
				reason = networktypes.Reason_ERROR
			}
			status = networktypes.Status_FAILED
		}
	}

	// If the status has changed, update the network config
	if config.Status != status || config.Reason != reason {
		config.Status = status
		config.Reason = reason
		if err := r.networkChanges.Update(config); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *Reconciler) deleteDeviceChanges(config *networktypes.NetworkChange) (bool, error) {
	for _, change := range config.Changes {
		deviceChange, err := r.deviceChanges.Get(change.ID)
		if err != nil {
			return false, err
		}
		err = r.deviceChanges.Delete(deviceChange)
		if err != nil {
			return false, err
		}
	}
	return true, nil
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
