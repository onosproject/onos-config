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

package store

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	log "k8s.io/klog"
	"regexp"
	"sort"
	"strings"
	"time"
)

const configurationNamePattern = `^[a-zA-Z0-9\-:_]{4,40}$`
const configurationVersionPattern = `^(\d+\.\d+\.\d+)$`

// ConfigName is an alias for string - is used to qualify identifier for Configuration
type ConfigName string

// Configuration is the connection between a device and Change objects
// The set of ChangeIds define it's content
type Configuration struct {
	Name    ConfigName
	Device  string
	Version string
	Type    string
	Created time.Time
	Updated time.Time
	Changes []change.ID
}

// ExtractFullConfig retrieves the full consolidated config for a Configuration
// This gets the change up to and including the latest
// Use "nBack" to specify a number of changes back to go
// If there are not as many changes in the history as nBack nothing is returned
func (b Configuration) ExtractFullConfig(newChange *change.Change, changeStore map[string]*change.Change, nBack int) []*change.ConfigValue {

	// Have to use a slice to have a consistent output order
	consolidatedConfig := make([]*change.ConfigValue, 0)

	for _, changeID := range b.Changes[0 : len(b.Changes)-nBack] {
		existingChange, ok := changeStore[B64(changeID)]
		if !ok {
			if newChange != nil && B64(newChange.ID) == B64(changeID) {
				existingChange = newChange
			} else {
				log.Error("No existing change with ID ", B64(changeID))
				return nil
			}
		}
		log.Infof("Change desc %s", existingChange.Description)

		for _, changeValue := range existingChange.Config {
			if changeValue.Remove {
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
						consolidatedConfig[idx].Value = changeValue.Value
						alreadyExists = true
						break
					}
				}
				if !alreadyExists {
					copyCv := changeValue.ConfigValue
					consolidatedConfig = append(consolidatedConfig, &copyCv)
				}
			}
		}
	}

	sort.Slice(consolidatedConfig, func(i, j int) bool {
		return consolidatedConfig[i].Path < consolidatedConfig[j].Path
	})

	return consolidatedConfig
}

// CreateConfiguration is a convenient method of creating a Configuration.
// The configuration name is a concatenation of device name and version
// Model data items must be unique and will be sorted. They should not be added
// to afterwards
// The ChangeIDs must unique, and will not be sorted. They can be added afterwards
func CreateConfiguration(deviceName string, version string, deviceType string,
	changes []change.ID) (*Configuration, error) {

	if deviceName == "" || version == "" || deviceType == "" {
		return nil, fmt.Errorf("deviceName, version and deviceType must have values")
	}

	rname := regexp.MustCompile(configurationNamePattern)
	rversion := regexp.MustCompile(configurationVersionPattern)

	if !rname.MatchString(deviceName) || len(deviceName) > 40 {
		return nil, fmt.Errorf("name %s does not match pattern %s",
			deviceName, configurationNamePattern)
	}

	if !rversion.MatchString(version) {
		return nil, fmt.Errorf("version %s does not match pattern %s",
			version, configurationVersionPattern)
	}

	if !rname.MatchString(deviceType) || len(deviceType) > 40 {
		return nil, fmt.Errorf("deviceType %s does not match pattern %s",
			deviceType, configurationNamePattern)
	}

	configName := deviceName + "-" + version

	//Look for duplicates in the changeId - do not sort
	var previousChange string
	for _, c := range changes {
		if previousChange == B64(c) {
			return nil, fmt.Errorf("duplicate last change ID '%s' in config", B64(c))

		}
		previousChange = B64(c)
	}

	deviceConfig := Configuration{
		Name:    ConfigName(configName),
		Device:  deviceName,
		Type:    deviceType,
		Version: version,
		Created: time.Now(),
		Updated: time.Now(),
		Changes: changes,
	}

	return &deviceConfig, nil
}
