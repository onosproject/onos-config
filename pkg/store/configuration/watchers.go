// Copyright 2021-present Open Networking Foundation.
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

package configuration

import (
	"sync"

	"github.com/google/uuid"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
)

type watcher struct {
	eventCh chan<- configapi.ConfigurationEvent
	replay  bool
	watchID configapi.ConfigurationID
}

// watchers stores the information about watchers
type watchers struct {
	watchers map[uuid.UUID]watcher
	rm       sync.RWMutex
}

// newWatchers creates watchers
func newWatchers() *watchers {
	return &watchers{
		watchers: make(map[uuid.UUID]watcher),
	}
}

// sendAll sends a configuration event for all registered watchers
func (ws *watchers) sendAll(event configapi.ConfigurationEvent) {
	ws.rm.RLock()
	for _, watcher := range ws.watchers {
		if watcher.watchID != "" && watcher.watchID == event.Configuration.ID {
			watcher.eventCh <- event
		} else if watcher.watchID == "" {
			watcher.eventCh <- event
		}
	}
	ws.rm.RUnlock()
}

// add adds a watcher
func (ws *watchers) add(id uuid.UUID, watcher watcher) {
	ws.rm.Lock()
	ws.watchers[id] = watcher
	ws.rm.Unlock()

}

// remove removes a watcher
func (ws *watchers) remove(id uuid.UUID) {
	ws.rm.Lock()
	delete(ws.watchers, id)
	ws.rm.Unlock()

}
