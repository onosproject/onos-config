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

package change

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/types"
	devicechange "github.com/onosproject/onos-config/pkg/types/device/change"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"strconv"
	"strings"
)

// ID is a network configuration identifier type
type ID types.ID

// GetIndex returns the Index
func (i ID) GetIndex() Index {
	index, _ := strconv.Atoi(strings.Split(string(i), "-")[1])
	return Index(index)
}

// Index is the index of a network configuration
type Index uint64

// GetID returns the ID for the index
func (i Index) GetID() ID {
	return ID(fmt.Sprintf("network-%d", i))
}

// Revision is a network configuration revision number
type Revision types.Revision

// GetChangeIDs returns a list of change IDs for the network
func (c *NetworkChange) GetChangeIDs() []devicechange.ID {
	changes := c.GetChanges()
	if changes == nil {
		return nil
	}

	ids := make([]devicechange.ID, len(changes))
	for i, change := range changes {
		ids[i] = c.GetChangeID(change.DeviceID)
	}
	return ids
}

// GetChangeID returns the ID for the change to the given device
func (c *NetworkChange) GetChangeID(deviceID device.ID) devicechange.ID {
	return devicechange.NewID(types.ID(c.ID), deviceID)
}
