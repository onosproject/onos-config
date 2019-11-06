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
	"github.com/onosproject/onos-config/api/types"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/controller"
)

// Partitioner is a WorkPartitioner for devices
type Partitioner struct {
}

// Partition returns the device as a partition key
func (p *Partitioner) Partition(id types.ID) (controller.PartitionKey, error) {
	return controller.PartitionKey(devicechange.ID(id).GetDeviceID()), nil
}

var _ controller.WorkPartitioner = &Partitioner{}
