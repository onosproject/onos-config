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
}

// Reconcile reconciles the state of a network configuration
func (c *Reconciler) Reconcile(id types.ID) (bool, error) {
	config, err := c.networkChanges.Get(networktypes.ID(id))
	if err != nil {
		return false, err
	}

	// TODO: Changes should be removed when a config is removed

	// Ensure changes are created in the change store for each device in the network config.
	for _, configChange := range config.Changes {
		change, err := c.deviceChanges.Get(config.GetChangeID(configChange.DeviceID))
		if err != nil {
			return false, err
		}

		// If the change has not been created, create it.
		if change == nil {
			change = configChange
			change.Status = devicetypes.Status_PENDING
			err = c.deviceChanges.Create(change)
			if err != nil {
				return false, err
			}
		}
	}

	// If the config status is APPLYING, ensure all device changes are in the APPLYING state.
	if config.Status == networktypes.Status_APPLYING {
		status := networktypes.Status_SUCCEEDED
		reason := networktypes.Reason_ERROR
		for _, id := range config.GetChangeIDs() {
			change, err := c.deviceChanges.Get(id)
			if err != nil {
				return false, err
			}

			// If the change is PENDING then change it to APPLYING
			if change.Status == devicetypes.Status_PENDING {
				change.Status = devicetypes.Status_APPLYING
				status = networktypes.Status_APPLYING
				err = c.deviceChanges.Update(change)
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
			if err := c.networkChanges.Update(config); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

var _ controller.Reconciler = &Reconciler{}
