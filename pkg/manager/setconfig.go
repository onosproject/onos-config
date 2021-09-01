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
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"sort"
	"strings"
)

// SetConfigAlreadyApplied is a string constant for "Already applied:"
const SetConfigAlreadyApplied = "Already applied:"

// ValidateNetworkConfig validates the given updates and deletes, according to the path on the configuration
// for the specified target (Atomix Based)
func (m *Manager) ValidateNetworkConfig(deviceName devicetype.ID, version devicetype.Version,
	deviceType devicetype.Type, updates devicechange.TypedValueMap, deletes []string, lastWrite networkchange.Revision) error {

	modelName := utils.ToModelName(deviceType, version)
	deviceModelYgotPlugin, err := m.ModelRegistry.GetPlugin(modelName)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Warn("No model ", modelName, " available as a plugin")
			if !mgr.allowUnvalidatedConfig {
				return fmt.Errorf("no model %s available as a plugin", modelName)
			}
			return nil
		}
		return err
	}

	configValues, err := m.DeviceStateStore.Get(devicetype.NewVersionedID(deviceName, version), lastWrite)
	if err != nil {
		return err
	}

	pathValues := make(devicechange.TypedValueMap)
	// Feed the deviceChange store contents in to map
	for _, configValue := range configValues {
		pathValues[configValue.Path] = configValue.Value
	}
	// then overlay with updates
	for changePath, changeValue := range updates {
		if len(changeValue.GetBytes()) == 0 &&
			(changeValue.GetType() == devicechange.ValueType_STRING ||
				changeValue.GetType() == devicechange.ValueType_BYTES) {
			return errors.NewInvalid("Empty string not allowed. Delete attribute instead. %s", changePath)
		}
		pathValues[changePath] = changeValue
	}
	// finally remove any deletes and children of the deleted
	for _, deletePath := range deletes {
		deletePathAnonIdx := modelregistry.AnonymizePathIndices(deletePath)
		if strings.HasSuffix(deletePathAnonIdx, "]") {
			deletePath = modelregistry.AddMissingIndexName(deletePath)[0]
			deletePathAnonIdx = modelregistry.AnonymizePathIndices(deletePath)
		}
		modelEntry, ok := deviceModelYgotPlugin.ReadWritePaths[deletePathAnonIdx]
		if ok && modelEntry.IsAKey { // Then delete all children
			deletePathRoot := deletePath[:strings.LastIndex(deletePath, "/")]
			for path := range pathValues {
				if strings.HasPrefix(path, deletePathRoot) {
					delete(pathValues, path)
				}
			}
		} else if _, exactPathValue := pathValues[deletePathAnonIdx]; exactPathValue {
			delete(pathValues, deletePath)
		} else { // else delete anything matching prefix
			for pathValue := range pathValues {
				if strings.HasPrefix(pathValue, deletePathAnonIdx) {
					delete(pathValues, pathValue)
				}
			}
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

	ygotModel, err := deviceModelYgotPlugin.Model.Unmarshaler()(jsonTree)
	if err != nil {
		log.Infof("Unmarshalling during validation failed. JSON tree %v", jsonTree)
		return fmt.Errorf("unmarshaller error: %v", err)
	}
	err = deviceModelYgotPlugin.Model.Validator()(ygotModel)
	if err != nil {
		return fmt.Errorf("validation error %s", err.Error())
	}
	log.Infof("New Configuration for %s, with version %s and type %s, is Valid according to model %s",
		deviceName, version, deviceType, modelName)

	return nil
}

// SetNetworkConfig creates and stores a new netork config for the given updates and deletes and targets
func (m *Manager) SetNetworkConfig(targetUpdates map[devicetype.ID]devicechange.TypedValueMap,
	targetRemoves map[devicetype.ID][]string, deviceInfo map[devicetype.ID]cache.Info, netChangeID string) (*networkchange.NetworkChange, error) {
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
func (m *Manager) computeNetworkConfig(targetUpdates map[devicetype.ID]devicechange.TypedValueMap,
	targetRemoves map[devicetype.ID][]string, deviceInfo map[devicetype.ID]cache.Info,
	description string) ([]*devicechange.Change, error) {

	deviceChanges := make([]*devicechange.Change, 0)
	for target, updates := range targetUpdates {
		//FIXME this is a sequential job, not parallelized
		version := deviceInfo[target].Version
		deviceType := deviceInfo[target].Type
		newChange, err := computeDeviceChange(
			target, version, deviceType, updates, targetRemoves[target], description)
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
		version := deviceInfo[target].Version
		deviceType := deviceInfo[target].Type
		newChange, err := computeDeviceChange(
			target, version, deviceType, make(devicechange.TypedValueMap), removes, description)
		if err != nil {
			log.Error("Error in setting config: ", newChange, " for target ", err)
			continue
		}
		log.Infof("Appending device change %v", newChange)
		deviceChanges = append(deviceChanges, newChange)
	}
	return deviceChanges, nil
}

// computeDeviceChange computes a given device change the given updates and deletes, according to the path
// on the configuration for the specified target
func computeDeviceChange(deviceName devicetype.ID, version devicetype.Version,
	deviceType devicetype.Type, updates devicechange.TypedValueMap,
	deletes []string, description string) (*devicechange.Change, error) {

	var newChanges = make([]*devicechange.ChangeValue, 0)
	//updates
	for path, value := range updates {
		updateValue, err := devicechange.NewChangeValue(path, value, false)
		if err != nil {
			log.Warnf("Error creating value for %s %v", path, err)
			// TODO: this should return an error
			continue
		}
		newChanges = append(newChanges, updateValue)
	}
	//deletes
	for _, path := range deletes {
		deleteValue, _ := devicechange.NewChangeValue(path, devicechange.NewTypedValueEmpty(), true)
		newChanges = append(newChanges, deleteValue)
	}
	//description := fmt.Sprintf("Originally created as part of %s", description)
	//if description == "" {
	//	description = fmt.Sprintf("Created at %s", time.Now().Format(time.RFC3339))
	//}
	//TODO lost description of Change
	changeElement := &devicechange.Change{
		DeviceID:      deviceName,
		DeviceVersion: version,
		DeviceType:    deviceType,
		Values:        newChanges,
	}

	return changeElement, nil
}
