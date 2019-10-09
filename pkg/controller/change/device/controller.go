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
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/types"
	changetype "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetype "github.com/onosproject/onos-config/pkg/types/change/device"
	deviceservice "github.com/onosproject/onos-topo/pkg/northbound/device"
)

// NewController returns a new network controller
func NewController(mastership mastershipstore.Store, devices devicestore.Store, changes changestore.Store) *controller.Controller {
	c := controller.NewController()
	c.Filter(&controller.MastershipFilter{
		Store: mastership,
	})
	c.Partition(&Partitioner{})
	c.Watch(&Watcher{
		DeviceStore: devices,
		ChangeStore: changes,
	})
	c.Reconcile(&Reconciler{
		devices: devices,
		changes: changes,
	})
	return c
}

// Reconciler is a device change reconciler
type Reconciler struct {
	devices devicestore.Store
	changes changestore.Store
}

// Reconcile reconciles the state of a device change
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	// Get the change from the store
	change, err := r.changes.Get(devicechangetype.ID(id))
	if err != nil {
		return false, err
	}

	// If the change is not found, skip processing
	if change == nil {
		return true, nil
	}

	// If the change is in the PENDING state and has been deleted, roll back the change
	if change.Status.State == changetype.State_PENDING && change.Status.Deleted {
		return r.applyRollback(change)
	}

	// If the change is in the APPLYING state and has not been deleted, apply the change
	if change.Status.State == changetype.State_APPLYING && !change.Status.Deleted {
		return r.applyChange(change)
	}

	// If the change is in the APPLYING state and has been deleted, roll back the change
	if change.Status.State == changetype.State_APPLYING && change.Status.Deleted {
		return r.applyRollback(change)
	}
	return true, nil
}

// applyChange attempts to apply a change
func (r *Reconciler) applyChange(change *devicechangetype.Change) (bool, error) {
	// Get the device from the device store
	device, err := r.devices.Get(change.DeviceID)
	if err != nil {
		return false, err
	}

	// Find the gNMI protocol state for the device
	var protocol *deviceservice.ProtocolState
	for _, p := range device.Protocols {
		if p.Protocol == deviceservice.Protocol_GNMI {
			protocol = p
			break
		}
	}

	// If the device is not available, fail the change with the UNAVAILABLE reason
	if protocol == nil || protocol.ChannelState != deviceservice.ChannelState_CONNECTED {
		change.Status.State = changetype.State_FAILED
		change.Status.Reason = changetype.Reason_UNAVAILABLE
		if err := r.changes.Update(change); err != nil {
			return false, err
		}
		return true, nil
	}

	// Finally, attempt to apply the change
	// If an error occurs when applying the change, fail the device change, otherwise succeed it
	if err := r.doChange(change); err != nil {
		change.Status.State = changetype.State_FAILED
		change.Status.Reason = changetype.Reason_ERROR
		change.Status.Message = err.Error()
	} else {
		change.Status.State = changetype.State_SUCCEEDED
	}

	// Update the change state in the store before returning
	if err := r.changes.Update(change); err != nil {
		return false, err
	}
	return true, nil
}

// doChange pushes the given change to the device
func (r *Reconciler) doChange(change *devicechangetype.Change) error {
	// TODO: Apply the change
	return nil
}

// applyRollback attempts to roll back a change
func (r *Reconciler) applyRollback(change *devicechangetype.Change) (bool, error) {
	// Finally, attempt to roll back the change
	// If an error occurs when rolling back the change and the change is in the APPLYING state,
	// fail the change
	// If the rollback is successful and the change is in the APPLYING state, succeed the change
	if err := r.doRollback(change); err != nil {
		if change.Status.State == changetype.State_APPLYING {
			change.Status.State = changetype.State_FAILED
			change.Status.Reason = changetype.Reason_ERROR
			change.Status.Message = err.Error()
		} else {
			return false, err
		}
	} else {
		if change.Status.State == changetype.State_APPLYING {
			change.Status.State = changetype.State_SUCCEEDED
		}
	}

	// Update the change state in the store before returning
	if err := r.changes.Update(change); err != nil {
		return false, err
	}
	return true, nil
}

// doRollback rolls back a change on the device
func (r *Reconciler) doRollback(change *devicechangetype.Change) error {
	// TODO: Roll back the change
	return nil
}

var _ controller.Reconciler = &Reconciler{}
