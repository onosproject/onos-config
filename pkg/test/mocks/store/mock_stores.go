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

package store

import networkchanges "github.com/onosproject/onos-config/pkg/store/change/network"

// MockStores is a struct to hold all of the mocked stores for a test
type MockStores struct {
	DeviceStore          *MockDeviceStore
	NetworkChangesStore  networkchanges.Store
	DeviceChangesStore   *MockDeviceChangesStore
	DeviceStateStore     *MockDeviceStateStore
	NetworkSnapshotStore *MockNetworkSnapshotStore
	DeviceSnapshotStore  *MockDeviceSnapshotStore
}
