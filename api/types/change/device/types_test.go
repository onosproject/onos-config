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
//

package device

import (
	"github.com/onosproject/onos-config/api/types"
	"github.com/onosproject/onos-config/api/types/device"
	"gotest.tools/assert"
	"strings"
	"testing"
)

func Test_NewVersionedId(t *testing.T) {
	netChangeID := types.ID("NetChange33")
	deviceID := device.ID("Device33")
	deviceVersion := device.Version("3.33.333")
	id := NewID(netChangeID, deviceID, deviceVersion)
	assert.Equal(t, id.GetDeviceID(), deviceID)
	assert.Equal(t, id.GetDeviceVersion(), deviceVersion)
	assert.Equal(t, id.GetNetworkChangeID(), netChangeID)

	deviceVersionedID := id.GetDeviceVersionedID()
	assert.Equal(t, deviceVersionedID.GetVersion(), deviceVersion)
	assert.Equal(t, deviceVersionedID.GetID(), deviceID)

	change := Change{
		DeviceID:      "device",
		DeviceVersion: "1.0",
	}

	versionedDeviceID := string(change.GetVersionedDeviceID())
	assert.Assert(t, strings.Contains(versionedDeviceID, "device"))
	assert.Assert(t, strings.Contains(versionedDeviceID, "1.0"))
}
