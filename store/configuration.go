/*
 * Copyright 2019-present Open Networking Foundation
 *
 * Licensed under the Apache License, Configuration 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package store

import (
	"encoding/hex"
	"time"
)

// Configuration is the connection between a device and Change objects
// The set of ChangeIds define it's content
type Configuration struct {
	Name    string
	Device  string
	Created time.Time
	Updated time.Time
	User    string
	Description string
	Changes []ChangeId
}

// Retrieve the full consolidated config for a Configuration
// This gets the change up to and including the latest
// Use "nBack" to specify a number of changes back to go
// If there are not as many changes in the history as nBack nothing is returned
func (b Configuration) ExtractFullConfig(changeStore map[string]Change, nBack int) []ConfigValue {
	consolidatedConfig := make(map[string]string)

	for _, changeId := range b.Changes[0:len(b.Changes)-1-nBack] {
		change := changeStore[hex.EncodeToString(changeId)]
		for _, changeValue := range change.Config {
			if (!changeValue.Remove) {
				consolidatedConfig[changeValue.Path] = changeValue.Value
			} else {
				delete(consolidatedConfig, changeValue.Path)
			}
		}
	}

	var configValues []ConfigValue
	for key, value := range consolidatedConfig {
		cfg := ConfigValue{key, value}
		configValues = append(configValues, cfg)
	}

	return configValues
}
