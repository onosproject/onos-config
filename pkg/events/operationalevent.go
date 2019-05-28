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

// OperationalStateEvent represents an event for an update in operational state on a device
type OperationalStateEvent Event

// CreateOperationalStateEvent creates a new operational state event object
func CreateOperationalStateEvent(subject string, pathsAndValues map[string]string) OperationalStateEvent {
	return OperationalStateEvent{
		subject:   subject,
		time:      time.Now(),
		eventtype: EventTypeOperationalState,
		values:    pathsAndValues,
	}
}
