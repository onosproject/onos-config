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
	"time"
)

// EventType is an enumeration of the kind of events that can occur.
type EventType uint16

// Values of the EventType enumeration
const ( // For event types
	EventTypeConfiguration EventType = iota
	EventTypeOperationalState
)

func (et EventType) String() string {
	return [...]string{"Configuration", "OperationalState"}[et]
}

// Event an interface which defines the Event methods
type Event interface {
	GetType() EventType
	GetTime() time.Time
	GetValues() interface{}
	GetSubject() string
	Clone() Event
}

// EventHappend is a general purpose base type of event
type EventHappend struct {
	Subject string
	Time    time.Time
	Etype   EventType
	Values  interface{}
}

// Clone clones the Event
func (eh *EventHappend) Clone() Event {
	clone := &EventHappend{}
	clone.Etype = eh.Etype
	clone.Subject = eh.Subject
	clone.Time = eh.Time
	clone.Values = eh.Values
	return clone
}

// GetType returns type of an Event
func (eh *EventHappend) GetType() EventType {
	return eh.Etype
}

// GetTime returns the time when the event occurs
func (eh *EventHappend) GetTime() time.Time {
	return eh.Time
}

// GetValues returns the values of the event
func (eh *EventHappend) GetValues() interface{} {
	return eh.Values
}

// GetSubject returns the subject of the event
func (eh *EventHappend) GetSubject() string {
	return eh.Subject
}
