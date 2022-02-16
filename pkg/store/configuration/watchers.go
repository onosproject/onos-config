// SPDX-FileCopyrightText: ${year}-present Open Networking Foundation <info@opennetworking.org>
// SPDX-License-Identifier: LicenseRef-ONF-Member-Only-1.0

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
	go func() {
		for _, watcher := range ws.watchers {
			if watcher.watchID != "" && watcher.watchID == event.Configuration.ID {
				watcher.eventCh <- event
			} else if watcher.watchID == "" {
				watcher.eventCh <- event
			}
		}
	}()
	ws.rm.RUnlock()
}

// add adds a watcher
func (ws *watchers) add(id uuid.UUID, watcher watcher) error {
	ws.rm.Lock()
	ws.watchers[id] = watcher
	ws.rm.Unlock()
	return nil

}

// remove removes a watcher
func (ws *watchers) remove(id uuid.UUID) error {
	ws.rm.Lock()
	watchers := make(map[uuid.UUID]watcher, len(ws.watchers)-1)
	for watcherID, watcher := range ws.watchers {
		if watcherID != id {
			watchers[id] = watcher
		}
	}
	ws.watchers = watchers
	ws.rm.Unlock()
	return nil

}
