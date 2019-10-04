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
	changestore "github.com/onosproject/onos-config/pkg/store/device/change"
	snapshotstore "github.com/onosproject/onos-config/pkg/store/device/snapshot"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/types"
	changetype "github.com/onosproject/onos-config/pkg/types/device/change"
	snapshottype "github.com/onosproject/onos-config/pkg/types/device/snapshot"
)

// NewController returns a new network controller
func NewController(mastership mastershipstore.Store, changes changestore.Store, snapshots snapshotstore.Store) *controller.Controller {
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
		changes:   changes,
	})
	return c
}

// Reconciler is the change reconciler
type Reconciler struct {
	snapshots snapshotstore.Store
	changes   changestore.Store
}

// Reconcile reconciles a change
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	request, err := r.snapshots.Get(snapshottype.ID(id))
	if err != nil {
		return false, err
	}

	// If the request is APPLYING, take the snapshot
	if request.Status == snapshottype.Status_APPLYING {
		return r.takeSnapshot(request)
	}
	return true, nil
}

// takeSnapshot takes a snapshot
func (r *Reconciler) takeSnapshot(request *snapshottype.DeviceSnapshot) (bool, error) {
	index := changetype.Index(0)

	// Determine the starting index for the snapshot
	currentSnapshot, err := r.snapshots.Load(request.DeviceID)
	if err != nil {
		return false, err
	} else if currentSnapshot != nil {
		index = changetype.Index(currentSnapshot.Index + 1)
	}
	snapshotIndex := snapshottype.Index(index)

	// If the snapshot has not been stored, take and store the snapshot
	if currentSnapshot != nil && currentSnapshot.SnapshotID != request.ID {
		// Get the index up to which to iterate
		lastIndex, err := r.changes.LastIndex(request.DeviceID)
		if err != nil {
			return false, err
		}

		// Loop through device changes and populate the change values state from all changes
		snapshotValues := make(map[string]*changetype.Value)
		for i := index; i < changetype.Index(lastIndex); i++ {
			change, err := r.changes.Get(i.GetID(request.DeviceID))
			if err != nil {
				return false, err
			}

			// If the change occurred after the snapshot time, stop processing snapshots
			if change.Created.UnixNano() < request.Timestamp.UnixNano() {
				break
			}

			for _, value := range change.Values {
				snapshotValues[value.Path] = value
			}
			snapshotIndex = snapshottype.Index(change.Index)
		}

		// Convert the snapshot values to a list for storage
		values := make([]*changetype.Value, 0, len(snapshotValues))
		for _, value := range snapshotValues {
			values = append(values, value)
		}

		// Store the snapshot
		snapshot := &snapshottype.Snapshot{
			ID:        snapshottype.ID(request.DeviceID),
			DeviceID:  request.DeviceID,
			Index:     snapshotIndex,
			Timestamp: request.Timestamp,
			Values:    values,
		}
		if err := r.snapshots.Store(snapshot); err != nil {
			return false, err
		}
	} else {
		snapshotIndex = currentSnapshot.Index
	}

	// Update the snapshot state
	request.Status = snapshottype.Status_SUCCEEDED
	if err := r.snapshots.Update(request); err != nil {
		return false, err
	}

	// Once the snapshot is successful, delete changes
	for i := index; i <= changetype.Index(snapshotIndex); i++ {
		change, err := r.changes.Get(i.GetID(request.DeviceID))
		if err != nil {
			return false, err
		} else if change != nil {
			if err := r.changes.Delete(change); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

var _ controller.Reconciler = &Reconciler{}
