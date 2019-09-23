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
	"time"
)

//TopoEvent is a topology event
type TopoEvent interface {
	Event
	Device() *device.Device
	ItemAction() EventAction
}

type topoEventObj struct {
	dev        *device.Device
	itemAction EventAction
}

type topoEventImpl struct {
	eventImpl
}

// Address represents the device address
func (topoEvent topoEventImpl) Device() *device.Device {
	to, ok := topoEvent.object.(topoEventObj)
	if ok {
		return to.dev
	}
	return nil
}

func (topoEvent topoEventImpl) ItemAction() EventAction {
	to, ok := topoEvent.object.(topoEventObj)
	if ok {
		return to.itemAction
	}
	return EventItemNone
}

// CreateTopoEvent creates a new topo event object
// It is important not to depend on topocache package here or we will get a
// circular dependency - we take the device.ID and treat it as a string
func CreateTopoEvent(subject device.ID, eventAction EventAction, dev *device.Device) TopoEvent {
	te := topoEventImpl{
		eventImpl: eventImpl{
			subject:   string(subject),
			time:      time.Time{},
			eventType: EventTypeTopoCache,
			object: topoEventObj{
				dev:        dev,
				itemAction: eventAction,
			},
		},
	}
	return &te
}
