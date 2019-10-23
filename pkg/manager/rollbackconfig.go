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

package manager

import (
	"fmt"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
	"time"
)

// RollbackTargetConfig rollbacks the last change for a given configuration on the target. Only the last one is
// restored to the previous state. Going back n changes in time requires n sequential calls of this method.
// A change is created so that it can be passed down to the real device
// Deprecated: RollbackTargetConfig works on legacy, non-atomix stores
func (m *Manager) RollbackTargetConfig(configname store.ConfigName) (change.ID, error) {
	targetID := m.ConfigStore.Store[store.ConfigName(configname)].Device
	log.Infof("Rolling back last change on config %s for target %s", configname, targetID)
	id, updates, deletes, err := computeRollback(m, targetID, configname)
	if err != nil {
		log.Errorf("Error on rollback: %s", err.Error())
		return nil, err
	}
	chg, err := m.computeChange(updates, deletes,
		fmt.Sprintf("Rollback of %s at %s", string(configname), time.Now().Format(time.RFC3339)))
	if err != nil {
		return id, err
	}
	changeID, err := m.storeChange(chg)
	if err != nil {
		return id, err
	}
	m.ChangesChannel <- events.NewConfigEvent(targetID,
		changeID, true)
	return id, listenForDeviceResponse(m, targetID)
}

// NewRollbackTargetConfig rollbacks the last change for a given configuration on the target, by setting phase to
// rollback and state to pending.
func (m *Manager) NewRollbackTargetConfig(networkChangeID networkchangetypes.ID) error {
	//TODO make sure this change is the last applied one
	changeRollback, errGet := m.NetworkChangesStore.Get(networkChangeID)
	if errGet != nil {
		log.Errorf("Error on get change %s for rollback: %s", networkChangeID, errGet)
		return errGet
	}

	computedChange, errCompute := computeNewRollback(m, changeRollback)
	if errCompute != nil {
		log.Errorf("Error in computing rollback for id %s for rollback: %s",
			networkChangeID, errCompute)
		return errCompute
	}
	log.Infof("Rolling back %s with %s", changeRollback.Changes, computedChange.Changes)
	errUpdate := m.NetworkChangesStore.Create(computedChange)
	if errUpdate != nil {
		log.Errorf("Error on setting change %s rollback: %s", computedChange.ID, errUpdate)
		return errUpdate
	}
	return listenForChangeNotification(m, computedChange.ID)
}

//Deprecated computeRollback works on old non atomix methods
func computeRollback(m *Manager, target string, configname store.ConfigName) (change.ID, devicechangetypes.TypedValueMap, []string, error) {
	id, err := m.ConfigStore.RemoveLastChangeEntry(configname)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("can't remove last entry on target %s in config %s, %s",
			target, configname, err.Error())
	}
	previousValues := make([]*devicechangetypes.PathValue, 0)
	deletes := make([]string, 0)
	rollbackChange := m.ChangeStore.Store[store.B64(id)]
	for _, valueColl := range rollbackChange.Config {
		value, err := m.GetTargetConfig(target, configname, valueColl.Path, 0)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("can't get last config for path %s on config %s for target %s",
				valueColl.Path, configname, err)
		}
		//Previously there was no such value configured, deleting from devicetopo
		if len(value) == 0 {
			deletes = append(deletes, valueColl.Path)
		} else {
			previousValues = append(previousValues, value[0])
		}
	}
	updates := make(devicechangetypes.TypedValueMap)
	for _, changeVal := range previousValues {
		updates[changeVal.Path] = changeVal.GetValue()
	}
	return id, updates, deletes, nil
}

