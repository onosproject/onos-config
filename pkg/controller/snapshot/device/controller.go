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

package device

import (
	"github.com/onosproject/onos-config/pkg/controller"
	changestore "github.com/onosproject/onos-config/pkg/store/change/device"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	snapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/onosproject/onos-config/pkg/types"
	changetype "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetype "github.com/onosproject/onos-config/pkg/types/change/device"
	snaptype "github.com/onosproject/onos-config/pkg/types/snapshot"
	devicesnaptype "github.com/onosproject/onos-config/pkg/types/snapshot/device"
	deviceservice "github.com/onosproject/onos-topo/pkg/northbound/device"
)

// NewController returns a new network controller
func NewController(mastership mastershipstore.Store, changes changestore.Store, snapshots snapstore.Store) *controller.Controller {
	c := controller.NewController()
	c.Filter(&controller.MastershipFilter{
		Store: mastership,
		Resolver: &Resolver{
			snapshots: snapshots,
		},
	})
	c.Partition(&Partitioner{})
	c.Watch(&Watcher{
		Store: snapshots,
	})
	c.Reconcile(&Reconciler{
		changes:   changes,
		snapshots: snapshots,
	})
	return c
}

// Resolver is a DeviceResolver that resolves device IDs from device change IDs
type Resolver struct {
	snapshots snapstore.Store
}

// Resolve resolves a device ID from a device change ID
func (r *Resolver) Resolve(id types.ID) (deviceservice.ID, error) {
	snapshot, err := r.snapshots.Get(devicesnaptype.ID(id))
	if err != nil {
		return "", err
	}
	return snapshot.DeviceID, nil
}

// Reconciler is a device change reconciler
type Reconciler struct {
	changes   changestore.Store
	snapshots snapstore.Store
}

// Reconcile reconciles the state of a device change
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	// Get the snapshot from the store
	snapshot, err := r.snapshots.Get(devicesnaptype.ID(id))
	if err != nil {
		return false, err
	}

	// The device controller only needs to handle snapshots in the RUNNING state
	if snapshot == nil || snapshot.Status.State != snaptype.State_RUNNING {
		return true, nil
	}
	return r.reconcileSnapshot(snapshot)
}

// reconcileSnapshot reconciles a snapshot in the RUNNING state
func (r *Reconciler) reconcileSnapshot(deviceSnapshot *devicesnaptype.DeviceSnapshot) (bool, error) {
	// Check if a snapshot already exists for the device
	prevSnapshot, err := r.snapshots.Load(devicesnaptype.ID(deviceSnapshot.DeviceID))
	if err != nil {
		return false, err
	}

	// If a snapshot exists and was taken after the requested time for this snapshot, complete this snapshot
	if prevSnapshot != nil && prevSnapshot.Timestamp.After(deviceSnapshot.Timestamp) {
		deviceSnapshot.Status.State = snaptype.State_COMPLETE
		if err := r.snapshots.Update(deviceSnapshot); err != nil {
			return false, err
		}
		return true, nil
	}

	// Take a snapshot of the device changes up to the requested timestamp
	index, err := r.takeSnapshot(deviceSnapshot, prevSnapshot)
	if err != nil {
		return false, err
	}

	// Once the snapshot has been taken, remove changes that occurred prior to the snapshot
	if err := r.clearChanges(deviceSnapshot, index); err != nil {
		return false, err
	}

	// Finally, mark the snapshot complete
	deviceSnapshot.Status.State = snaptype.State_COMPLETE
	if err := r.snapshots.Update(deviceSnapshot); err != nil {
		return false, err
	}
	return true, nil
}

// takeSnapshot takes and stores a snapshot of the device
func (r *Reconciler) takeSnapshot(deviceSnapshot *devicesnaptype.DeviceSnapshot, prevSnapshot *devicesnaptype.Snapshot) (devicechangetype.Index, error) {
	// If the previous snapshot ID matches the device snapshot request ID, return the previous snapshot
	if prevSnapshot != nil && prevSnapshot.SnapshotID == deviceSnapshot.ID {
		return devicechangetype.Index(prevSnapshot.Index), nil
	}

	// Record the state of the new snapshot in a map keyed by paths in the device
	state := make(map[string]*devicechangetype.Value)

	// Get the index at which to start the snapshot
	var prevSnapshotIndex devicechangetype.Index

	// If a prior snapshot was found, initialize the state from that snapshot and replay changes
	// starting after the snapshot index
	if prevSnapshot != nil {
		for _, value := range prevSnapshot.Values {
			state[value.Path] = value
		}
		prevSnapshotIndex = devicechangetype.Index(prevSnapshot.Index)
	}

	// Get the last index stored for the device
	lastIndex, err := r.changes.LastIndex(deviceSnapshot.DeviceID)
	if err != nil {
		return 0, err
	}

	// Starting at the index of the last snapshot, replay completed changes to populate the state of the new snapshot
	snapshotIndex := prevSnapshotIndex
	for index := prevSnapshotIndex + 1; index <= lastIndex; index++ {
		deviceChange, err := r.changes.GetByIndex(deviceSnapshot.DeviceID, index)
		if err != nil {
			return 0, err
		} else if deviceChange != nil {
			if deviceChange.Status.Phase == changetype.Phase_CHANGE && deviceChange.Status.State == changetype.State_COMPLETE {
				for _, value := range deviceChange.Values {
					if !value.Removed {
						state[value.Path] = value
					} else {
						delete(state, value.Path)
					}
				}
				snapshotIndex = index
			} else {
				break
			}
		}
	}

	// If the snapshot has not recorded any changes, skip it and complete the snapshot
	if snapshotIndex <= prevSnapshotIndex {
		return devicechangetype.Index(prevSnapshot.Index), nil
	}

	// If changes have been recorded, create and store the snapshot
	snapshotValues := make([]*devicechangetype.Value, 0, len(state))
	for _, value := range state {
		snapshotValues = append(snapshotValues, value)
	}

	snapshot := &devicesnaptype.Snapshot{
		ID:         devicesnaptype.ID(deviceSnapshot.DeviceID),
		DeviceID:   deviceSnapshot.DeviceID,
		SnapshotID: deviceSnapshot.ID,
		Index:      devicesnaptype.Index(snapshotIndex),
		Values:     snapshotValues,
		Timestamp:  deviceSnapshot.Timestamp,
	}
	if err := r.snapshots.Store(snapshot); err != nil {
		return 0, err
	}
	return devicechangetype.Index(snapshot.Index), nil
}

// clearChanges removes all changes up to the given index from the device change store
func (r *Reconciler) clearChanges(deviceSnapshot *devicesnaptype.DeviceSnapshot, index devicechangetype.Index) error {
	ch := make(chan *devicechangetype.Change)
	if err := r.changes.List(deviceSnapshot.DeviceID, ch); err != nil {
		return err
	}

	for change := range ch {
		if change.Index <= index {
			if err := r.changes.Delete(change); err != nil {
				return err
			}
		}
	}
	return nil
}

var _ controller.Reconciler = &Reconciler{}
