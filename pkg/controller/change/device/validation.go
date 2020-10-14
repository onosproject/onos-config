// Copyright 2020-present Open Networking Foundation.
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
	"sort"

	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/utils"
)

func (r *Reconciler) validateDeviceChange(change *devicechange.DeviceChange) error {

	deviceType := change.Change.DeviceType
	version := change.Change.DeviceVersion
	deviceID := change.Change.DeviceID

	changeValues := change.Change.Values

	modelName := utils.ToModelName(deviceType, version)
	deviceModelYgotPlugin, ok := r.modelReg.ModelPlugins[modelName]
	if !ok {
		log.Warn("No model ", modelName, " available as a plugin")
		return fmt.Errorf("no model %s available as a plugin", modelName)
	}

	pathValues := make(map[string]*devicechange.TypedValue)
	for _, configValue := range changeValues {
		pathValues[configValue.Path] = configValue.Value
	}
	for _, changeValue := range changeValues {
		if changeValue.Removed {
			delete(pathValues, changeValue.Path)
		} else {
			pathValues[changeValue.Path] = changeValue.Value
		}
	}

	configValues := make([]*devicechange.PathValue, 0, len(pathValues))
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
		deviceID, version, deviceType, modelName)

	return nil

}
