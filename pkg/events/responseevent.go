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
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"time"
)

// DeviceResponse a response event
type DeviceResponse interface {
	Event
	ChangeID() string
	Error() error
	Response() string
}

type deviceResponseObj struct {
	changeID string
	error    error
	response string
}

type deviceResponseImpl struct {
	eventImpl
}

// ChangeID returns the changeID of the response event
func (respEvent deviceResponseImpl) ChangeID() string {
	re, ok := respEvent.object.(deviceResponseObj)
	if ok {
		return re.changeID
	}
	return ""
}

// Error returns the error object of the response event
func (respEvent deviceResponseImpl) Error() error {
	err, ok := respEvent.object.(deviceResponseObj)
	if ok {
		return err.error
	}
	return nil
}

// Response returns the response object of the response event
func (respEvent deviceResponseImpl) Response() string {
	err, ok := respEvent.object.(deviceResponseObj)
	if ok {
		return err.response
	}
	return ""
}

// NewResponseEvent creates a new response event object
func NewResponseEvent(eventType EventType, subject string, changeID devicechange.ID, response string) DeviceResponse {
	dr := deviceResponseImpl{
		eventImpl: eventImpl{
			subject:   subject,
			time:      time.Now(),
			eventType: eventType,
			object: deviceResponseObj{
				changeID: string(changeID),
				error:    nil,
				response: response,
			},
		},
	}
	return &dr
}

// NewDeviceConnectedEvent creates a new response event object
func NewDeviceConnectedEvent(eventType EventType, subject string) DeviceResponse {
	dr := deviceResponseImpl{
		eventImpl: eventImpl{
			subject:   subject,
			time:      time.Now(),
			eventType: eventType,
			object: deviceResponseObj{
				changeID: "",
				error:    nil,
				response: "",
			},
		},
	}
	return &dr
}

// NewErrorEventNoChangeID creates a new error event object with no changeID attached
func NewErrorEventNoChangeID(eventType EventType, subject string, err error) DeviceResponse {
	dr := deviceResponseImpl{
		eventImpl: eventImpl{
			subject:   subject,
			time:      time.Now(),
			eventType: eventType,
			object: deviceResponseObj{
				changeID: "",
				error:    err,
				response: "",
			},
		},
	}
	return &dr
}
