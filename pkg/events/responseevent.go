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
	"time"
)

// DeviceResponse a response event
type DeviceResponse Event

// ChangeID returns the changeId of the response event
func (respEvent *DeviceResponse) ChangeID() string {
	return respEvent.values[ChangeID]
}

// EventType returns the EventType of the response event
func (respEvent *DeviceResponse) EventType() EventType {
	return respEvent.eventtype
}

// Error returns the error of the response event
func (respEvent *DeviceResponse) Error() error {
	return fmt.Errorf(respEvent.values[Error])
}

// Response returns the Response of the response event
func (respEvent *DeviceResponse) Response() error {
	return fmt.Errorf(respEvent.values[Response])
}

// CreateResponseEvent creates a new response event object
func CreateResponseEvent(eventType EventType, subject string, changeID change.ID, response string) DeviceResponse {
	values := make(map[string]string)
	values[ChangeID] = base64.StdEncoding.EncodeToString(changeID)
	values[Response] = response
	return DeviceResponse{
		subject:   subject,
		time:      time.Now(),
		eventtype: eventType,
		values:    values,
	}
}

// CreateErrorEvent creates a new error event object
func CreateErrorEvent(eventType EventType, subject string, changeID change.ID, err error) DeviceResponse {
	values := make(map[string]string)
	values[ChangeID] = base64.StdEncoding.EncodeToString(changeID)
	values[Error] = string(err.Error())
	return DeviceResponse{
		subject:   subject,
		time:      time.Now(),
		eventtype: eventType,
		values:    values,
	}
}

// CreateErrorEventNoChangeID creates a new error event object with no changeID attached
func CreateErrorEventNoChangeID(eventType EventType, subject string, err error) DeviceResponse {
	values := make(map[string]string)
	values[Error] = string(err.Error())
	return DeviceResponse{
		subject:   subject,
		time:      time.Now(),
		eventtype: eventType,
		values:    values,
	}
}
