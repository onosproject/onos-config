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

package topocache

import (
	"github.com/onosproject/onos-config/pkg/events"
	"testing"
)
import (
	"gotest.tools/assert"
)

func loadDeviceStore(t *testing.T) *DeviceStore {
	topoChannel := make(chan events.TopoEvent, 10)
	deviceStore, err := LoadDeviceStore("testdata/deviceStore.json", topoChannel)
	assert.NilError(t, err, "Cannot load device store ")
	assert.Assert(t, deviceStore != nil)
	return deviceStore
}

func Test_LocalDeviceStore(t *testing.T) {
	deviceStore := loadDeviceStore(t)
	assert.Equal(t, deviceStore.Version, "1.0.0", "Device store loaded wrong version string")

	d1, d1exists := deviceStore.Store["localhost-1"]
	assert.Assert(t, d1exists)
	assert.Equal(t, d1.Addr, "localhost:10161")
}

func Test_BadDeviceStore(t *testing.T) {
	topoChannel := make(chan events.TopoEvent, 10)
	_, err := LoadDeviceStore("testdata/badDeviceStore.json", topoChannel)
	assert.ErrorContains(t, err, "Store type invalid")
}

func Test_DeviceStoreNoAddr(t *testing.T) {
	_, err := LoadDeviceStore("testdata/deviceStoreNoAddr.json", nil)
	assert.ErrorContains(t, err, "has blank address")
}

func Test_DeviceStoreNoSwVer(t *testing.T) {
	_, err := LoadDeviceStore("testdata/deviceStoreNoVersion.json", nil)
	assert.ErrorContains(t, err, "has blank software version")
}

func Test_NonexistentDeviceStore(t *testing.T) {
	topoChannel := make(chan events.TopoEvent, 10)
	_, err := LoadDeviceStore("testdata/noSuchDeviceStore.json", topoChannel)
	assert.ErrorContains(t, err, "no such file")
}

// In the case of a duplicate it cannot be determined which will overwrite the
// other as the file contents are loaded in to a map
func Test_DeviceStoreDuplicates(t *testing.T) {
	topoChannel := make(chan events.TopoEvent, 10)
	deviceStore, err := LoadDeviceStore("testdata/deviceStoreDuplicate.json", topoChannel)
	assert.NilError(t, err)
	lh2 := deviceStore.Store["localhost-2"]
	assert.Assert(t, lh2.SoftwareVersion == "1.0.0" || lh2.SoftwareVersion == "2.0.0")
}

func Test_AddDevice(t *testing.T) {
	deviceStore := loadDeviceStore(t)
	deviceStore.AddDevice("foobar", Device{Addr: "foobar:123", SoftwareVersion: "1.0"})
	d, ok := deviceStore.Store["foobar"]
	assert.Assert(t, ok, "device not added")
	assert.Assert(t, d.Addr == "foobar:123", "wrong device added")
}

func Test_AddBadDevice(t *testing.T) {
	deviceStore := loadDeviceStore(t)
	err := deviceStore.AddDevice("foobar", Device{Addr: "foobar:123"})
	assert.Assert(t, err != nil, "device without version should not be added")
}

func Test_DeleteDevice(t *testing.T) {
	deviceStore := loadDeviceStore(t)
	deviceStore.RemoveDevice("localhost-2")
	_, ok := deviceStore.Store["localhost-2"]
	assert.Assert(t, !ok, "device not removed")
}
