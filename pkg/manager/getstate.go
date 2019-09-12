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
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
)

// GetTargetState returns a set of state values given a target and a path.
func (m *Manager) GetTargetState(target string, path string) []*change.ConfigValue {
	log.Info("Getting State for ", target, path)
	configValues := make([]*change.ConfigValue, 0)
	//First check the cache, if it's not empty for this path we read that and return,
	pathRegexp := utils.MatchWildcardRegexp(path)
	for pathCache, value := range m.OperationalStateCache[device.ID(target)] {
		if pathRegexp.MatchString(pathCache) {
			configValues = append(configValues, &change.ConfigValue{
				Path:       pathCache,
				TypedValue: *value,
			})
		}
	}
	if len(configValues) == 0 {
		log.Warningf("Path %s is not in the operational state cache of device %s", path, target)
	}
	return configValues
}
