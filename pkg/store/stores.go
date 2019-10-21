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
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	"os"
)

// StoreVersion is to check compatibility of a store loaded from file
// Deprecated: do not use
const StoreVersion = "1.0.0"

// StoreTypeChange is for Change stores
// Deprecated: do not use
const StoreTypeChange = "change"

// StoreTypeConfig is for Config stores
// Deprecated: do not use
const StoreTypeConfig = "config"

// StoreTypeNetwork is for Config stores
// Deprecated: do not use
const StoreTypeNetwork = "network"

// ConfigurationStore is the model of the Configuration store
// Deprecated: ConfigurationStore is a legacy implementation of an internal Store
type ConfigurationStore struct {
	Version   string
	Storetype string
	Store     map[ConfigName]Configuration
}

// LoadConfigStore loads the config store from a file
// Deprecated: LoadConfigStore is a method for loading legacy ConfigurationStore from file
func LoadConfigStore(file string) (ConfigurationStore, error) {
	storeFile, err := os.Open(file)
	if err != nil {
		return ConfigurationStore{}, err
	}
	defer storeFile.Close()

	jsonDecoder := json.NewDecoder(storeFile)
	var configStore = ConfigurationStore{}
	_ = jsonDecoder.Decode(&configStore)
	if configStore.Storetype != StoreTypeConfig {
		return ConfigurationStore{},
			fmt.Errorf("Store type invalid: " + configStore.Storetype)
	} else if configStore.Version != StoreVersion {
		return ConfigurationStore{},
			fmt.Errorf("Store version invalid: " + configStore.Version)
	}

	return configStore, nil
}

// RemoveEntry removes a named Configuration
// Deprecated: RemoveEntry is a method for legacy ConfigurationStore
func (s *ConfigurationStore) RemoveEntry(name ConfigName) {
	delete(s.Store, name)
}

// RemoveLastChangeEntry removes a change entry from a named Configuration
// Keeps the configuration even if no changes in that config are present
// Deprecated: RemoveLastChangeEntry is a method for legacy ConfigurationStore
func (s *ConfigurationStore) RemoveLastChangeEntry(name ConfigName) (change.ID, error) {

	changeID := s.Store[name].Changes[len(s.Store[name].Changes)-1]
	newConf, err := NewConfiguration(s.Store[name].Device, s.Store[name].Version, s.Store[name].Type,
		s.Store[name].Changes[:len(s.Store[name].Changes)-1])
	if err != nil {
		return nil, err
	}

	s.Store[name] = *newConf
	return changeID, nil
}

// ChangeStore is the model of the Change store
// Deprecated: ChangeStore is a legacy implementation of an internal Store
type ChangeStore struct {
	// Deprecated
	Version string
	// Deprecated
	Storetype string
	// Deprecated
	Store map[string]*change.Change
}

// LoadChangeStore loads the change store from a file
// Deprecated: LoadConfigStore is a method for loading legacy ChangeStore from file
func LoadChangeStore(file string) (ChangeStore, error) {
	storeFile, err := os.Open(file)
	if err != nil {
		return ChangeStore{}, err
	}
	defer storeFile.Close()

	jsonDecoder := json.NewDecoder(storeFile)
	var changeStore = ChangeStore{}
	_ = jsonDecoder.Decode(&changeStore)
	if changeStore.Storetype != StoreTypeChange {
		return ChangeStore{},
			fmt.Errorf("Store type invalid: " + changeStore.Storetype)
	} else if changeStore.Version != StoreVersion {
		return ChangeStore{},
			fmt.Errorf("Store version invalid: " + changeStore.Version)
	}

	return changeStore, nil
}

// NetworkStore is the model of the Network store
// Deprecated: ChangeStore is a legacy implementation of an internal Store
type NetworkStore struct {
	Version   string
	Storetype string
	Store     []NetworkConfiguration
}

// RemoveEntry removes a named entry from the Network Store
// Deprecated: RemoveEntry is a method for legacy NetworkStore
func (s *NetworkStore) RemoveEntry(name string) error {
	var rmvIdx int
	for idx, entry := range s.Store {
		if entry.Name == name {
			rmvIdx = idx
			break
		}
	}

	s.Store = append(s.Store[:rmvIdx], s.Store[rmvIdx+1:]...)

	return nil
}

// LoadNetworkStore loads the change store from a file
// Deprecated: LoadNetworkStore is a method for loading legacy NetworkStore from file
func LoadNetworkStore(file string) (*NetworkStore, error) {
	storeFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer storeFile.Close()

	jsonDecoder := json.NewDecoder(storeFile)
	var networkStore = NetworkStore{}
	_ = jsonDecoder.Decode(&networkStore)
	if networkStore.Storetype != StoreTypeNetwork {
		return nil,
			fmt.Errorf("Store type invalid: " + networkStore.Storetype)
	} else if networkStore.Version != StoreVersion {
		return nil,
			fmt.Errorf("Store version invalid: " + networkStore.Version)
	}

	return &networkStore, nil
}