//Deprecated computeRollback works on old non atomix methods
func computeNewRollback(m *Manager, rollbackChange *networkchangetypes.NetworkChange) (*networkchangetypes.NetworkChange, error) {
	deltaChange := &networkchangetypes.NetworkChange{
		ID: networkchangetypes.ID(namesgenerator.GetRandomName(0)),
		Status: changetypes.Status{
			Phase:   changetypes.Phase_ROLLBACK,
			State:   changetypes.State_PENDING,
			Reason:  changetypes.Reason_NONE,
			Message: "Change generated for administratively requested rollback of " + string(rollbackChange.ID),
		},
	}
	prevChanges := make([]*devicechangetypes.Change, 0)
	for _, deviceChange := range rollbackChange.Changes {
		previousValues := make([]*devicechangetypes.ChangeValue, 0)
		for _, value := range deviceChange.Values {
			preVal, err := m.GetTargetNewConfig(string(deviceChange.DeviceID), value.Path, 1)
			if err != nil {
				return nil, fmt.Errorf("can't get last config for path %s on network config %s for target %s, %s",
					value.Path, string(rollbackChange.ID), deviceChange.DeviceID, err)
			}
			//Previously there was no such value configured, deleting from devicetopo
			if len(preVal) == 0 {
				deleteVal := &devicechangetypes.ChangeValue{
					Path:    value.Path,
					Value:   nil,
					Removed: true,
				}
				previousValues = append(previousValues, deleteVal)
			} else {
				updateVal := &devicechangetypes.ChangeValue{
					Path:    preVal[0].Path,
					Value:   preVal[0].Value,
					Removed: false,
				}
				previousValues = append(previousValues, updateVal)
			}

		}
		rollbackChange := &devicechangetypes.Change{
			DeviceID:      deviceChange.DeviceID,
			DeviceVersion: deviceChange.DeviceVersion,
			DeviceType:    deviceChange.DeviceType,
			Values:        previousValues,
		}
		prevChanges = append(prevChanges, rollbackChange)
	}
	deltaChange.Changes = prevChanges
	deltaChange.Created = time.Now()
	deltaChange.Updated = time.Now()
	return deltaChange, nil
}

func listenForDeviceResponse(mgr *Manager, target string) error {
	respChan, ok := mgr.Dispatcher.GetResponseListener(devicetopo.ID(target))
	if !ok {
		log.Infof("Device %s not properly registered, not waiting for southbound confirmation", target)
		return nil
	}
	//blocking until we receive something from the channel or for 5 seconds, whatever comes first.
	select {
	case response := <-respChan:
		switch eventType := response.EventType(); eventType {
		case events.EventTypeAchievedSetConfig:
			log.Infof("Rollback succeeded on %s ", target)
			go func() {
				respChan <- events.NewResponseEvent(events.EventTypeSubscribeNotificationSetConfig,
					response.Subject(), []byte(response.ChangeID()), response.String())
			}()
			return nil
		case events.EventTypeErrorSetConfig:
			//TODO if the device gives an error during this rollback we currently do nothing
			go func() {
				respChan <- events.NewResponseEvent(events.EventTypeSubscribeErrorNotificationSetConfig,
					response.Subject(), []byte(response.ChangeID()), response.String())
			}()
			return fmt.Errorf("Rollback thrown error %s, system is in inconsistent state for device %s ",
				response.Error().Error(), target)
		case events.EventTypeSubscribeNotificationSetConfig:
			return nil
		case events.EventTypeSubscribeErrorNotificationSetConfig:
			return nil
		default:
			return fmt.Errorf("undhandled Error Type %s, error %s", response.EventType(), response.Error())

		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("Timeout on waiting for device reply %s", target)
	}
}

func listenForChangeNotification(mgr *Manager, changeID networkchangetypes.ID) error {
	networkChan := make(chan stream.Event)
	ctx, errWatch := mgr.NetworkChangesStore.Watch(networkChan, networkchangestore.WithChangeID(changeID))
	if errWatch != nil {
		return fmt.Errorf("can't complete rollback operation on target due to %s", errWatch)
	}
	defer ctx.Close()
	for changeEvent := range networkChan {
		change := changeEvent.Object.(*networkchangetypes.NetworkChange)
		log.Infof("Received notification for change ID %s, phase %s, state %s", change.ID,
			change.Status.Phase, change.Status.State)
		if change.Status.Phase == changetypes.Phase_ROLLBACK {
			switch changeStatus := change.Status.State; changeStatus {
			case changetypes.State_COMPLETE:
				log.Infof("Rollback succeeded for change %s ", changeID)
				return nil
			case changetypes.State_FAILED:
				log.Infof("Received Change Status %s", changeStatus)
				return fmt.Errorf("issue in setting config reson %s, error %s, rolling back change %s",
					change.Status.Reason, change.Status.Message, changeID)
			default:
				continue
			}
		}
	}
	return nil
}
