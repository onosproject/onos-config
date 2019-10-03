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
	"github.com/onosproject/onos-topo/pkg/northbound/device"
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
}

// Reconcile reconciles a network snapshot request
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	networkRequest, err := r.networkRequests.Get(networksnapshottype.ID(id))
	if err != nil {
		return false, err
	}

	// If the list of devices in the request is empty, set the devices
	if networkRequest.Devices == nil || len(networkRequest.Devices) == 0 {
		deviceIDs := make([]device.ID, 0)
		ch := make(chan *device.Device)
		if err := r.devices.List(ch); err != nil {
			return false, err
		}

		for device := range ch {
			deviceIDs = append(deviceIDs, device.ID)
		}
		networkRequest.Devices = deviceIDs
		if err := r.networkRequests.Update(networkRequest); err != nil {
			return false, err
		}
	}

	// Loop through devices and ensure device snapshots have been created
	for _, deviceID := range networkRequest.Devices {
		snapshotID := networkRequest.GetDeviceSnapshotID(deviceID)
		deviceRequest, err := r.deviceRequests.Get(devicesnapshottype.ID(snapshotID))
		if err != nil {
			return false, err
		} else if deviceRequest == nil {
			deviceRequest = &devicesnapshottype.DeviceSnapshot{
				ID:        devicesnapshottype.ID(snapshotID),
				DeviceID:  deviceID,
				Timestamp: networkRequest.Timestamp,
				Status:    devicesnapshottype.Status_PENDING,
			}
			if err := r.deviceRequests.Create(deviceRequest); err != nil {
				return false, err
			}
		}

		// Ensure the device status matches the network status.
		if networkRequest.Status == networksnapshottype.Status_PENDING {
			if deviceRequest.Status != devicesnapshottype.Status_PENDING {
				deviceRequest.Status = devicesnapshottype.Status_PENDING
				if err := r.deviceRequests.Update(deviceRequest); err != nil {
					return false, err
				}
			}
		} else if networkRequest.Status == networksnapshottype.Status_APPLYING {
			if deviceRequest.Status == devicesnapshottype.Status_PENDING {
				deviceRequest.Status = devicesnapshottype.Status_APPLYING
				if err := r.deviceRequests.Update(deviceRequest); err != nil {
					return false, err
				}
			}
		} else {
			return true, nil
		}
	}
	return true, nil
}

var _ controller.Reconciler = &Reconciler{}
