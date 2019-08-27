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
	"github.com/onosproject/onos-config/pkg/utils"
	log "k8s.io/klog"
	"sort"
)

// GetTargetConfig returns a set of change values given a target, a configuration name, a path and a layer.
// The layer is the numbers of config changes we want to go back in time for. 0 is the latest
func (m *Manager) GetTargetConfig(target string, configname store.ConfigName, path string, layer int) ([]*change.ConfigValue, error) {
	log.Info("Getting config for ", target, path)
	//TODO the key of the config store should be a tuple of (devicename, configname) use the param
	var config store.Configuration
	if target != "" {
		for _, cfg := range m.ConfigStore.Store {
			if cfg.Device == target {
				config = cfg
				break
			}
		}
		if config.Name == "" {
			return make([]*change.ConfigValue, 0),
				fmt.Errorf("no Configuration found for %s", target)
		}
	} else if configname != "" {
		config = m.ConfigStore.Store[configname]
		if config.Name == "" {
			return make([]*change.ConfigValue, 0),
				fmt.Errorf("no Configuration found for %s", configname)
		}
	}
	configValues := config.ExtractFullConfig(nil, m.ChangeStore.Store, layer)
	if len(configValues) == 0 {
		return configValues, nil
	}
	filteredValues := make([]*change.ConfigValue, 0)
	pathRegexp := utils.MatchWildcardRegexp(path)
	for _, cv := range configValues {
		if pathRegexp.MatchString(cv.Path) {
			filteredValues = append(filteredValues, cv)
		}
	}
	//TODO if filteredValue is empty return error
	return filteredValues, nil
}

// GetAllDeviceIds returns a list of just DeviceIDs from the Config store
func (m *Manager) GetAllDeviceIds() *[]string {
	var deviceIds = make([]string, 0)

	for _, v := range m.ConfigStore.Store {
		deviceIds = append(deviceIds, v.Device+" ("+v.Version+")")
	}
	sort.Strings(deviceIds)

	return &deviceIds
}
