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

package events

import (
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"strconv"
	"time"

	log "k8s.io/klog"
)

//TopoEvent is a topology event
type TopoEvent Event

// Connect represents if the device connected or disconnected
func (topoEvent *TopoEvent) Connect() bool {
	b, err := strconv.ParseBool(topoEvent.values[Connect])
	if err != nil {
		log.Warning("error in conversion", err)
		return false
	}
	return b
}

// Address represents the device address
func (topoEvent *TopoEvent) Address() string {
	return topoEvent.values[Address]
}

// CreateTopoEvent creates a new topo event object
// It is important not to depend on topocache package here or we will get a
// circular dependency - we take the device.ID and treat it as a string
func CreateTopoEvent(subject device.ID, connect bool, address string, dev device.Device) TopoEvent {
	values := make(map[string]string)
	values[Connect] = strconv.FormatBool(connect)
	values[Address] = address
	return TopoEvent{
		subject:   string(subject),
		time:      time.Now(),
		eventtype: EventTypeTopoCache,
		values:    values,
		object:    dev,
	}
}
