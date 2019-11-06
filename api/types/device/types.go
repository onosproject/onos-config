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
	topodevice "github.com/onosproject/onos-topo/api/device"
	"strings"
)

const separator = ":"

// Type is a device type
type Type topodevice.Type

// ID is a device ID
type ID topodevice.ID

// Version is a device version
type Version string

// VersionedID is a versioned device ID
type VersionedID types.ID

// NewVersionedID returns a new versioned device identifier
func NewVersionedID(id ID, version Version) VersionedID {
	return VersionedID(fmt.Sprintf("%s%s%s", id, separator, version))
}

// GetID returns the device ID
func (i VersionedID) GetID() ID {
	return ID(i[:strings.Index(string(i), separator)])
}

// GetVersion returns the device version
func (i VersionedID) GetVersion() Version {
	return Version(i[strings.Index(string(i), separator)+1:])
}
