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

// ErrorEvent a error event
type ErrorEvent Event

// ChangeID returns the changeId of the error event
func (errEvent *ErrorEvent) ChangeID() string {
	return errEvent.values[ChangeID]
}

// EventType returns the EventType of the error event
func (errEvent *ErrorEvent) EventType() EventType {
	return errEvent.eventtype
}

// Error returns the error of the error event
func (errEvent *ErrorEvent) Error() error {
	return fmt.Errorf(errEvent.values[Error])
}

// CreateErrorEvent creates a new error event object
func CreateErrorEvent(eventType EventType, subject string, changeID change.ID, err error) ErrorEvent {
	values := make(map[string]string)
	values[ChangeID] = base64.StdEncoding.EncodeToString(changeID)
	values[Error] = string(err.Error())
	return ErrorEvent{
		subject:   subject,
		time:      time.Now(),
		eventtype: eventType,
		values:    values,
	}
}
