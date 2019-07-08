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
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	log "k8s.io/klog"
)

// RollbackTargetConfig rollbacks the last change for a given configuration on the target. Only the last one is
// restored to the previous state. Going back n changes in time requires n sequential calls of this method.
func (m *Manager) RollbackTargetConfig(target string, configname string, id change.ID) error {
	log.Infof("Rolling back %s on config %s for target %s", id, configname, target)
	updates, deletes, err := computeRollback(m, target, configname, id)
	if err != nil {
		return err
	}
	_, _, errSet := m.SetNetworkConfig(store.ConfigName(configname), updates, deletes)
	//TODO if error we might want to take further action
	return errSet
}

func computeRollback(m *Manager, target string, configname string, id change.ID) (map[string]string, []string, error) {
	err := m.ConfigStore.RemoveLastChangeEntry(store.ConfigName(configname))
	if err != nil {
		return nil, nil, fmt.Errorf(fmt.Sprintf("Can't remove last entry for %s", configname), err)
	}
	previousValues := make([]*change.ConfigValue, 0)
	deletes := make([]string, 0)
	rollbackChange := m.ChangeStore.Store[store.B64(id)]
	for _, valueColl := range rollbackChange.Config {
		value, err := m.GetTargetConfig(target, configname, valueColl.Path, 0)
		if err != nil {
			return nil, nil, fmt.Errorf("Can't get last config for path %s on config %s for target %s",
				valueColl.Path, configname, err)
		}
		//Previously there was no such value configured, deletring from device
		if len(value) == 0 {
			deletes = append(deletes, valueColl.Path)
		} else {
			previousValues = append(previousValues, value[0])
		}
	}
	updates := make(map[string]string)
	for _, changeVal := range previousValues {
		updates[changeVal.Path] = changeVal.Value
	}
	return updates, deletes, nil
}
