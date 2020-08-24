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
	"strings"

	"github.com/onosproject/onos-config/api/types"
	changetype "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	snaptype "github.com/onosproject/onos-config/api/types/snapshot"
	devicesnapshot "github.com/onosproject/onos-config/api/types/snapshot/device"
	"github.com/onosproject/onos-config/pkg/controller"
	changestore "github.com/onosproject/onos-config/pkg/store/change/device"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	snapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	topodevice "github.com/onosproject/onos-topo/api/device"
)

var log = logging.GetLogger("controller", "snapshot", "device")

// NewController returns a new device snapshot controller
func NewController(mastership mastershipstore.Store, changes changestore.Store, snapshots snapstore.Store) *controller.Controller {
	c := controller.NewController("DeviceSnapshot")
	c.Filter(&controller.MastershipFilter{
		Store:    mastership,
		Resolver: &Resolver{snapshots: snapshots},
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

// Resolver is a DeviceResolver that resolves device IDs from device snapshot IDs
type Resolver struct {
	snapshots snapstore.Store
}

// Resolve resolves a device ID from a device snapshot ID
func (r *Resolver) Resolve(id types.ID) (topodevice.ID, error) {
	return topodevice.ID(devicesnapshot.ID(id).GetDeviceID()), nil
}

// Reconciler is a device snapshot reconciler
type Reconciler struct {
	changes   changestore.Store
	snapshots snapstore.Store
}

// Reconcile reconciles the state of a device snapshot
func (r *Reconciler) Reconcile(id types.ID) (controller.Result, error) {
	// Get the snapshot from the store
	deviceSnapshot, err := r.snapshots.Get(devicesnapshot.ID(id))
	if err != nil {
		return controller.Result{}, err
	}

	// The device controller only needs to handle snapshots in the RUNNING state
	if deviceSnapshot == nil || deviceSnapshot.Status.State != snaptype.State_RUNNING {
		return controller.Result{}, nil
	}

	// Handle the snapshot for each phase
	switch deviceSnapshot.Status.Phase {
	case snaptype.Phase_MARK:
		return r.reconcileMark(deviceSnapshot)
	case snaptype.Phase_DELETE:
		return r.reconcileDelete(deviceSnapshot)
	}
	return controller.Result{}, nil
}

// reconcileMark reconciles a snapshot in the MARK phase
func (r *Reconciler) reconcileMark(deviceSnapshot *devicesnapshot.DeviceSnapshot) (controller.Result, error) {
	// Get the previous snapshot if any
	var prevIndex devicechange.Index
	prevSnapshot, err := r.snapshots.Load(deviceSnapshot.GetVersionedDeviceID())
	if err != nil {
		return controller.Result{}, err
	} else if prevSnapshot != nil {
		prevIndex = prevSnapshot.ChangeIndex
	}

	// Create a map to track the current state of the device
	state := make(map[string]*devicechange.PathValue)

	// Initialize the state map from the previous snapshot if available
	if prevSnapshot != nil {
		for _, value := range prevSnapshot.Values {
			state[value.Path] = value
		}
	}

	// List the changes for the device
	changes := make(chan *devicechange.DeviceChange)
	ctx, err := r.changes.List(deviceSnapshot.GetVersionedDeviceID(), changes)
	if err != nil {
		return controller.Result{}, err
	}
	defer ctx.Close()

	// Iterate through changes and populate the snapshot
	log.Infof("Taking snapshot of device %s", deviceSnapshot.DeviceID)
	var snapshotIndex = prevIndex
	for change := range changes {
		// If the change index is included in the last snapshot, ignore the change
		if change.Index <= snapshotIndex {
			continue
		}

		// If the change is from a NetworkChange greater than the highest change to be snapshotted, break out of the loop
		if change.NetworkChange.Index > deviceSnapshot.MaxNetworkChangeIndex {
			break
		}

		// If the change is within the window to be snapshotted and the change has not been rolled back, record it
		if change.Status.Phase == changetype.Phase_CHANGE {
			for _, value := range change.Change.Values {
				if value.Removed {
					// remove any previous paths that started with this deleted path
					// including those from the previous snapshot
					for p := range state {
						if strings.HasPrefix(p, value.Path) {
							delete(state, p)
						}
					}
				} else {
					state[value.Path] = &devicechange.PathValue{
						Path:  value.GetPath(),
						Value: value.GetValue(),
					}
				}
			}
		}
		snapshotIndex = change.Index
	}

	// If the snapshot index is greater than the previous snapshot index, store the snapshot
	if snapshotIndex > prevIndex {
		values := make([]*devicechange.PathValue, 0, len(state))
		for _, value := range state {
			values = append(values, value)
		}

		snapshot := &devicesnapshot.Snapshot{
			ID:            devicesnapshot.ID(deviceSnapshot.DeviceID),
			DeviceID:      deviceSnapshot.DeviceID,
			DeviceVersion: deviceSnapshot.DeviceVersion,
			DeviceType:    deviceSnapshot.DeviceType,
			SnapshotID:    deviceSnapshot.ID,
			ChangeIndex:   snapshotIndex,
			Values:        values,
		}
		log.Infof("Storing Snapshot %v", snapshot)
		if err := r.snapshots.Store(snapshot); err != nil {
			return controller.Result{}, err
		}
	}

	// Complete the snapshot MARK phase
	deviceSnapshot.Status.State = snaptype.State_COMPLETE
	log.Infof("Completing DeviceSnapshot %v", deviceSnapshot)
	if err := r.snapshots.Update(deviceSnapshot); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// reconcileDelete reconciles a snapshot in the DELETE phase
func (r *Reconciler) reconcileDelete(deviceSnapshot *devicesnapshot.DeviceSnapshot) (controller.Result, error) {
	// Load the current snapshot
	snapshot, err := r.snapshots.Load(deviceSnapshot.GetVersionedDeviceID())
	if err != nil {
		return controller.Result{}, err
	} else if snapshot == nil {
		deviceSnapshot.Status.State = snaptype.State_COMPLETE
		log.Infof("Completing DeviceSnapshot %v", deviceSnapshot)
		if err := r.snapshots.Update(deviceSnapshot); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// List the changes for the device
	changes := make(chan *devicechange.DeviceChange)
	ctx, err := r.changes.List(deviceSnapshot.GetVersionedDeviceID(), changes)
	if err != nil {
		return controller.Result{}, err
	}
	defer ctx.Close()

	// Iterate through changes up to the current snapshot index and delete changes
	count := 0
	for change := range changes {
		if change.Index <= snapshot.ChangeIndex {
			if err := r.changes.Delete(change); err != nil {
				return controller.Result{}, err
			}
			count++
		}
	}
	log.Infof("Deleted %d DeviceChanges for device %s", count, deviceSnapshot.DeviceID)

	// Finally, complete the phase
	deviceSnapshot.Status.State = snaptype.State_COMPLETE
	log.Infof("Completing DeviceSnapshot %v", deviceSnapshot)
	if err := r.snapshots.Update(deviceSnapshot); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

var _ controller.Reconciler = &Reconciler{}
