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

/*
Package events is a general definition of an event type to be passed through channels.
*/
package events

import (
	"fmt"
	"time"
)

// EventType is an enumerated type
type EventType int

// Values of the EventType enumeration
const ( // For event types
	EventTypeOperationalState EventType = iota
	EventTypeDeviceConnected
	EventTypeErrorParseConfig
	EventTypeErrorDeviceConnect
	EventTypeErrorDeviceCapabilities
	EventTypeErrorDeviceConnectInitialConfigSync
	EventTypeErrorDeviceDisconnect
	EventTypeErrorSubscribe
	EventTypeErrorMissingModelPlugin
	EventTypeErrorTranslation
	EventTypeErrorGetWithRoPaths
	EventTypeTopoUpdate
)

// EventAction is an enumerated type
type EventAction int

// Values of the EventItem enumeration
const (
	EventItemNone EventAction = iota
	EventItemAdded
	EventItemUpdated
	EventItemDeleted
)

func (et EventType) String() string {
	return [...]string{"EventOperationalState", "EventTypeDeviceConnected",
		"EventTypeErrorParseConfig", "EventTypeErrorDeviceConnect",
		"EventTypeErrorDeviceCapabilities", "EventTypeErrorDeviceConnectInitialConfigSync",
		"EventTypeErrorDeviceDisconnect",
		"EventTypeErrorSubscribe, EventTypeErrorMissingModelPlugin, EventTypeErrorTranslation",
		"EventTypeErrorGetWithRoPaths", "EventTypeTopoUpdate"}[et]
}

// Event is a general purpose base type of event
// Specializations of Event are possible by extending the interface
// See configEvent for more details
type Event interface {
	Subject() string
	Time() time.Time
	EventType() EventType
	Object() interface{}
	fmt.Stringer
}

type eventImpl struct {
	subject   string
	time      time.Time
	eventType EventType
	object    interface{}
}

func (e eventImpl) Subject() string {
	return e.subject
}

func (e eventImpl) Time() time.Time {
	return e.time
}

func (e eventImpl) EventType() EventType {
	return e.eventType
}

// Object returns the object associated with the event
func (e eventImpl) Object() interface{} {
	return e.object
}

func (e eventImpl) String() string {
	return fmt.Sprintf("%s %s %s {%s}",
		e.subject, e.eventType, e.time.Format(time.RFC3339), e.object)
}

// NewEvent creates a new event object
func NewEvent(subject string, eventType EventType, obj interface{}) Event {
	return eventImpl{
		subject:   subject,
		time:      time.Now(),
		eventType: eventType,
		object:    obj,
	}
}
