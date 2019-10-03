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

package snapshot

import (
	"github.com/onosproject/onos-config/pkg/controller"
	snapshotstore "github.com/onosproject/onos-config/pkg/store/device/snapshot"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/types"
	snapshottype "github.com/onosproject/onos-config/pkg/types/device/snapshot"
)

// NewController returns a new network controller
func NewController(mastership mastershipstore.Store, snapshots snapshotstore.Store) *controller.Controller {
	c := controller.NewController()
	c.Filter(&controller.MastershipFilter{
		Store: mastership,
	})
	c.Partition(&Partitioner{})
	c.Watch(&Watcher{
		Store: snapshots,
	})
	c.Reconcile(&Reconciler{
		snapshots: snapshots,
	})
	return c
}

// Reconciler is the change reconciler
type Reconciler struct {
	snapshots snapshotstore.Store
}

// Reconcile reconciles a change
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	request, err := r.snapshots.Get(snapshottype.ID(id))
	if err != nil {
		return false, err
	}

	// If the request is APPLYING, take the snapshot
	if request.Status == snapshottype.Status_APPLYING {
		if err := r.takeSnapshot(request); err != nil {
			return false, err
		}
	}
	return true, nil
}

// takeSnapshot takes a snapshot
func (r *Reconciler) takeSnapshot(request *snapshottype.DeviceSnapshot) error {
	// TODO
	snapshot := &snapshottype.Snapshot{}
	return r.snapshots.Store(snapshot)
}

var _ controller.Reconciler = &Reconciler{}
