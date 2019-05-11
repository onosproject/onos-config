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

func Test_LocalDeviceStore(t *testing.T) {
	topoChannel := make(chan events.Event, 10)
	deviceStore, err := LoadDeviceStore("testdata/deviceStore.json", topoChannel)
	assert.NilError(t, err, "Cannot load device store ")
	assert.Assert(t, deviceStore != nil)

	assert.Equal(t, deviceStore.Version, "1.0.0", "Device store loaded wrong version string")

	d1, d1exists := deviceStore.Store["localhost:10161"]
	assert.Assert(t, d1exists)
	assert.Equal(t, d1.Addr, "localhost:10161")
}

func Test_BadDeviceStore(t *testing.T) {
	topoChannel := make(chan events.Event, 10)
	_, err := LoadDeviceStore("testdata/badDeviceStore.json", topoChannel)
	assert.ErrorContains(t, err, "Store type invalid")
}

func Test_NonexistentDeviceStore(t *testing.T) {
	topoChannel := make(chan events.Event, 10)
	_, err := LoadDeviceStore("testdata/noSuchDeviceStore.json", topoChannel)
	assert.ErrorContains(t, err, "no such file")
}