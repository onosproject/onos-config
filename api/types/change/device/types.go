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
	"fmt"
	"github.com/onosproject/onos-config/api/types"
	"github.com/onosproject/onos-config/api/types/device"
	"strings"
)

const separator = ":"

// ID is a device change identifier type
type ID types.ID

// NewID return sa new device change identifier
func NewID(netChangeID types.ID, deviceID device.ID, deviceVersion device.Version) ID {
	return ID(fmt.Sprintf("%s%s%s%s%s", netChangeID, separator, deviceID, separator, deviceVersion))
}

// GetNetworkChangeID returns the NetworkChange ID for the DeviceChange
func (i ID) GetNetworkChangeID() types.ID {
	return types.ID(i[:strings.Index(string(i), separator)])
}

// GetDeviceID returns the Device ID to which the change is targeted
func (i ID) GetDeviceID() device.ID {
	return device.ID(i[strings.Index(string(i), separator)+1 : strings.LastIndex(string(i), separator)])
}

// GetDeviceVersion returns the device version to which the change is targeted
func (i ID) GetDeviceVersion() device.Version {
	return device.Version(i[strings.LastIndex(string(i), separator)+1:])
}

// GetDeviceVersionedID returns the VersionedID for the device to which the change is targeted
func (i ID) GetDeviceVersionedID() device.VersionedID {
	return device.NewVersionedID(i.GetDeviceID(), i.GetDeviceVersion())
}

// Index is the index of a network configuration
type Index uint64

// Revision is a network configuration revision number
type Revision types.Revision

// GetVersionedDeviceID returns the device VersionedID for the change
func (c *Change) GetVersionedDeviceID() device.VersionedID {
	return device.NewVersionedID(c.DeviceID, c.DeviceVersion)
}
