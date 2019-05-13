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
	"gotest.tools/assert"
	"strings"
	"testing"
	"time"
)

const (
	eventSubject = "device22"
	eventType = EventTypeConfiguration
	eventValueKey = ChangeID
	eventValue = "test-event"
)

func Test_eventConstruction(t *testing.T) {
	values := make(map[string]string)
	values[eventValueKey] = eventValue
	event := CreateEvent(eventSubject, eventType, values)

	assert.Equal(t, event.EventType(), eventType)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Equal(t, (*event.Values())[eventValueKey], eventValue)
	assert.Equal(t, event.Value(eventValueKey), eventValue)

	assert.Assert(t, strings.Contains(event.String(), eventSubject))
	assert.Assert(t, strings.Contains(EventTypeConfiguration.String(), "Configuration"))
}