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
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"time"
)

// OperationalStateEvent represents an event for an update in operational state on a device
type OperationalStateEvent interface {
	Event
	ItemAction() EventAction
	Path() string
	Value() *devicechange.TypedValue
}

type operationalStateEventObj struct {
	path       string
	value      *devicechange.TypedValue
	itemAction EventAction
}

type operationalStateEventImpl struct {
	eventImpl
}

func (e operationalStateEventImpl) ItemAction() EventAction {
	oe, ok := e.object.(operationalStateEventObj)
	if ok {
		return oe.itemAction
	}
	return EventItemNone
}

func (e operationalStateEventImpl) Path() string {
	oe, ok := e.object.(operationalStateEventObj)
	if ok {
		return oe.path
	}
	return ""
}

func (e operationalStateEventImpl) Value() *devicechange.TypedValue {
	oe, ok := e.object.(operationalStateEventObj)
	if ok {
		return oe.value
	}
	return nil
}

// NewOperationalStateEvent creates a new operational state event object
func NewOperationalStateEvent(subject string, path string, value *devicechange.TypedValue,
	eventAction EventAction) OperationalStateEvent {
	opStateEvent := operationalStateEventImpl{
		eventImpl: eventImpl{
			subject:   subject,
			time:      time.Now(),
			eventType: EventTypeOperationalState,
			object: operationalStateEventObj{
				itemAction: eventAction,
				path:       path,
				value:      value,
			},
		},
	}
	return opStateEvent
}
