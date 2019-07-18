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
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	log "k8s.io/klog"
	"time"
)

// RollbackTargetConfig rollbacks the last change for a given configuration on the target. Only the last one is
// restored to the previous state. Going back n changes in time requires n sequential calls of this method.
func (m *Manager) RollbackTargetConfig(configname string) (change.ID, error) {
	targetID := m.ConfigStore.Store[store.ConfigName(configname)].Device
	log.Infof("Rolling back last change on config %s for target %s", configname, targetID)
	id, updates, deletes, err := computeRollback(m, targetID, configname)
	if err != nil {
		log.Errorf("Error on rollback: %s", err.Error())
		return nil, err
	}
	chg, err := m.computeChange(updates, deletes)
	if err != nil {
		return id, err
	}
	changeID, err := m.storeChange(chg)
	if err != nil {
		return id, err
	}
	m.ChangesChannel <- events.CreateConfigEvent(targetID,
		changeID, true)
	return id, listenForDeviceResponse(m, targetID)
}

func computeRollback(m *Manager, target string, configname string) (change.ID, map[string]*change.TypedValue, []string, error) {
	id, err := m.ConfigStore.RemoveLastChangeEntry(store.ConfigName(configname))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Can't remove last entry on target %s in config %s, %s",
			target, configname, err.Error())
	}
	previousValues := make([]*change.ConfigValue, 0)
	deletes := make([]string, 0)
	rollbackChange := m.ChangeStore.Store[store.B64(id)]
	for _, valueColl := range rollbackChange.Config {
		value, err := m.GetTargetConfig(target, configname, valueColl.Path, 0)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Can't get last config for path %s on config %s for target %s",
				valueColl.Path, configname, err)
		}
		//Previously there was no such value configured, deleting from device
		if len(value) == 0 {
			deletes = append(deletes, valueColl.Path)
		} else {
			previousValues = append(previousValues, value[0])
		}
	}
	updates := make(map[string]*change.TypedValue)
	for _, changeVal := range previousValues {
		updates[changeVal.Path] = &changeVal.TypedValue
	}
	return id, updates, deletes, nil
}

func listenForDeviceResponse(mgr *Manager, target string) error {
	respChan, ok := mgr.Dispatcher.GetResponseListener(topocache.ID(target))
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
			return nil
		case events.EventTypeErrorSetConfig:
			//TODO if the device gives an error during this rollback we currently do nothing
			return fmt.Errorf("Rollback thrown error %s, system is in inconsistent state for device %s ",
				response.Error().Error(), target)

		default:
			return fmt.Errorf("undhandled Error Type")

		}
	case <-time.After(5 * time.Second):
		return fmt.Errorf("Timeout on waiting for device reply %s", target)
	}
}
