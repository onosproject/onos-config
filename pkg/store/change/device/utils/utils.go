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

package utils

import (
	"sort"
	"strings"

	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/api/types/device"
	devicechangestore "github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("store", "change", "device")

// ExtractFullConfig retrieves the full consolidated config for a Configuration
// This gets the change up to and including the latest
// Use "nBack" to specify a number of changes back to go
// If there are not as many changes in the history as nBack nothing is returned
func ExtractFullConfig(deviceID device.VersionedID, newChange *devicechange.Change, changeStore devicechangestore.Store,
	nBack int) ([]*devicechange.PathValue, error) {

	// Have to use a slice to have a consistent output order
	consolidatedConfig := make([]*devicechange.PathValue, 0)

	changeChan := make(chan *devicechange.DeviceChange)

	ctx, err := changeStore.List(deviceID, changeChan)

	if err != nil {
		return nil, err
	}

	defer ctx.Close()

	if newChange != nil {
		consolidatedConfig = getPathValue(newChange, consolidatedConfig)
	}

	if nBack == 0 {
		for storeChange := range changeChan {
			if storeChange.Status.Phase == changetypes.Phase_CHANGE {
				consolidatedConfig = getPathValue(storeChange.Change, consolidatedConfig)
			}
		}
	} else {
		changes := make([]*devicechange.DeviceChange, 0)
		for storeChange := range changeChan {
			if storeChange.Status.Phase == changetypes.Phase_CHANGE {
				changes = append(changes, storeChange)
			}
		}
		end := len(changes) - nBack
		for _, storeChange := range changes[0:end] {
			consolidatedConfig = getPathValue(storeChange.Change, consolidatedConfig)
		}
	}

	sort.Slice(consolidatedConfig, func(i, j int) bool {
		return consolidatedConfig[i].Path < consolidatedConfig[j].Path
	})

	return consolidatedConfig, nil
}

func getPathValue(storeChange *devicechange.Change, consolidatedConfig []*devicechange.PathValue) []*devicechange.PathValue {
	for _, changeValue := range storeChange.Values {
		if changeValue.Removed {
			// Delete everything at that path and all below it
			// Have to search through consolidated config
			// Make a list of indices to remove
			indices := make([]int, 0)
			for idx, cce := range consolidatedConfig {
				if strings.Contains(cce.Path, changeValue.Path) {
					indices = append(indices, idx)
				}
			}
			// Remove in reverse
			for i := len(indices) - 1; i >= 0; i-- {
				consolidatedConfig = append(consolidatedConfig[:indices[i]], consolidatedConfig[indices[i]+1:]...)
			}

		} else {
			var alreadyExists bool
			for idx, cv := range consolidatedConfig {
				if changeValue.Path == cv.Path {
					consolidatedConfig[idx].Value = changeValue.GetValue()
					alreadyExists = true
					break
				}
			}
			if !alreadyExists {
				copyCv := devicechange.PathValue{
					Path:  changeValue.GetPath(),
					Value: changeValue.GetValue(),
				}
				consolidatedConfig = append(consolidatedConfig, &copyCv)
			}
		}
	}
	return consolidatedConfig
}
