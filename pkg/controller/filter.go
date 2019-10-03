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

package controller

import (
	mastershipstore "github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-config/pkg/types"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
)

// Filter filters individual events for a node
type Filter interface {
	// Accept indicates whether to accept the given object
	Accept(id types.ID) bool
}

// DeviceResolver resolves a device from a type ID
type DeviceResolver interface {
	// Resolve resolves a device
	Resolve(id types.ID) (device.ID, error)
}

// MastershipFilter activates a controller on acquiring mastership
type MastershipFilter struct {
	Store    mastershipstore.Store
	Resolver DeviceResolver
}

// Accept accepts the given ID if the local node is the master
func (f *MastershipFilter) Accept(id types.ID) bool {
	device, err := f.Resolver.Resolve(id)
	if err != nil {
		return false
	}
	master, err := f.Store.IsMaster(device)
	if err != nil {
		return false
	}
	return master
}

var _ Filter = &MastershipFilter{}
