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
	"strconv"
	"time"

	"github.com/onosproject/onos-config/pkg/store/change"
	log "k8s.io/klog"
)

// ConfigEvent a configuration event
type ConfigEvent Event

// ChangeID returns the changeId of the event
func (cfgevent *ConfigEvent) ChangeID() string {
	return cfgevent.values[ChangeID]
}

// Applied returns if the event is for an application or a rollback
func (cfgevent *ConfigEvent) Applied() bool {
	b, err := strconv.ParseBool(cfgevent.values[Applied])
	if err != nil {
		log.Warning("error in conversion ", err)
		return false
	}
	return b
}

// CreateConfigEvent creates a new config event object
func CreateConfigEvent(subject string, changeID change.ID, applied bool) ConfigEvent {
	values := make(map[string]string)
	values[ChangeID] = base64.StdEncoding.EncodeToString(changeID)
	values[Applied] = strconv.FormatBool(applied)
	return ConfigEvent{
		subject:   subject,
		time:      time.Now(),
		eventtype: EventTypeConfiguration,
		values:    values,
	}
}
