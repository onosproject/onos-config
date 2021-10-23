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
	"context"
	changetypes "github.com/onosproject/onos-api/go/onos/config/change"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	"github.com/onosproject/onos-api/go/onos/topo"
	configcontroller "github.com/onosproject/onos-config/pkg/controller"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/southbound"
	changestore "github.com/onosproject/onos-config/pkg/store/change/device"
	devicechangeutils "github.com/onosproject/onos-config/pkg/store/change/device/utils"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
)

var log = logging.GetLogger("controller", "change", "device")

// NewController returns a new network controller
func NewController(mastership mastershipstore.Store, devices devicestore.Store, changes changestore.Store) *controller.Controller {
	c := controller.NewController("DeviceChange")
	c.Filter(&configcontroller.MastershipFilter{
		Store:    mastership,
		Resolver: &Resolver{},
	})
	c.Partition(&Partitioner{})
	c.Watch(&Watcher{
		DeviceStore: devices,
		ChangeStore: changes,
	})
	c.Watch(&ChangeWatcher{
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
type Resolver struct{}

// Resolve resolves a device ID from a device change ID
func (r *Resolver) Resolve(id controller.ID) (topodevice.ID, error) {
	return topodevice.ID(devicechange.ID(id.String()).GetDeviceID()), nil
}

// Reconciler is a device change reconciler
type Reconciler struct {
	devices devicestore.Store
	changes changestore.Store
}

// Reconcile reconciles the state of a device change
func (r *Reconciler) Reconcile(id controller.ID) (controller.Result, error) {
	log.Infof("Reconciling DeviceChange '%s'", id.String())
	deviceChange, err := r.changes.Get(devicechange.ID(id.String()))
	if err != nil {
		if errors.IsNotFound(err) {
			log.Debugf("DeviceChange '%s' not found", id.String())
			return controller.Result{}, nil
		}
		log.Errorf("Error fetching DeviceChange '%s'", id.String(), err)
		return controller.Result{}, err
	}

	log.Debug(deviceChange)

	// If the change is not PENDING, skip reconciliation
	if deviceChange.Status.State != changetypes.State_PENDING {
		log.Debugf("Skipping reconciliation for DeviceChange '%s': Change is already %s", deviceChange.ID, deviceChange.Status.State)
		return controller.Result{}, nil
	}

	// Get the device from the device store
	device, err := r.devices.Get(topodevice.ID(deviceChange.Change.DeviceID))
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warnf("Waiting for Device '%s' to connect", deviceChange.Change.DeviceID)
			return controller.Result{}, nil
		}
		log.Errorf("Error fetching Device '%s'", deviceChange.Change.DeviceID, err)
		return controller.Result{}, err
	} else if getProtocolState(device) != topo.ChannelState_CONNECTED {
		log.Warnf("Waiting for Device '%s' to connect", deviceChange.Change.DeviceID)
		return controller.Result{}, nil
	}

	// Handle the change for each phase
	switch deviceChange.Status.Phase {
	case changetypes.Phase_CHANGE:
		return r.reconcileChange(deviceChange)
	case changetypes.Phase_ROLLBACK:
		return r.reconcileRollback(deviceChange)
	}
	return controller.Result{}, nil
}

// reconcileChange reconciles a CHANGE in the RUNNING state
func (r *Reconciler) reconcileChange(deviceChange *devicechange.DeviceChange) (controller.Result, error) {
	// Attempt to apply the change to the device and update the change with the result
	if complete, err := r.doChange(deviceChange); complete && err != nil {
		deviceChange.Status.State = changetypes.State_FAILED
		deviceChange.Status.Reason = changetypes.Reason_ERROR
		deviceChange.Status.Message = err.Error()
		log.Infof("Failing DeviceChange '%s': '%s'", deviceChange.ID, err.Error())
	} else if err != nil {
		log.Warnf("Applying DeviceChange '%s' failed. Retrying...", deviceChange.ID)
		return controller.Result{}, err
	} else if complete {
		deviceChange.Status.State = changetypes.State_COMPLETE
		log.Infof("Completing DeviceChange '%s'", deviceChange.ID)
	}

	// Update the change status in the store
	log.Debug(deviceChange)
	if err := r.changes.Update(deviceChange); err != nil && !errors.IsNotFound(err) && !errors.IsConflict(err) {
		log.Errorf("Error updating DeviceChange '%s'", deviceChange.ID, err)
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// doChange pushes the given change to the device
func (r *Reconciler) doChange(deviceChange *devicechange.DeviceChange) (bool, error) {
	log.Infof("Applying DeviceChange '%s' on Device '%s'", deviceChange.ID, deviceChange.Change.DeviceID)
	return r.translateAndSendChange(deviceChange.Change)
}

// reconcileRollback reconciles a ROLLBACK in the RUNNING state
func (r *Reconciler) reconcileRollback(deviceChange *devicechange.DeviceChange) (controller.Result, error) {
	// Attempt to roll back the change to the device and update the change with the result
	if complete, err := r.doRollback(deviceChange); complete && err != nil {
		deviceChange.Status.State = changetypes.State_FAILED
		deviceChange.Status.Reason = changetypes.Reason_ERROR
		deviceChange.Status.Message = err.Error()
		log.Infof("Failing DeviceChange '%s': '%s'", deviceChange.ID, err.Error())
	} else if err != nil {
		log.Warnf("Rolling back DeviceChange '%s' failed. Retrying...", deviceChange.ID)
		return controller.Result{}, err
	} else if complete {
		deviceChange.Status.State = changetypes.State_COMPLETE
		log.Infof("Completing DeviceChange '%s'", deviceChange.ID)
	}

	// Update the change status in the store
	log.Debug(deviceChange)
	if err := r.changes.Update(deviceChange); err != nil && !errors.IsNotFound(err) && !errors.IsConflict(err) {
		log.Errorf("Error updating DeviceChange '%s'", deviceChange.ID, err)
		return controller.Result{}, err
	}
	return controller.Result{}, nil
}

// doRollback rolls back a change on the device
func (r *Reconciler) doRollback(deviceChange *devicechange.DeviceChange) (bool, error) {
	log.Infof("Rolling back DeviceChange '%s' on Device '%s'", deviceChange.ID, deviceChange.Change.DeviceID)
	deltaChange, err := r.computeRollback(deviceChange)
	if err != nil {
		log.Errorf("Error computing rollback for DeviceChange '%s'", deviceChange.ID, err)
		return false, err
	}
	return r.translateAndSendChange(deltaChange)
}

func (r *Reconciler) translateAndSendChange(change *devicechange.Change) (bool, error) {
	setRequest, err := values.NativeChangeToGnmiChange(change)
	if err != nil {
		log.Errorf("Conversion of %+v to gNMI change failed", change, err)
		return true, err
	}
	deviceTarget, err := southbound.GetTarget(change.GetVersionedDeviceID())
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warnf("Device %s:%s (%s) is not connected", change.DeviceID, change.DeviceVersion, change.DeviceType)
			return false, err
		}
		log.Error(err)
		return true, err
	}
	log.Debugf("Sending gNMI SetRequest %+v to Device %s:%s", setRequest, change.DeviceID, change.DeviceVersion)
	setResponse, err := deviceTarget.Set(*deviceTarget.Context(), setRequest)
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return false, err
		}
		code := status.Code(err)
		if code == codes.Unavailable ||
			code == codes.Canceled ||
			code == codes.DeadlineExceeded ||
			code == codes.Aborted ||
			code == codes.Unauthenticated ||
			code == codes.PermissionDenied {
			log.Warnf("Error in SetRequest %+v to Device %s:%s", setRequest, change.DeviceID, change.DeviceVersion, err)
			return false, err
		}
		log.Errorf("Error in SetRequest %+v to Device %s:%s", setRequest, change.DeviceID, change.DeviceVersion, err)
		return true, err
	}
	log.Debugf("Received gNMI SetResponse %+v from Device %s:%s", setResponse, change.DeviceID, change.DeviceVersion)
	return true, nil
}

// computeRollback returns a change containing the previous value for each path of the rollbackChange
func (r *Reconciler) computeRollback(deviceChange *devicechange.DeviceChange) (*devicechange.Change, error) {
	//TODO We might want to consider doing reverse iteration to get the previous value for a path instead of
	// reading up to the previous change for the target. see comments on PR #805
	previousValues := make([]*devicechange.ChangeValue, 0)
	prevValues, err := devicechangeutils.ExtractFullConfig(deviceChange.Change.GetVersionedDeviceID(), nil, r.changes, 0)
	if err != nil {
		return nil, err
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

func getProtocolState(device *topodevice.Device) topo.ChannelState {
	// Find the gNMI protocol state for the device
	var protocol *topo.ProtocolState
	for _, p := range device.Protocols {
		if p.Protocol == topo.Protocol_GNMI {
			protocol = p
			break
		}
	}
	if protocol == nil {
		return topo.ChannelState_UNKNOWN_CHANNEL_STATE
	}
	return protocol.ChannelState
}

var _ controller.Reconciler = &Reconciler{}
