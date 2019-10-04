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
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"strings"
)

// ID is a snapshot identifier type
type ID types.ID

// GetDeviceID returns the Device ID
func (i ID) GetDeviceID() device.ID {
	return device.ID(string(i)[:strings.LastIndex(string(i), "-")])
}

// Revision is a network configuration revision number
type Revision types.Revision
