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
	log "k8s.io/klog"
	"strings"
	"time"
)

// SetConfigAlreadyApplied is a string constant for "Already applied:"
const SetConfigAlreadyApplied = "Already applied:"

// ValidateNetworkConfig validates the given updates and deletes, according to the path on the configuration
// for the specified target
func (m *Manager) ValidateNetworkConfig(configName store.ConfigName, updates change.TypedValueMap,
	deletes []string) error {

	deviceConfig, _, err := m.getStoredConfig(configName)
	if err != nil {
		return err
	}
	deviceConfigTemporary, _ := store.CreateConfiguration(deviceConfig.Device, deviceConfig.Version,
		deviceConfig.Type, deviceConfig.Changes)
	chg, err := m.computeChange(updates, deletes)
	if err != nil {
		return err
	}
	changeID, err := m.storeChange(chg)
	if err != nil {
		return err
	}

	deviceConfigTemporary.Changes = append(deviceConfig.Changes, changeID)

	modelName := fmt.Sprintf("%s-%s", deviceConfig.Type, deviceConfig.Version)
	deviceModelYgotPlugin, ok := m.ModelRegistry.ModelPlugins[modelName]
	if !ok {
		log.Warning("No model ", modelName, " available as a plugin")
	} else {
		configValues := deviceConfigTemporary.ExtractFullConfig(m.ChangeStore.Store, 0)
		jsonTree, err := store.BuildTree(configValues, true)
		if err != nil {
			log.Error("Error building JSON tree from Config Values ", err, jsonTree)
		} else {
			ygotModel, err := deviceModelYgotPlugin.UnmarshalConfigValues(jsonTree)
			if err != nil {
				log.Error("Error unmarshaling JSON tree in to YGOT model ", err)
				return err
			}
			err = deviceModelYgotPlugin.Validate(ygotModel)
			if err != nil {
				return err
			}
			log.Info("Configuration is Valid according to model",
				modelName, "after adding change", store.B64(changeID))
		}
	}
	return nil
}

// SetNetworkConfig sets the given the given updates and deletes, according to the path on the configuration
// for the specified target
func (m *Manager) SetNetworkConfig(configName store.ConfigName, updates change.TypedValueMap,
	deletes []string) (change.ID, store.ConfigName, error) {

	deviceConfig, configName, err := m.getStoredConfig(configName)
	if err != nil {
		return nil, configName, err
	}
	chg, err := m.computeChange(updates, deletes)
	if err != nil {
		return nil, configName, err
	}
	changeID := chg.ID
	// If the last change applied to deviceConfig is the same as this one then don't apply it again
	if len(deviceConfig.Changes) > 0 &&
		store.B64(deviceConfig.Changes[len(deviceConfig.Changes)-1]) == store.B64(changeID) {
		log.Info("Change ", store.B64(changeID),
			"has already been applied to", configName, "Ignoring")
		return changeID, configName, fmt.Errorf("%s %s",
			SetConfigAlreadyApplied, store.B64(changeID))
	}
	//FIXME this needs to hold off until the device replies with an ok message for the change.
	deviceConfig.Changes = append(deviceConfig.Changes, changeID)
	deviceConfig.Updated = time.Now()
	m.ConfigStore.Store[configName] = deviceConfig

	// TODO: At this stage
	//  2) Do a precheck that the device is reachable
	//  3) Check that the caller is authorized to make the change
	m.ChangesChannel <- events.CreateConfigEvent(deviceConfig.Device,
		changeID, true)

	return changeID, configName, nil
}

// getStoredConfig looks for an exact match for the config name or then a partial match based on the device name
func (m *Manager) getStoredConfig(configName store.ConfigName) (store.Configuration, store.ConfigName, error) {
	deviceConfig, ok := m.ConfigStore.Store[configName]
	if !ok {
		similarDevices := make([]store.ConfigName, 0)
		for key := range m.ConfigStore.Store {
			deviceNameFromKey := string(key)[:strings.LastIndex(string(key), "-")]
			if deviceNameFromKey == string(configName) {
				similarDevices = append(similarDevices, key)
			}
		}

		if len(similarDevices) == 1 {
			log.Warning("No exact match in Configurations for ", configName,
				" using ", similarDevices[0])
			configName = similarDevices[0]
			deviceConfig = m.ConfigStore.Store[configName]
		} else if len(similarDevices) > 1 {
			var conv = func(similarDevices []store.ConfigName) string {
				similarDevicesStr := make([]string, len(similarDevices))
				for i, d := range similarDevices {
					similarDevicesStr[i] = string(d)
				}
				return strings.Join(similarDevicesStr, ", ")
			}
			return store.Configuration{}, configName, fmt.Errorf("%d configurations found for '%s': %s"+
				". Please specify a version in extension 101", len(similarDevices),
				configName, conv(similarDevices))
		} else {
			return store.Configuration{}, configName, fmt.Errorf("no configuration found matching '%s'."+
				"Please specify version and device type in extensions 101 and 102", configName)
			// FIXME add in handler for actually dealing with getting version and type and modeldata
		}
	}
	return deviceConfig, configName, nil
}
