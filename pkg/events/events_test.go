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
	"encoding/base64"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
)

const (
	eventSubject  = "device22"
	eventAddress  = "device22:10161"
	eventTypeCfg  = EventTypeConfiguration
	eventTypeTopo = EventTypeTopoCache
	eventValueKey = ChangeID
	eventValue    = "test-event"
	path1         = "test1/cont1a/cont2a/leaf2a"
	value1        = "value1"
	path2         = "test1/cont1a/cont2a/leaf2a"
	value2        = "value1"
)

func Test_eventConstruction(t *testing.T) {
	values := make(map[string]string)
	values[eventValueKey] = eventValue
	event := createEvent(eventSubject, eventTypeCfg, values)

	assert.Equal(t, event.EventType(), eventTypeCfg)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Equal(t, (*event.Values())[eventValueKey], eventValue)
	assert.Equal(t, event.Value(eventValueKey), eventValue)

	assert.Assert(t, strings.Contains(event.String(), eventSubject))
}

func Test_configEventConstruction(t *testing.T) {

	b := []byte(eventValue)

	event := CreateConfigEvent(eventSubject, b, true)

	assert.Equal(t, Event(event).EventType(), eventTypeCfg)
	assert.Equal(t, Event(event).Subject(), eventSubject)
	assert.Assert(t, Event(event).Time().Before(time.Now()))

	assert.Equal(t, event.ChangeID(), base64.StdEncoding.EncodeToString(b))
	assert.Equal(t, event.Applied(), true)

	assert.Assert(t, strings.Contains(Event(event).String(), eventSubject))
	assert.Assert(t, strings.Contains(EventTypeConfiguration.String(), "Configuration"))
}

func Test_topoEventConstruction(t *testing.T) {

	event := CreateTopoEvent(eventSubject, true, eventAddress, device.Device{ID: device.ID("foo")})

	assert.Equal(t, Event(event).EventType(), eventTypeTopo)
	assert.Equal(t, Event(event).Subject(), eventSubject)
	assert.Assert(t, Event(event).Time().Before(time.Now()))
	assert.Equal(t, event.Address(), eventAddress)
	assert.Equal(t, device.ID("foo"), Event(event).Object().(device.Device).ID)

	assert.Equal(t, event.Connect(), true)

	assert.Assert(t, strings.Contains(Event(event).String(), eventSubject))
	assert.Assert(t, strings.Contains(EventTypeTopoCache.String(), "Topo"))
}

func Test_operationalStateEventConstruction(t *testing.T) {

	values := make(map[string]string)
	values[path1] = value1
	values[path2] = value2

	event := CreateOperationalStateEvent(eventSubject, values)

	assert.Equal(t, Event(event).EventType(), EventTypeOperationalState)
	assert.Equal(t, Event(event).Subject(), eventSubject)
	assert.Assert(t, Event(event).Time().Before(time.Now()))

	assert.Equal(t, (*Event(event).Values())[path1], value1)
	assert.Equal(t, (*Event(event).Values())[path2], value2)

	assert.Assert(t, strings.Contains(Event(event).String(), eventSubject))
	assert.Assert(t, strings.Contains(EventTypeOperationalState.String(), "OperationalState"))
}
