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
	"fmt"
	devicechangetypes "github.com/onosproject/onos-config/api/types/change/device"
	"strings"
	"testing"
	"time"

	"gotest.tools/assert"
)

const (
	eventSubject = "device22"
	path1        = "test1/cont1a/cont2a/leaf2a"
	value1       = "value1"
	testResponse = "test response"
)

func Test_eventConstruction(t *testing.T) {
	values := make(map[string]string)
	event := NewEvent(eventSubject, EventTypeOperationalState, values)

	assert.Equal(t, event.EventType(), EventTypeOperationalState)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Assert(t, strings.Contains(event.String(), eventSubject))
	assert.Equal(t, len(event.Object().(map[string]string)), 0)
}

func Test_operationalStateEventConstruction(t *testing.T) {

	event := NewOperationalStateEvent(eventSubject, path1, devicechangetypes.NewTypedValueString(value1), EventItemAdded)

	assert.Equal(t, event.EventType(), EventTypeOperationalState)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Assert(t, strings.Contains(event.String(), eventSubject))
	assert.Assert(t, strings.Contains(EventTypeOperationalState.String(), "OperationalState"))
	assert.Equal(t, event.ItemAction(), EventItemAdded)

	assert.Equal(t, event.Path(), path1)
	assert.Equal(t, event.Value().ValueToString(), value1)
}

func Test_errorEventBoChangeIDConstruction(t *testing.T) {
	event := NewErrorEventNoChangeID(EventTypeErrorDeviceConnect, eventSubject, fmt.Errorf(testResponse))

	assert.Equal(t, event.EventType(), EventTypeErrorDeviceConnect)
	assert.Equal(t, event.Subject(), eventSubject)
	assert.Assert(t, event.Time().Before(time.Now()))

	assert.Equal(t, event.ChangeID(), "")
	assert.Equal(t, event.Response(), "")
	assert.Error(t, event.Error(), testResponse, "expected an error")
}
