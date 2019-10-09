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
	if config.Status.State == changetypes.State_PENDING {
		apply, err := r.canApplyNetworkChange(config)
		if err != nil {
			return false, err
		} else if !apply {
			return false, nil
		}
		return r.applyNetworkChange(config)
	}

	// If the change is the APPLYING state, ensure all device changes are in the APPLYING
	// state and check results
	if config.Status.State == changetypes.State_APPLYING {
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
		} else if change != nil {
			if change.Status.State == changetypes.State_PENDING || change.Status.State == changetypes.State_APPLYING {
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
	}
	return true, nil
}

func (r *Reconciler) applyNetworkChange(config *networktypes.NetworkChange) (bool, error) {
	config.Status.State = changetypes.State_APPLYING
	err := r.networkChanges.Update(config)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Reconciler) createDeviceChanges(config *networktypes.NetworkChange) (bool, error) {
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
	return true, nil
}

func (r *Reconciler) applyDeviceChanges(config *networktypes.NetworkChange) (bool, error) {
	state := changetypes.State_SUCCEEDED
	reason := changetypes.Reason_UNKNOWN
	for _, deviceChange := range config.Changes {
		change, err := r.deviceChanges.Get(deviceChange.ID)
		if err != nil {
			return false, err
		}

		// If the change is PENDING then change it to APPLYING
		if change.Status.State == changetypes.State_PENDING {
			change.Status.State = changetypes.State_APPLYING
			state = changetypes.State_APPLYING
			err = r.deviceChanges.Update(change)
			if err != nil {
				return false, err
			}
		} else if change.Status.State == changetypes.State_APPLYING {
			// If the change is APPLYING then ensure the network state is APPLYING
			state = changetypes.State_APPLYING
		} else if change.Status.State == changetypes.State_FAILED {
			// If the change is FAILED then set the network to FAILED
			// If the change failure reason is UNAVAILABLE then all changes must be UNAVAILABLE, otherwise
			// the network must be failed with an ERROR.
			if state != changetypes.State_FAILED {
				switch change.Status.Reason {
				case changetypes.Reason_ERROR:
					reason = changetypes.Reason_ERROR
				case changetypes.Reason_UNAVAILABLE:
					reason = changetypes.Reason_UNAVAILABLE
				}
			} else if reason == changetypes.Reason_UNAVAILABLE && change.Status.Reason == changetypes.Reason_ERROR {
				reason = changetypes.Reason_ERROR
			} else if reason == changetypes.Reason_UNKNOWN && change.Status.Reason == changetypes.Reason_UNAVAILABLE {
				reason = changetypes.Reason_ERROR
			}
			state = changetypes.State_FAILED
		}
	}

	// If the state has changed, update the network config
	if config.Status.State != state || config.Status.Reason != reason {
		config.Status.State = state
		config.Status.Reason = reason
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
