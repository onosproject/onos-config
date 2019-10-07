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
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"strconv"
	"strings"
)

const separator = ":"

// ID is a device change identifier type
type ID types.ID

// containsVersion indicates whether a version is present in the change ID
func (i ID) containsVersion() bool {
	return strings.Count(string(i), separator) == 2
}

// GetDeviceID returns the Device ID to which the change is targeted
func (i ID) GetDeviceID() device.ID {
	return device.ID(string(i)[:strings.Index(string(i), separator)])
}

// GetDeviceVersion returns the device version to which the change is targeted
func (i ID) GetDeviceVersion() string {
	if !i.containsVersion() {
		return ""
	}
	return string(i)[strings.Index(string(i), separator)+1 : strings.LastIndex(string(i), separator)]
}

// GetIndex returns the Index
func (i ID) GetIndex() Index {
	index, _ := strconv.Atoi(string(i)[strings.LastIndex(string(i), separator)+1:])
	return Index(index)
}

// Index is the index of a network configuration
type Index uint64

// GetChangeID returns the device change ID for the index given a device/version
func (i Index) GetChangeID(deviceID device.ID, version string) ID {
	if version == "" {
		return ID(fmt.Sprintf("%s%s%d", deviceID, separator, i))
	}
	return ID(fmt.Sprintf("%s%s%s%s%d", deviceID, separator, version, separator, i))
}

// Revision is a network configuration revision number
type Revision types.Revision
