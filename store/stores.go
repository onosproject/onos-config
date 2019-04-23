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
	"encoding/json"
	"os"
)


const STORE_VERSION = "1.0.0"
const STORE_TYPE_CHANGE = "change"
const STORE_TYPE_CONFIG = "config"

type ErrStoreError string

func (e ErrStoreError) Error() string {
	return "Store error:" + string(e)
}


// The configuration store
type ConfigurationStore struct {
	Version string
	Storetype string
	Store map[string]Configuration
}

// Load the config store from a file
func LoadConfigStore(file string) (ConfigurationStore, error) {
	storeFile, err := os.Open(file)
	if err != nil {
		return ConfigurationStore{}, err
	}
	defer storeFile.Close()

	jsonDecoder := json.NewDecoder(storeFile)
	var configStore = ConfigurationStore{}
	jsonDecoder.Decode(&configStore)
	if configStore.Storetype != STORE_TYPE_CONFIG {
		return ConfigurationStore{},
			ErrStoreError("Store type invalid: " + configStore.Storetype)
	} else if configStore.Version != STORE_VERSION {
		return ConfigurationStore{},
			ErrStoreError("Store version invalid: " + configStore.Version)
	}

	return configStore, nil
}

// The change store
type ChangeStore struct {
	Version string
	Storetype string
	Store map[string]Change
}

// Load the change store from a file
func LoadChangeStore(file string) (ChangeStore, error) {
	storeFile, err := os.Open(file)
	if err != nil {
		return ChangeStore{}, err
	}
	defer storeFile.Close()

	jsonDecoder := json.NewDecoder(storeFile)
	var changeStore = ChangeStore{}
	jsonDecoder.Decode(&changeStore)
	if changeStore.Storetype != STORE_TYPE_CHANGE {
		return ChangeStore{},
			ErrStoreError("Store type invalid: " + changeStore.Storetype)
	} else if changeStore.Version != STORE_VERSION {
		return ChangeStore{},
			ErrStoreError("Store version invalid: " + changeStore.Version)
	}

	return changeStore, nil
}

