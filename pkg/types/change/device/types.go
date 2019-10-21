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

package device

import (
	"github.com/onosproject/onos-config/pkg/types"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"strings"
)

const separator = ":"

// ID is a device change identifier type
type ID types.ID

// GetNetworkChangeID returns the NetworkChange ID for the DeviceChange
func (i ID) GetNetworkChangeID() types.ID {
	return types.ID(string(i)[:strings.Index(string(i), separator)])
}

// GetDeviceID returns the Device ID to which the change is targeted
func (i ID) GetDeviceID() devicetopo.ID {
	return devicetopo.ID(string(i)[strings.Index(string(i), separator)+1:])
}

// Index is the index of a network configuration
type Index uint64

// Revision is a network configuration revision number
type Revision types.Revision
