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
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	"io"
	"regexp"
	"time"
)

// NetworkConfiguration is a model of a network defined configuration
// change. It is a combination of changes to devices (configurations)
type NetworkConfiguration struct {
	Name                 string
	Created              time.Time
	User                 string
	ConfigurationChanges map[ConfigName]change.ID
}

// CreateNetworkStoreWithName creates a NetworkConfiguration object
func CreateNetworkStoreWithName(name string, user string) (*NetworkConfiguration, error) {
	r1 := regexp.MustCompile(`[a-zA-Z0-9\-_]+`)
	match := r1.FindString(name)
	if name != match {
		return nil, fmt.Errorf("Error in name %s", name)
	}

	return &NetworkConfiguration{
		Name:                 name,
		Created:              time.Now(),
		User:                 user,
		ConfigurationChanges: make(map[ConfigName]change.ID),
	}, nil
}

// CreateNetworkStore creates a NetworkConfiguration object
func CreateNetworkStore(user string, changes map[ConfigName]change.ID) (*NetworkConfiguration, error) {

	nwConf := NetworkConfiguration{
		Name:                 "temp",
		Created:              time.Now(),
		User:                 user,
		ConfigurationChanges: changes,
	}

	_, err := nwConf.ChangeNameToHash()
	if err != nil {
		return nil, err
	}
	return &nwConf, nil
}

// ChangeNameToHash changes the name of a NetworkConfiguration to a hash of its contents
func (n *NetworkConfiguration) ChangeNameToHash() (*string, error) {
	h := sha1.New()

	// Calculate a hash from the config, description and timestamp
	jsonstr, _ := json.Marshal(n.ConfigurationChanges)
	_, err1 := io.WriteString(h, string(jsonstr))
	if err1 != nil {
		return nil, err1
	}

	hash := h.Sum(nil)
	n.Name = fmt.Sprintf("Change-%s", B64(hash))
	return &n.Name, nil
}
