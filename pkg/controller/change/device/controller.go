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
	"fmt"
	"github.com/onosproject/onos-config/pkg/controller"
	"github.com/onosproject/onos-config/pkg/southbound"
	changestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicechangeutils "github.com/onosproject/onos-config/pkg/store/change/device/utils"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/types"
	changetype "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/utils/values"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
	"strings"
)

// NewController returns a new network controller
func NewController(mastership mastershipstore.Store, devices devicestore.Store, changes changestore.Store) *controller.Controller {
	c := controller.NewController("DeviceChange")
	c.Filter(&controller.MastershipFilter{
		Store:    mastership,
		Resolver: &Resolver{},
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

// Resolver is a DeviceResolver that resolves device IDs from device change IDs
type Resolver struct {
}

// Resolve resolves a device ID from a device change ID
func (r *Resolver) Resolve(id types.ID) (devicetopo.ID, error) {
	return devicetopo.ID(devicechangetypes.ID(id).GetDeviceID()), nil
}

// Reconciler is a device change reconciler
type Reconciler struct {
	devices devicestore.Store
	changes changestore.Store
}

// Reconcile reconciles the state of a device change
func (r *Reconciler) Reconcile(id types.ID) (bool, error) {
	// Get the change from the store
	change, err := r.changes.Get(devicechangetypes.ID(id))
	if err != nil {
		return false, err
	}

	// The device controller only needs to handle changes in the RUNNING state
	if change == nil || change.Status.State != changetype.State_RUNNING {
		return true, nil
	}

	// Get the device from the device store
	device, err := r.devices.Get(devicetopo.ID(change.Change.DeviceID))
	if err != nil {
		return false, err
	}

	// If the device is not available, fail the change
	if getProtocolState(device) != devicetopo.ChannelState_CONNECTED {
		change.Status.State = changetype.State_FAILED
		change.Status.Reason = changetype.Reason_ERROR
		log.Infof("Failing DeviceChange %v", change)
		if err := r.changes.Update(change); err != nil {
			return false, err
		}
		return true, nil
	}

	// Handle the change for each phase
	switch change.Status.Phase {
	case changetype.Phase_CHANGE:
		return r.reconcileChange(change)
	case changetype.Phase_ROLLBACK:
		return r.reconcileRollback(change)
	}
	return true, nil
}

// reconcileChange reconciles a CHANGE in the RUNNING state
func (r *Reconciler) reconcileChange(change *devicechangetypes.DeviceChange) (bool, error) {
	// Attempt to apply the change to the device and update the change with the result
	if err := r.doChange(change); err != nil {
		change.Status.State = changetype.State_FAILED
		change.Status.Reason = changetype.Reason_ERROR
		change.Status.Message = err.Error()
		log.Infof("Failing DeviceChange %v", change)
	} else {
		change.Status.State = changetype.State_COMPLETE
		log.Infof("Completing DeviceChange %v", change)
	}

	// Update the change status in the store
	if err := r.changes.Update(change); err != nil {
		return false, err
	}
	return true, nil
}

// doChange pushes the given change to the device
func (r *Reconciler) doChange(change *devicechangetypes.DeviceChange) error {
	log.Infof("Applying change %v ", change.Change)
	return r.translateAndSendChange(change.Change)
}

// reconcileRollback reconciles a ROLLBACK in the RUNNING state
func (r *Reconciler) reconcileRollback(change *devicechangetypes.DeviceChange) (bool, error) {
	// Attempt to roll back the change to the device and update the change with the result
	if err := r.doRollback(change); err != nil {
		change.Status.State = changetype.State_FAILED
		change.Status.Reason = changetype.Reason_ERROR
		change.Status.Message = err.Error()
		log.Infof("Failing DeviceChange %v", change)
	} else {
		change.Status.State = changetype.State_COMPLETE
		log.Infof("Completing DeviceChange %v", change)
	}

	// Update the change status in the store
	if err := r.changes.Update(change); err != nil {
		return false, err
	}
	return true, nil
}

// doRollback rolls back a change on the device
func (r *Reconciler) doRollback(change *devicechangetypes.DeviceChange) error {
	log.Infof("Execucting Rollback for %v", change)
	deltaChange, err := r.computeNewRollback(change)
	if err != nil {
		return err
	}
	log.Infof("Rolling back %v with %v", change.Change, deltaChange)
	return r.translateAndSendChange(deltaChange)
}

func (r *Reconciler) translateAndSendChange(change *devicechangetypes.Change) error {
	setRequest, err := values.NativeNewChangeToGnmiChange(change)
	if err != nil {
		return err
	}
	log.Info("Reconciler set request ", setRequest)
	log.Info("Device ", change.DeviceID)
	deviceTarget, err := southbound.GetTarget(devicetopo.ID(change.DeviceID))
	if err != nil {
		//TODO revisit change state.
		log.Infof("Device %s is not connected, accepting change", change.DeviceID)
		return nil
	}
	setResponse, err := deviceTarget.Set(deviceTarget.Ctx, setRequest)
	if err != nil {
		log.Error("Error while doing set: ", err)
		return err
	}
	log.Info(change.DeviceID, " SetResponse ", setResponse)
	return nil
}

func getProtocolState(device *devicetopo.Device) devicetopo.ChannelState {
	// Find the gNMI protocol state for the device
	var protocol *devicetopo.ProtocolState
	for _, p := range device.Protocols {
		if p.Protocol == devicetopo.Protocol_GNMI {
			protocol = p
			break
		}
	}
	if protocol == nil {
		return devicetopo.ChannelState_UNKNOWN_CHANNEL_STATE
	}
	return protocol.ChannelState
}

// computeRollback returns a change containing the previous value for each path of the rollbackChange
func (r *Reconciler) computeNewRollback(deviceChange *devicechangetypes.DeviceChange) (*devicechangetypes.Change, error) {
	//TODO We might want to consider doing reverse iteration to get the previous value for a path instead of
	// reading up to the previous change for the target. see comments on PR #805
	previousValues := make([]*devicechangetypes.ChangeValue, 0)
	prevValues, err := devicechangeutils.ExtractFullConfig(deviceChange.Change.GetVersionedDeviceID(), nil, r.changes, 0)
	if err != nil {
		return nil, fmt.Errorf("can't get last config on network config %s for target %s, %s",
			string(deviceChange.ID), deviceChange.Change.DeviceID, err)
	}
	rollbackChange := deviceChange.Change
	alreadyUpdated := make(map[string]struct{})
	for _, rbValue := range rollbackChange.Values {
		for _, prevVal := range prevValues {
			if prevVal.Path == rbValue.Path ||
				rbValue.Removed && strings.HasPrefix(prevVal.Path, rbValue.Path) {
				alreadyUpdated[rbValue.Path] = struct{}{}
				previousValues = append(previousValues, &devicechangetypes.ChangeValue{
					Path:  prevVal.Path,
					Value: prevVal.Value,
				})
			}
		}
		if _, ok := alreadyUpdated[rbValue.Path]; !ok {
			previousValues = append(previousValues, &devicechangetypes.ChangeValue{
				Path:    rbValue.Path,
				Removed: true,
			})
		}
	}
	deltaChange := &devicechangetypes.Change{
		DeviceID:      rollbackChange.DeviceID,
		DeviceVersion: rollbackChange.DeviceVersion,
		Values:        previousValues,
	}
	return deltaChange, nil
}

var _ controller.Reconciler = &Reconciler{}
