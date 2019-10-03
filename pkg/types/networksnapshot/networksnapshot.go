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

package networksnapshot

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
)

// ID is a snapshot request identifier
type ID types.ID

// Index is the index of a snapshot request
type Index types.Index

// Revision is a snapshot revision number
type Revision types.Revision

// GetDeviceSnapshotID returns the device snapshot ID for the given device
func (s *NetworkSnapshot) GetDeviceSnapshotID(deviceID device.ID) types.ID {
	return types.ID(fmt.Sprintf("%s-%s", s.ID, deviceID))
}

// GetDeviceSnapshots returns a list of device snapshot IDs
func (s *NetworkSnapshot) GetDeviceSnapshots() []types.ID {
	snapshotIDs := make([]types.ID, len(s.Devices))
	for i, deviceID := range s.Devices {
		snapshotIDs[i] = s.GetDeviceSnapshotID(deviceID)
	}
	return snapshotIDs
}
