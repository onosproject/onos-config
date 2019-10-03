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
	rollbackstore "github.com/onosproject/onos-config/pkg/store/device/rollback"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/types"
	rollbacktype "github.com/onosproject/onos-config/pkg/types/device/rollback"
)

// NewController returns a new network controller
func NewController(mastership mastershipstore.Store, changes rollbackstore.Store) *controller.Controller {
	c := controller.NewController()
	c.Filter(&controller.MastershipFilter{
		Store: mastership,
	})
	c.Partition(&Partitioner{})
	c.Watch(&Watcher{
		Store: changes,
	})
	c.Reconcile(&Reconciler{
		changes: changes,
	})
	return c
}

// Reconciler is the change reconciler
type Reconciler struct {
	changes rollbackstore.Store
}

// Reconcile reconciles a change
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	change, err := r.changes.Get(rollbacktype.ID(id))
	if err != nil {
		return false, err
	}

	// If the change is in the applying state, apply the change.
	if change.Status == rollbacktype.Status_APPLYING {
		if err := r.applyChange(change); err != nil {
			return false, err
		}
	}
	return true, nil
}

// applyChange applies the given change to the device
func (r *Reconciler) applyChange(change *rollbacktype.Rollback) error {
	change.Status = rollbacktype.Status_FAILED
	change.Reason = rollbacktype.Reason_UNAVAILABLE
	return r.changes.Update(change)
}

var _ controller.Reconciler = &Reconciler{}
