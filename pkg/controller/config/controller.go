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

package config

import (
	"github.com/onosproject/onos-config/pkg/controller"
	"github.com/onosproject/onos-config/pkg/controller/change"
	changestore "github.com/onosproject/onos-config/pkg/store/change"
	configstore "github.com/onosproject/onos-config/pkg/store/config"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/types"
	changetype "github.com/onosproject/onos-config/pkg/types/change"
	configtype "github.com/onosproject/onos-config/pkg/types/config"
)

// NewController returns a new config controller
func NewController(leadership leadershipstore.Store, configs configstore.Store, changes changestore.Store) *controller.Controller {
	c := controller.NewController()
	c.Activate(&controller.LeadershipActivator{
		Store: leadership,
	})
	c.Watch(&Watcher{
		Store: configs,
	})
	c.Watch(&change.OwnerWatcher{
		Store: changes,
	})
	c.Reconcile(&Reconciler{
		configs: configs,
		changes: changes,
	})
	return c
}

// Reconciler is a config reconciler
type Reconciler struct {
	*controller.Controller
	configs configstore.Store
	changes changestore.Store
}

// Reconcile reconciles the state of a network configuration
func (c *Reconciler) Reconcile(id types.ID) (bool, error) {
	config, err := c.configs.Get(configtype.ID(id))
	if err != nil {
		return false, err
	}

	// TODO: Changes should be removed when a config is removed

	// Ensure changes are created in the change store for each device in the network config.
	for _, configChange := range config.Changes {
		change, err := c.changes.Get(config.GetChangeID(configChange.DeviceID))
		if err != nil {
			return false, err
		}

		// If the change has not been created, create it.
		if change == nil {
			change = configChange
			change.Status = changetype.Status_PENDING
			err = c.changes.Create(change)
			if err != nil {
				return false, err
			}
		}
	}

	// If the config status is APPLYING, ensure all device changes are in the APPLYING state.
	if config.Status == configtype.Status_APPLYING {
		status := configtype.Status_SUCCEEDED
		reason := configtype.Reason_ERROR
		for _, id := range config.GetChangeIDs() {
			change, err := c.changes.Get(id)
			if err != nil {
				return false, err
			}

			// If the change is PENDING then change it to APPLYING
			if change.Status == changetype.Status_PENDING {
				change.Status = changetype.Status_APPLYING
				status = configtype.Status_APPLYING
				err = c.changes.Update(change)
				if err != nil {
					return false, err
				}
			} else if change.Status == changetype.Status_APPLYING {
				// If the change is APPLYING then ensure the network status is APPLYING
				status = configtype.Status_APPLYING
			} else if change.Status == changetype.Status_FAILED {
				// If the change is FAILED then set the network to FAILED
				// If the change failure reason is UNAVAILABLE then all changes must be UNAVAILABLE, otherwise
				// the network must be failed with an ERROR.
				if status != configtype.Status_FAILED {
					switch change.Reason {
					case changetype.Reason_ERROR:
						reason = configtype.Reason_ERROR
					case changetype.Reason_UNAVAILABLE:
						reason = configtype.Reason_UNAVAILABLE
					}
				} else if reason == configtype.Reason_UNAVAILABLE && change.Reason == changetype.Reason_ERROR {
					reason = configtype.Reason_ERROR
				} else if reason == configtype.Reason_ERROR && change.Reason == changetype.Reason_UNAVAILABLE {
					reason = configtype.Reason_ERROR
				}
				status = configtype.Status_FAILED
			}
		}

		// If the status has changed, update the network config
		if config.Status != status || config.Reason != reason {
			config.Status = status
			config.Reason = reason
			if err := c.configs.Update(config); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

var _ controller.Reconciler = &Reconciler{}
