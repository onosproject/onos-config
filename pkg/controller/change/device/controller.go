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
	"strings"

	"github.com/onosproject/onos-config/api/types"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/controller"
	"github.com/onosproject/onos-config/pkg/southbound"
	changestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicechangeutils "github.com/onosproject/onos-config/pkg/store/change/device/utils"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	topodevice "github.com/onosproject/onos-topo/api/device"
)

var log = logging.GetLogger("controller", "change", "device")

// NewController returns a new network controller
func NewController(mastership mastershipstore.Store, devices devicestore.Store,
	cache cache.Cache, changes changestore.Store) *controller.Controller {

	c := controller.NewController("DeviceChange")
	c.Filter(&controller.MastershipFilter{
		Store:    mastership,
		Resolver: &Resolver{},
	})
	c.Partition(&Partitioner{})
	c.Watch(&Watcher{
		DeviceCache: cache,
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
func (r *Resolver) Resolve(id types.ID) (topodevice.ID, error) {
	return topodevice.ID(devicechange.ID(id).GetDeviceID()), nil
}

// Reconciler is a device change reconciler
type Reconciler struct {
	devices devicestore.Store
	changes changestore.Store
}

// Reconcile reconciles the state of a device change
func (r *Reconciler) Reconcile(id types.ID) (controller.Result, error) {
	// Get the change from the store
	change, err := r.changes.Get(devicechange.ID(id))
	if err != nil {
		return controller.Result{}, err
	}

	log.Infof("Reconciling DeviceChange %v", change)

	// The device controller only needs to handle changes in the RUNNING state
	if change == nil || change.Status.Incarnation == 0 || change.Status.State != changetypes.State_PENDING {
		return controller.Result{}, nil
	}

	// Get the device from the device store
	log.Infof("Checking Device store for %s", change.Change.DeviceID)
	device, err := r.devices.Get(topodevice.ID(change.Change.DeviceID))
	if err != nil {
		return controller.Result{}, err
	} else if getProtocolState(device) != topodevice.ChannelState_CONNECTED {
		// If the device is not available, fail the change
		change.Status.State = changetypes.State_FAILED
		change.Status.Reason = changetypes.Reason_ERROR
		change.Status.Message = fmt.Sprintf("Device %s not connected %v", change.Change.DeviceID, getProtocolState(device))
		log.Infof("Failing DeviceChange %v", change)
		if err := r.changes.Update(change); err != nil {
			return controller.Result{}, err
		}
		return controller.Result{}, nil
	}

	// Handle the change for each phase
	switch change.Status.Phase {
	case changetypes.Phase_CHANGE:
		return r.reconcileChange(change)
	case changetypes.Phase_ROLLBACK:
		return r.reconcileRollback(change)
	}
	return controller.Result{}, nil
}

// reconcileChange reconciles a CHANGE in the RUNNING state
func (r *Reconciler) reconcileChange(change *devicechange.DeviceChange) (controller.Result, error) {
	// Attempt to apply the change to the device and update the change with the result
	if err := r.doChange(change); err != nil {
		change.Status.State = changetypes.State_FAILED
		change.Status.Reason = changetypes.Reason_ERROR
		change.Status.Message = err.Error()
		log.Infof("Failing DeviceChange %v", change)
	} else {
		change.Status.State = changetypes.State_COMPLETE
		log.Infof("Completing DeviceChange %v", change)
	}

	// Update the change status in the store
	if err := r.changes.Update(change); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// doChange pushes the given change to the device
func (r *Reconciler) doChange(change *devicechange.DeviceChange) error {
	log.Infof("Applying change %v ", change.Change)
	return r.translateAndSendChange(change.Change)
}

// reconcileRollback reconciles a ROLLBACK in the RUNNING state
func (r *Reconciler) reconcileRollback(change *devicechange.DeviceChange) (controller.Result, error) {
	// Attempt to roll back the change to the device and update the change with the result
	if err := r.doRollback(change); err != nil {
		change.Status.State = changetypes.State_FAILED
		change.Status.Reason = changetypes.Reason_ERROR
		change.Status.Message = err.Error()
		log.Infof("Failing DeviceChange %v", change)
	} else {
		change.Status.State = changetypes.State_COMPLETE
		log.Infof("Completing DeviceChange %v", change)
	}

	// Update the change status in the store
	if err := r.changes.Update(change); err != nil {
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// doRollback rolls back a change on the device
func (r *Reconciler) doRollback(change *devicechange.DeviceChange) error {
	log.Infof("Execucting Rollback for %v", change)
	deltaChange, err := r.computeRollback(change)
	if err != nil {
		return err
	}
	log.Infof("Rolling back %v with %v", change.Change, deltaChange)
	return r.translateAndSendChange(deltaChange)
}

func (r *Reconciler) translateAndSendChange(change *devicechange.Change) error {
	setRequest, err := values.NativeChangeToGnmiChange(change)
	if err != nil {
		return err
	}
	log.Infof("Reconciler set request for %s: %v", change.DeviceID, setRequest)
	deviceTarget, err := southbound.GetTarget(topodevice.ID(change.DeviceID))
	if err != nil {
		log.Infof("Device %s is not connected, accepting change", change.DeviceID)
		return fmt.Errorf("Device not connected %s, error %s", change.DeviceID, err.Error())
	}
	log.Infof("Target for device %s: %v %v", change.DeviceID, deviceTarget, deviceTarget.Context())
	setResponse, err := deviceTarget.Set(*deviceTarget.Context(), setRequest)
	if err != nil {
		log.Error("Error while doing set: ", err)
		return err
	}
	log.Info(change.DeviceID, " SetResponse ", setResponse)
	return nil
}

func getProtocolState(device *topodevice.Device) topodevice.ChannelState {
	// Find the gNMI protocol state for the device
	var protocol *topodevice.ProtocolState
	for _, p := range device.Protocols {
		if p.Protocol == topodevice.Protocol_GNMI {
			protocol = p
			break
		}
	}
	if protocol == nil {
		return topodevice.ChannelState_UNKNOWN_CHANNEL_STATE
	}
	return protocol.ChannelState
}

// computeRollback returns a change containing the previous value for each path of the rollbackChange
func (r *Reconciler) computeRollback(deviceChange *devicechange.DeviceChange) (*devicechange.Change, error) {
	//TODO We might want to consider doing reverse iteration to get the previous value for a path instead of
	// reading up to the previous change for the target. see comments on PR #805
	previousValues := make([]*devicechange.ChangeValue, 0)
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
				previousValues = append(previousValues, &devicechange.ChangeValue{
					Path:  prevVal.Path,
					Value: prevVal.Value,
				})
			}
		}
		if _, ok := alreadyUpdated[rbValue.Path]; !ok {
			previousValues = append(previousValues, &devicechange.ChangeValue{
				Path:    rbValue.Path,
				Removed: true,
			})
		}
	}
	deltaChange := &devicechange.Change{
		DeviceID:      rollbackChange.DeviceID,
		DeviceVersion: rollbackChange.DeviceVersion,
		Values:        previousValues,
	}
	return deltaChange, nil
}

var _ controller.Reconciler = &Reconciler{}
