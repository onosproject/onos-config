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
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
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
	testChangeId  = "dGVzdDE="
	testResponse  = "test response"
)

func Test_eventConstruction(t *testing.T) {
	values := make(map[string]string)
	values[eventValueKey] = eventValue
	event := createEvent(eventSubject, eventTypeCfg, values)

	assert.Equal(t, event.EventType(), eventTypeCfg)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Assert(t, strings.Contains(event.String(), eventSubject))
	assert.Equal(t, len(event.Object().(map[string]string)), 1)
}

func Test_configEventConstruction(t *testing.T) {

	b := []byte(eventValue)

	event := CreateConfigEvent(eventSubject, b, true)

	assert.Equal(t, event.EventType(), eventTypeCfg)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Equal(t, event.ChangeID(), base64.StdEncoding.EncodeToString(b))
	assert.Equal(t, event.Applied(), true)

	assert.Assert(t, strings.Contains(event.String(), eventSubject))
	assert.Assert(t, strings.Contains(EventTypeConfiguration.String(), "Configuration"))
}

func Test_topoEventConstruction(t *testing.T) {

	event := CreateTopoEvent(eventSubject, EventItemAdded, &device.Device{ID: device.ID("foo"), Address: eventAddress})

	assert.Equal(t, event.EventType(), eventTypeTopo)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))
	assert.Equal(t, event.Device().Address, eventAddress)
	assert.Equal(t, device.ID("foo"), event.Device().ID)

	assert.Equal(t, event.ItemAction(), EventItemAdded)

	assert.Assert(t, strings.Contains(event.String(), eventSubject))
	assert.Assert(t, strings.Contains(EventTypeTopoCache.String(), "Topo"))
}

func Test_operationalStateEventConstruction(t *testing.T) {

	event := CreateOperationalStateEvent(eventSubject, path1, change.CreateTypedValueString(value1), EventItemAdded)

	assert.Equal(t, event.EventType(), EventTypeOperationalState)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Assert(t, strings.Contains(event.String(), eventSubject))
	assert.Assert(t, strings.Contains(EventTypeOperationalState.String(), "OperationalState"))
	assert.Equal(t, event.ItemAction(), EventItemAdded)

	assert.Equal(t, event.Path(), path1)
	assert.Equal(t, event.Value().String(), value1)
}

func Test_responseEventConstruction(t *testing.T) {
	cid1, _ := base64.StdEncoding.DecodeString(testChangeId)
	event := CreateResponseEvent(EventTypeAchievedSetConfig, eventSubject, cid1, testResponse)

	assert.Equal(t, event.EventType(), EventTypeAchievedSetConfig)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Equal(t, event.ChangeID(), testChangeId)
	assert.Equal(t, event.Response(), testResponse)
	assert.NilError(t, event.Error())
}

func Test_errorEventConstruction(t *testing.T) {
	cid1, _ := base64.StdEncoding.DecodeString(testChangeId)
	event := CreateErrorEvent(EventTypeErrorGetWithRoPaths, eventSubject, cid1, fmt.Errorf(testResponse))

	assert.Equal(t, event.EventType(), EventTypeErrorGetWithRoPaths)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Equal(t, event.ChangeID(), testChangeId)
	assert.Equal(t, event.Response(), "")
	assert.Error(t, event.Error(), testResponse, "expected an error")
}

func Test_errorEventBoChangeIDConstruction(t *testing.T) {
	event := CreateErrorEventNoChangeID(EventTypeErrorDeviceConnect, eventSubject, fmt.Errorf(testResponse))

	assert.Equal(t, event.EventType(), EventTypeErrorDeviceConnect)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Equal(t, event.ChangeID(), "")
	assert.Equal(t, event.Response(), "")
	assert.Error(t, event.Error(), testResponse, "expected an error")
}
