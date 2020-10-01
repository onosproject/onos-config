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
	"sort"

	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/utils"
)

// SetConfigAlreadyApplied is a string constant for "Already applied:"
const SetConfigAlreadyApplied = "Already applied:"

// ValidateNetworkConfig validates the given updates and deletes, according to the path on the configuration
// for the specified target (Atomix Based)
func (m *Manager) ValidateNetworkConfig(deviceName devicetype.ID, version devicetype.Version,
	deviceType devicetype.Type, updates devicechange.TypedValueMap, deletes []string, lastWrite networkchange.Revision) error {

	chg, err := m.ComputeDeviceChange(deviceName, version, deviceType, updates, deletes, "Generated for validation")
	if err != nil {
		return err
	}
	modelName := utils.ToModelName(deviceType, version)
	deviceModelYgotPlugin, ok := m.ModelRegistry.ModelPlugins[modelName]
	if !ok {
		log.Warn("No model ", modelName, " available as a plugin")
		if !mgr.allowUnvalidatedConfig {
			return fmt.Errorf("no model %s available as a plugin", modelName)
		}
		return nil
	}

	configValues, err := m.DeviceStateStore.Get(devicetype.NewVersionedID(deviceName, version), lastWrite)
	if err != nil {
		return err
	}

	pathValues := make(map[string]*devicechange.TypedValue)
	for _, configValue := range configValues {
		pathValues[configValue.Path] = configValue.Value
	}
	for _, changeValue := range chg.Values {
		if changeValue.Removed {
			delete(pathValues, changeValue.Path)
		} else {
			pathValues[changeValue.Path] = changeValue.Value
		}
	}

	configValues = make([]*devicechange.PathValue, 0, len(pathValues))
	for path, value := range pathValues {
		configValues = append(configValues, &devicechange.PathValue{
			Path:  path,
			Value: value,
		})
	}
	sort.Slice(configValues, func(i, j int) bool {
		return configValues[i].Path < configValues[j].Path
	})

	jsonTree, err := store.BuildTree(configValues, true)
	if err != nil {
		log.Error("Error building JSON tree from Config Values ", err, jsonTree)
		return err
	}

	ygotModel, err := deviceModelYgotPlugin.UnmarshalConfigValues(jsonTree)
	if err != nil {
		log.Error("Error unmarshaling JSON tree in to YGOT model ", err, string(jsonTree))
		return err
	}
	err = deviceModelYgotPlugin.Validate(ygotModel)
	if err != nil {
		return err
	}
	log.Infof("New Configuration for %s, with version %s and type %s, is Valid according to model %s",
		deviceName, version, deviceType, modelName)

	return nil
}

// SetNetworkConfig creates and stores a new netork config for the given updates and deletes and targets
func (m *Manager) SetNetworkConfig(targetUpdates map[string]devicechange.TypedValueMap,
	targetRemoves map[string][]string, deviceInfo map[devicetype.ID]cache.Info, netChangeID string) (*networkchange.NetworkChange, error) {
	//TODO evaluate need of user and add it back if need be.

	allDeviceChanges, errChanges := m.computeNetworkConfig(targetUpdates, targetRemoves, deviceInfo, "")
	if errChanges != nil {
		return nil, errChanges
	}
	newNetworkConfig, errNetChange := networkchange.NewNetworkChange(netChangeID, allDeviceChanges)
	if errNetChange != nil {
		return nil, errNetChange
	}
	//Writing to the atomix backed store too
	errStoreChange := m.NetworkChangesStore.Create(newNetworkConfig)
	if errStoreChange != nil {
		return nil, errStoreChange
	}
	return newNetworkConfig, nil
}

//computeNetworkConfig computes each device change
func (m *Manager) computeNetworkConfig(targetUpdates map[string]devicechange.TypedValueMap,
	targetRemoves map[string][]string, deviceInfo map[devicetype.ID]cache.Info,
	description string) ([]*devicechange.Change, error) {

	deviceChanges := make([]*devicechange.Change, 0)
	for target, updates := range targetUpdates {
		//FIXME this is a sequential job, not parallelized
		version := deviceInfo[devicetype.ID(target)].Version
		deviceType := deviceInfo[devicetype.ID(target)].Type
		newChange, err := m.ComputeDeviceChange(
			devicetype.ID(target), version, deviceType, updates, targetRemoves[target], description)
		if err != nil {
			log.Error("Error in setting config: ", newChange, " for target ", err)
			continue
		}
		log.Infof("Appending device change %v", newChange)
		deviceChanges = append(deviceChanges, newChange)
		delete(targetRemoves, target)
	}

	// Some targets might only have removes
	for target, removes := range targetRemoves {
		version := deviceInfo[devicetype.ID(target)].Version
		deviceType := deviceInfo[devicetype.ID(target)].Type
		newChange, err := m.ComputeDeviceChange(
			devicetype.ID(target), version, deviceType, make(devicechange.TypedValueMap), removes, description)
		if err != nil {
			log.Error("Error in setting config: ", newChange, " for target ", err)
			continue
		}
		log.Infof("Appending device change %v", newChange)
		deviceChanges = append(deviceChanges, newChange)
	}
	return deviceChanges, nil
}
