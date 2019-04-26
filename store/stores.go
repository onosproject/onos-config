// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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
	"encoding/json"
	"fmt"
	"os"
)

// StoreVersion is to check compatibility of a store loaded from file
const StoreVersion = "1.0.0"

// StoreTypeChange is for Change stores
const StoreTypeChange = "change"

// StoreTypeConfig is for Config stores
const StoreTypeConfig = "config"

// ConfigurationStore is the model of the Configuration store
type ConfigurationStore struct {
	Version   string
	Storetype string
	Store     map[string]Configuration
}

// LoadConfigStore loads the config store from a file
func LoadConfigStore(file string) (ConfigurationStore, error) {
	storeFile, err := os.Open(file)
	if err != nil {
		return ConfigurationStore{}, err
	}
	defer storeFile.Close()

	jsonDecoder := json.NewDecoder(storeFile)
	var configStore = ConfigurationStore{}
	jsonDecoder.Decode(&configStore)
	if configStore.Storetype != StoreTypeConfig {
		return ConfigurationStore{},
			fmt.Errorf("Store type invalid: " + configStore.Storetype)
	} else if configStore.Version != StoreVersion {
		return ConfigurationStore{},
			fmt.Errorf("Store version invalid: " + configStore.Version)
	}

	return configStore, nil
}

// ChangeStore is the model of the Change store
type ChangeStore struct {
	Version   string
	Storetype string
	Store     map[string]*Change
}

// LoadChangeStore loads the change store from a file
func LoadChangeStore(file string) (ChangeStore, error) {
	storeFile, err := os.Open(file)
	if err != nil {
		return ChangeStore{}, err
	}
	defer storeFile.Close()

	jsonDecoder := json.NewDecoder(storeFile)
	var changeStore = ChangeStore{}
	jsonDecoder.Decode(&changeStore)
	if changeStore.Storetype != StoreTypeChange {
		return ChangeStore{},
			fmt.Errorf("Store type invalid: " + changeStore.Storetype)
	} else if changeStore.Version != StoreVersion {
		return ChangeStore{},
			fmt.Errorf("Store version invalid: " + changeStore.Version)
	}

	return changeStore, nil
}
