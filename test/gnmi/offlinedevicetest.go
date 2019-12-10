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

package gnmi

import (
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	modPath           = "/system/clock/config/timezone-name"
	offlineDeviceName = "offline-dev-1"
)

// TestOfflineDevice tests set/query of a single GNMI path to a single device that is initially not in the config
func (s *TestSuite) TestOfflineDevice(t *testing.T) {
	simulator := env.NewSimulator().SetName(offlineDeviceName)

	// Make a GNMI client to use for requests
	c, err := env.Config().NewGNMIClient()
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	// Set a value using gNMI client to the offline device
	extNameDeviceType := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionDeviceType,
			Msg: []byte("Devicesim"),
		},
	}
	extNameDeviceVersion := gnmi_ext.Extension_RegisteredExt{
		RegisteredExt: &gnmi_ext.RegisteredExtension{
			Id:  gnmi.GnmiExtensionVersion,
			Msg: []byte("1.0.0"),
		},
	}

	extensions := []*gnmi_ext.Extension{{Ext: &extNameDeviceType}, {Ext: &extNameDeviceVersion}}

	setPath := makeDevicePath(offlineDeviceName, tzPath)
	setPath[0].pathDataValue = tzValue
	setPath[0].pathDataType = StringVal

	go func() {
		_, extensions, errorSet := gNMISet(MakeContext(), c, setPath, noPaths, extensions)
		assert.NoError(t, errorSet)
		assert.Equal(t, 2, len(extensions))
		extensionBefore := extensions[0].GetRegisteredExt()
		assert.Equal(t, extensionBefore.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))
	}()

	// Check that the value was set correctly
	simulator.AddOrDie()
	time.Sleep(5 * time.Second)

	valueAfter, extensions, errorAfter := gNMIGet(MakeContext(), c, makeDevicePath(offlineDeviceName, modPath))
	assert.NoError(t, errorAfter)
	assert.Equal(t, 0, len(extensions))
	assert.NotEqual(t, "", valueAfter, "Query after set returned an error: %s\n", errorAfter)
	assert.Equal(t, tzValue, valueAfter[0].pathDataValue, "Query after set returned the wrong value: %s\n", valueAfter)

	// Remove the path we added
	_, extensions, errorDelete := gNMISet(MakeContext(), c, noPaths, makeDevicePath(offlineDeviceName, modPath), noExtensions)
	assert.NoError(t, errorDelete)
	assert.Equal(t, 1, len(extensions))
	extensionAfter := extensions[0].GetRegisteredExt()
	assert.Equal(t, extensionAfter.Id.String(), strconv.Itoa(gnmi.GnmiExtensionNetwkChangeID))

	//  Make sure it got removed
	valueAfterDelete, extensions, errorAfterDelete := gNMIGet(MakeContext(), c, makeDevicePath(offlineDeviceName, modPath))
	assert.NoError(t, errorAfterDelete)
	assert.Equal(t, 0, len(extensions))
	assert.Equal(t, valueAfterDelete[0].pathDataValue, "",
		"incorrect value found for path %s after delete", modPath)
}
