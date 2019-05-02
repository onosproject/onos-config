// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

type EventType int

const ( // For event types
	EventTypeConfiguration EventType = iota
	EventTypeTopoCache
)

func (et EventType) String() string {
	return [...]string{"Configuration", "TopoCache"}[et]
}

// Event is a general purpose base type of event
type Event struct {
	subject    string
	time       time.Time
	eventtype  EventType
	values     map[string]string
}

func (e Event) Subject() string {
	return e.subject
}

func (e Event) Time() time.Time {
	return e.time
}

func (e Event) EventType() EventType {
	return e.eventtype
}

func (e Event) Values() *map[string]string {
	return &e.values
}

func (e Event) Value(name string) string {
	return e.values[name]
}

func CreateEvent(subject string, eventtype EventType, values map[string]string) Event {
	return Event{
		subject:subject,
		time:time.Now(),
		eventtype: eventtype,
		values: values,
	}
}
