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

const (
	// ChangeID :
	ChangeID = "ChangeID"

	// Applied :
	Applied = "Applied"

	// Connect :
	Connect = "Connect"

	// Address :
	Address = "Address"

	//Error :
	Error = "Error"

	//Response :
	Response = "Response"
)

// EventType is an enumerated type
type EventType int

// Values of the EventType enumeration
const ( // For event types
	EventTypeConfiguration EventType = iota
	EventTypeTopoCache
	EventTypeOperationalState
	EventTypeAchievedSetConfig
	EventTypeErrorSetConfig
	EventTypeErrorParseConfig
	EventTypeErrorSetInitialConfig
	EventTypeErrorDeviceConnect
	EventTypeErrorDeviceCapabilities
	EventTypeErrorDeviceDisconnect
	EventTypeErrorSubscribe
	EventTypeErrorMissingModelPlugin
	EventTypeErrorTranslation
	EventTypeErrorGetWithRoPaths
)

func (et EventType) String() string {
	return [...]string{"Configuration", "TopoCache", "OperationalState", "EventTypeAchievedSetConfig",
		"EventTypeErrorSetConfig", "EventTypeErrorParseConfig", "EventTypeErrorSetInitialConfig",
		"EventTypeErrorDeviceConnect", "EventTypeErrorDeviceCapabilities", "EventTypeErrorDeviceDisconnect",
		"EventTypeErrorSubscribe, EventTypeErrorMissingModelPlugin, EventTypeErrorTranslation"}[et]
}

// Event is a general purpose base type of event
type Event struct {
	subject   string
	time      time.Time
	eventtype EventType
	values    map[string]string
	object    interface{}
}

func (e Event) String() string {
	var evtValues string
	for k, v := range e.values {
		evtValues = evtValues + k + ":" + v + ","
	}
	return fmt.Sprintf("%s %s %s {%s}",
		e.subject, e.eventtype, e.time.Format(time.RFC3339), evtValues)
}

// Subject extracts the subject field from the event
func (e Event) Subject() string {
	return e.subject
}

// Time extracts the time field from the event
func (e Event) Time() time.Time {
	return e.time
}

// EventType extracts the eventtype field from the event
func (e Event) EventType() EventType {
	return e.eventtype
}

// Values extracts a map of values from the event
func (e Event) Values() *map[string]string {
	return &e.values
}

// Value extracts a single value from the event
func (e Event) Value(name string) string {
	return e.values[name]
}

// Object returns the object associated with the event
func (e Event) Object() interface{} {
	return e.object
}

// createEvent creates a new event object
func createEvent(subject string, eventType EventType, values map[string]string) Event {
	return Event{
		subject:   subject,
		time:      time.Now(),
		eventtype: eventType,
		values:    values,
	}
}
