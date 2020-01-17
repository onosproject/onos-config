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
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/utils"
)

// GetTargetConfig returns a set of change values given a target, a configuration name, a path and a layer.
// The layer is the numbers of config changes we want to go back in time for. 0 is the latest (Atomix based)
func (m *Manager) GetTargetConfig(deviceID devicetype.ID, version devicetype.Version, path string, revision networkchange.Revision) ([]*devicechange.PathValue, error) {
	log.Infof("Getting config for %s at %s", deviceID, path)
	configValues, errGetTargetCfg := m.DeviceStateStore.Get(devicetype.NewVersionedID(deviceID, version), revision)
	if errGetTargetCfg != nil {
		log.Error("Error while extracting config", errGetTargetCfg)
		return nil, errGetTargetCfg
	}
	if len(configValues) == 0 {
		return configValues, nil
	}
	filteredValues := make([]*devicechange.PathValue, 0)
	pathRegexp := utils.MatchWildcardRegexp(path)
	for _, cv := range configValues {
		if pathRegexp.MatchString(cv.Path) {
			filteredValues = append(filteredValues, cv)
		}
	}
	//TODO if filteredValue is empty return error
	return filteredValues, nil
}

// GetAllDeviceIds returns a list of just DeviceIDs from the device cache
func (m *Manager) GetAllDeviceIds() *[]string {

	var deviceIds = make([]string, 0)
	for _, dev := range m.DeviceCache.GetDevices() {
		deviceIds = append(deviceIds, string(dev.DeviceID))
	}

	return &deviceIds
}
