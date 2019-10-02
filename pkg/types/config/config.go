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

package config

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
)

// ID is a network configuration identifier type
type ID types.ID

// Index is the index of a network configuration
type Index uint64

// Revision is a network configuration revision number
type Revision types.Revision

// GetChangeIDs returns a list of change IDs for the network
func (c *NetworkConfig) GetChangeIDs() []change.ID {
	changes := c.GetChanges()
	if changes == nil {
		return nil
	}

	ids := make([]change.ID, len(changes))
	for i, change := range changes {
		ids[i] = c.GetChangeID(change.DeviceID)
	}
	return ids
}

// GetChangeID returns the ID for the change to the given device
func (c *NetworkConfig) GetChangeID(deviceID device.ID) change.ID {
	return change.ID(fmt.Sprintf("%s-%s", c.ID, deviceID))
}
