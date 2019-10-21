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
	"github.com/onosproject/onos-config/pkg/types"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
)

const separator = ":"

// ID is a network configuration identifier type
type ID types.ID

// GetDeviceChangeID returns a device change ID for the given device ID
func (i ID) GetDeviceChangeID(deviceID devicetopo.ID) devicechangetypes.ID {
	return devicechangetypes.ID(fmt.Sprintf("%s%s%s", i, separator, deviceID))
}

// Index is the index of a network configuration
type Index uint64

// Revision is a network configuration revision number
type Revision types.Revision
