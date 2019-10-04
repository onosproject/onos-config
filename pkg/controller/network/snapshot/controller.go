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
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	devicesnapshotstore "github.com/onosproject/onos-config/pkg/store/device/snapshot"
	leadershipstore "github.com/onosproject/onos-config/pkg/store/leadership"
	networksnapshotstore "github.com/onosproject/onos-config/pkg/store/network/snapshot"
	"github.com/onosproject/onos-config/pkg/types"
	devicesnapshottype "github.com/onosproject/onos-config/pkg/types/device/snapshot"
	networksnapshottype "github.com/onosproject/onos-config/pkg/types/network/snapshot"
)

// NewController returns a new network snapshot request controller
func NewController(leadership leadershipstore.Store, devices devicestore.Store, networkRequests networksnapshotstore.Store, deviceRequests devicesnapshotstore.Store) *controller.Controller {
	c := controller.NewController()
	c.Activate(&controller.LeadershipActivator{
		Store: leadership,
	})
	c.Watch(&Watcher{
		Store: networkRequests,
	})
	c.Reconcile(&Reconciler{
		devices:         devices,
		networkRequests: networkRequests,
		deviceRequests:  deviceRequests,
	})
	return c
}

// Reconciler is the network snapshot request reconciler
type Reconciler struct {
	devices         devicestore.Store
	networkRequests networksnapshotstore.Store
	deviceRequests  devicesnapshotstore.Store
	snapshotIndex   networksnapshottype.Index
}

// Reconcile reconciles the state of a network snapshot
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	config, err := r.networkRequests.Get(networksnapshottype.ID(id))
	if err != nil {
		return false, err
	}

	// If the revision number is 0, the change has been removed
	if config.Revision == 0 {
		return r.deleteDeviceSnapshots(config)
	}

	// Ensure changes have been created in the device change store
	succeeded, err := r.createDeviceSnapshots(config)
	if !succeeded || err != nil {
		return succeeded, err
	}

	// If the change is in the PENDING state, check if it can be applied and apply it if so.
	if config.Status == networksnapshottype.Status_PENDING {
		apply, err := r.canApplyNetworkSnapshot(config)
		if err != nil {
			return false, err
		} else if !apply {
			return true, nil
		}
		return r.applyNetworkSnapshot(config)
	}

	// If the change is the APPLYING state, ensure all device changes are in the APPLYING
	// state and check results
	if config.Status == networksnapshottype.Status_APPLYING {
		return r.applyDeviceSnapshots(config)
	}

	// If the snapshot is SUCCEEDED or FAILED, apply the next snapshot in the queue
	if config.Status == networksnapshottype.Status_SUCCEEDED || config.Status == networksnapshottype.Status_FAILED {
		return r.applyNextNetworkSnapshot(config)
	}
	return true, nil
}

func (r *Reconciler) canApplyNetworkSnapshot(config *networksnapshottype.NetworkSnapshot) (bool, error) {
	for index := r.snapshotIndex; index < config.Index; index++ {
		snapshot, err := r.networkRequests.GetByIndex(index)
		if err != nil {
			return false, err
		} else if snapshot.Status == networksnapshottype.Status_PENDING || snapshot.Status == networksnapshottype.Status_APPLYING {
			return false, nil
		}
	}
	return true, nil
}

func (r *Reconciler) applyNetworkSnapshot(config *networksnapshottype.NetworkSnapshot) (bool, error) {
	config.Status = networksnapshottype.Status_APPLYING
	err := r.networkRequests.Update(config)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Reconciler) createDeviceSnapshots(config *networksnapshottype.NetworkSnapshot) (bool, error) {
	for _, deviceID := range config.Devices {
		snapshotID := config.GetDeviceSnapshotID(deviceID)
		deviceSnapshot, err := r.deviceRequests.Get(snapshotID)
		if err != nil {
			return false, err
		} else if deviceSnapshot == nil {
			deviceSnapshot = &devicesnapshottype.DeviceSnapshot{
				ID:        snapshotID,
				DeviceID:  deviceID,
				Timestamp: config.Timestamp,
			}
			if err := r.deviceRequests.Create(deviceSnapshot); err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

func (r *Reconciler) applyDeviceSnapshots(config *networksnapshottype.NetworkSnapshot) (bool, error) {
	status := networksnapshottype.Status_SUCCEEDED
	reason := networksnapshottype.Reason_ERROR
	for _, snapshotID := range config.GetDeviceSnapshotIDs() {
		deviceSnapshot, err := r.deviceRequests.Get(snapshotID)
		if err != nil {
			return false, err
		}

		// If the snapshot is PENDING then snapshot it to APPLYING
		if deviceSnapshot.Status == devicesnapshottype.Status_PENDING {
			deviceSnapshot.Status = devicesnapshottype.Status_APPLYING
			status = networksnapshottype.Status_APPLYING
			err = r.deviceRequests.Update(deviceSnapshot)
			if err != nil {
				return false, err
			}
		} else if deviceSnapshot.Status == devicesnapshottype.Status_APPLYING {
			// If the snapshot is APPLYING then ensure the network status is APPLYING
			status = networksnapshottype.Status_APPLYING
		} else if deviceSnapshot.Status == devicesnapshottype.Status_FAILED {
			// If the snapshot is FAILED then set the network to FAILED
			// If the snapshot failure reason is UNAVAILABLE then all snapshots must be UNAVAILABLE, otherwise
			// the network must be failed with an ERROR.
			if status != networksnapshottype.Status_FAILED {
				switch deviceSnapshot.Reason {
				case devicesnapshottype.Reason_ERROR:
					reason = networksnapshottype.Reason_ERROR
				case devicesnapshottype.Reason_UNAVAILABLE:
					reason = networksnapshottype.Reason_UNAVAILABLE
				}
			} else if reason == networksnapshottype.Reason_UNAVAILABLE && deviceSnapshot.Reason == devicesnapshottype.Reason_ERROR {
				reason = networksnapshottype.Reason_ERROR
			} else if reason == networksnapshottype.Reason_ERROR && deviceSnapshot.Reason == devicesnapshottype.Reason_UNAVAILABLE {
				reason = networksnapshottype.Reason_ERROR
			}
			status = networksnapshottype.Status_FAILED
		}
	}

	// If the status has changed, update the network config
	if config.Status != status || config.Reason != reason {
		config.Status = status
		config.Reason = reason
		if err := r.networkRequests.Update(config); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *Reconciler) deleteDeviceSnapshots(config *networksnapshottype.NetworkSnapshot) (bool, error) {
	for _, snapshotID := range config.GetDeviceSnapshotIDs() {
		deviceSnapshot, err := r.deviceRequests.Get(snapshotID)
		if err != nil {
			return false, err
		}
		err = r.deviceRequests.Delete(deviceSnapshot)
		if err != nil {
			return false, err
		}
	}
	return true, nil
}

func (r *Reconciler) applyNextNetworkSnapshot(config *networksnapshottype.NetworkSnapshot) (bool, error) {
	nextSnapshot, err := r.networkRequests.GetByIndex(config.Index + 1)
	if err != nil {
		return false, err
	} else if nextSnapshot != nil && nextSnapshot.Status == networksnapshottype.Status_PENDING {
		nextSnapshot.Status = networksnapshottype.Status_APPLYING
		if err := r.networkRequests.Update(nextSnapshot); err != nil {
			return false, err
		}
	}
	return true, nil
}

var _ controller.Reconciler = &Reconciler{}
