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

package network

import (
	"fmt"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"regexp"
	"time"
)

// NewNetworkChange creates a new network configuration
func NewNetworkChange(networkChangeID string, changes []*devicechange.Change, existingDeviceConfig bool) (*NetworkChange, error) {
	r1 := regexp.MustCompile(`[a-zA-Z0-9\-_]+`)
	match := r1.FindString(networkChangeID)
	if networkChangeID == "" {
		return nil, fmt.Errorf("Empty name not allowed")
	} else if networkChangeID != match {
		return nil, fmt.Errorf("Error in name %s", networkChangeID)
	}

	return &NetworkChange{
		ID:                   ID(networkChangeID),
		Created:              time.Now(),
		Updated:              time.Now(),
		Changes:              changes,
		ExistingDeviceConfig: existingDeviceConfig,
	}, nil
}
