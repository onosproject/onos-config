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

package snapshot

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/types"
	devicesnapshot "github.com/onosproject/onos-config/pkg/types/device/snapshot"
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
	return ID(fmt.Sprintf("network-snapshot-%d", i))
}

// Revision is a network configuration revision number
type Revision types.Revision

// GetDeviceSnapshotID gets the device snapshot ID for the given device
func (s *NetworkSnapshot) GetDeviceSnapshotID(deviceID device.ID) devicesnapshot.ID {
	return devicesnapshot.ID(fmt.Sprintf("device-snapshot-%d-%s", s.Index, deviceID))
}

// GetDeviceSnapshotIDs returns a list of snapshot IDs for the devices in the snapshot
func (s *NetworkSnapshot) GetDeviceSnapshotIDs() []devicesnapshot.ID {
	snapshotIDs := make([]devicesnapshot.ID, len(s.Devices))
	for i, device := range s.Devices {
		snapshotIDs[i] = s.GetDeviceSnapshotID(device)
	}
	return snapshotIDs
}
