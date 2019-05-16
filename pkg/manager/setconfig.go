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
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"log"
	"time"
)

// SetConfigAlreadyApplied is a string constant for "Already applied:"
const SetConfigAlreadyApplied = "Already applied:"

// SetNetworkConfig sets the given value, according to the path on the configuration for the specified targed
func (m *Manager) SetNetworkConfig(target string, configName string, updates map[string]string,
	deletes []string) (change.ID, error) {
	if _, ok := m.DeviceStore.Store[target]; !ok {
		return nil, fmt.Errorf("Device not present %s", target)
	}
	//checks if config exists, otherwise create new
	deviceConfig, ok := m.ConfigStore.Store[configName]
	if !ok {
		deviceConfig = store.Configuration{
			Name:        store.ConfigName(configName),
			Device:      target,
			Created:     time.Now(),
			Updated:     time.Now(),
			User:        "User1",
			Description: fmt.Sprintf("Configuration %s for target %s", configName, target),
			Changes:     make([]change.ID, 0),
		}
	}

	var newChange = make([]*change.Value, 0)
	//updates
	for path, value := range updates {
		changeValue, _ := change.CreateChangeValue(path, value, false)
		newChange = append(newChange, changeValue)
	}

	//deletes
	for _, path := range deletes {
		changeValue, _ := change.CreateChangeValue(path, "", true)
		newChange = append(newChange, changeValue)
	}

	configChange, err := change.CreateChange(newChange,
		fmt.Sprintf("Update at %s", time.Now().Format(time.RFC3339)))
	if err != nil {
		return nil, err
	}

	if m.ChangeStore.Store[store.B64(configChange.ID)] != nil {
		log.Println("Change ID", store.B64(configChange.ID), "already exists - not overwriting")
	} else {
		m.ChangeStore.Store[store.B64(configChange.ID)] = configChange
		log.Println("Added change", store.B64(configChange.ID), "to ChangeStore (in memory)")
	}

	// If the last change applied to deviceConfig is the same as this one then don't apply it again
	if len(deviceConfig.Changes) > 0 &&
		store.B64(deviceConfig.Changes[len(deviceConfig.Changes)-1]) == store.B64(configChange.ID) {
		log.Println("Change", store.B64(configChange.ID),
			"has already been applied to", target, "Ignoring")
		return configChange.ID, fmt.Errorf("%s %s",
			SetConfigAlreadyApplied, store.B64(configChange.ID))
	}
	deviceConfig.Changes = append(deviceConfig.Changes, configChange.ID)
	deviceConfig.Updated = time.Now()
	m.ConfigStore.Store[configName] = deviceConfig

	// TODO: At this stage
	//  1) check that the new configuration is valid according to
	// 		the (YANG) models for the device
	//  2) Do a precheck that the device is reachable
	//  3) Check that the caller is authorized to make the change
	eventValues := make(map[string]string)
	eventValues[events.ChangeID] = store.B64(configChange.ID)
	eventValues[events.Committed] = "true"
	m.ChangesChannel <- events.CreateEvent(deviceConfig.Device,
		events.EventTypeConfiguration, eventValues)
	return configChange.ID, nil
}
