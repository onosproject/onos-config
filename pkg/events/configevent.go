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
	"github.com/onosproject/onos-config/pkg/store/change"
	"time"
)

// ConfigEvent a configuration event. It is a specialization of Event, by adding
// 2 new methods to interface
type ConfigEvent interface {
	Event
	ChangeID() string
	Applied() bool // TODO: See if this can be removed
}

// An object type to contain extra attributes
type configEventObj struct {
	changeID string
	applied  bool
}

// extend the original event struct.
// Any new attributes will be stored in the object, so don't store them here
type configEventImpl struct {
	eventImpl
}

// ChangeID returns the changeID of the event
func (e configEventImpl) ChangeID() string {
	ce, ok := e.object.(configEventObj)
	if ok {
		return ce.changeID
	}
	return ""
}

// Applied returns if the event is for an application or a rollback
func (e configEventImpl) Applied() bool {
	ce, ok := e.object.(configEventObj)
	if ok {
		return ce.applied
	}
	return false
}

// NewConfigEvent creates a new config event object
func NewConfigEvent(subject string, changeID change.ID, applied bool) ConfigEvent {
	ce := configEventImpl{eventImpl{
		subject:   subject,
		time:      time.Now(),
		eventType: EventTypeConfiguration,
		object: configEventObj{
			changeID: base64.StdEncoding.EncodeToString(changeID),
			applied:  applied,
		},
	}}
	return ce
}
