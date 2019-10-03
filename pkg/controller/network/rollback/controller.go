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
	devicerollbackstore "github.com/onosproject/onos-config/pkg/store/device/rollback"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	networkrollbackstore "github.com/onosproject/onos-config/pkg/store/network/rollback"
	"github.com/onosproject/onos-config/pkg/types"
	networkrollbacktypes "github.com/onosproject/onos-config/pkg/types/network/rollback"
)

// NewController returns a new config controller
func NewController(leadership leadershipstore.Store, networkChanges networkrollbackstore.Store, deviceChanges devicerollbackstore.Store) *controller.Controller {
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
	networkChanges networkrollbackstore.Store
	deviceChanges  devicerollbackstore.Store
}

// Reconcile reconciles the state of a network configuration
func (c *Reconciler) Reconcile(id types.ID) (bool, error) {
	_, err := c.networkChanges.Get(networkrollbacktypes.ID(id))
	if err != nil {
		return false, err
	}
	// TODO
	return true, nil
}

var _ controller.Reconciler = &Reconciler{}
