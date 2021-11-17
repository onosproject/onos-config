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

package v1beta1

import (
	"context"
	"sync"

	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// GNMIConnManager gNMI connection manager
type GNMIConnManager interface {
	Get(ctx context.Context, id ConnID) (*GNMIConn, error)
	List(ctx context.Context) ([]*GNMIConn, error)
	Watch(ctx context.Context, ch chan<- *GNMIConn) error
	Open(conn *GNMIConn)
}

// NewGNMIConnManager creates a new gNMI connection manager
func NewGNMIConnManager() GNMIConnManager {
	mgr := &gnmiConnManager{
		conns:   make(map[ConnID]*GNMIConn),
		eventCh: make(chan *GNMIConn),
	}
	go mgr.processEvents()
	return mgr
}

type gnmiConnManager struct {
	conns      map[ConnID]*GNMIConn
	connsMu    sync.RWMutex
	watchers   []chan<- *GNMIConn
	watchersMu sync.RWMutex
	eventCh    chan *GNMIConn
}

func (m *gnmiConnManager) processEvents() {
	for conn := range m.eventCh {
		m.processEvent(conn)
	}
}

func (m *gnmiConnManager) processEvent(conn *GNMIConn) {
	log.Infof("Notifying gNMI connection: %s", conn.ID)
	m.watchersMu.RLock()
	for _, watcher := range m.watchers {
		watcher <- conn
	}
	m.watchersMu.RUnlock()
}

// Get returns a gNMI connection based on a given connection ID
func (m *gnmiConnManager) Get(ctx context.Context, connID ConnID) (*GNMIConn, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conn, ok := m.conns[connID]
	if !ok {
		return nil, errors.NewNotFound("gNMI connection '%s' not found", connID)
	}
	return conn, nil
}

// List lists all  gNMI connections
func (m *gnmiConnManager) List(ctx context.Context) ([]*GNMIConn, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conns := make([]*GNMIConn, 0, len(m.conns))
	for _, conn := range m.conns {
		conns = append(conns, conn)
	}
	return conns, nil
}

// Watch watches gNMI connection changes
func (m *gnmiConnManager) Watch(ctx context.Context, ch chan<- *GNMIConn) error {
	m.watchersMu.Lock()
	m.connsMu.Lock()
	m.watchers = append(m.watchers, ch)
	m.watchersMu.Unlock()

	go func() {
		for _, stream := range m.conns {
			ch <- stream
		}
		m.connsMu.Unlock()

		<-ctx.Done()
		m.watchersMu.Lock()
		watchers := make([]chan<- *GNMIConn, 0, len(m.watchers)-1)
		for _, watcher := range watchers {
			if watcher != ch {
				watchers = append(watchers, watcher)
			}
		}
		m.watchers = watchers
		m.watchersMu.Unlock()
	}()
	return nil
}

func (m *gnmiConnManager) Open(conn *GNMIConn) {
	log.Infof("Opened gNMI connection %s", conn.ID)
	m.connsMu.Lock()
	m.conns[conn.ID] = conn
	m.connsMu.Unlock()
	m.eventCh <- conn
	go func() {
		<-conn.Context().Done()
		log.Infof("Closing gNMI connection %s", conn.ID)
		m.connsMu.Lock()
		delete(m.conns, conn.ID)
		m.connsMu.Unlock()
		m.eventCh <- conn
	}()

}

var _ GNMIConnManager = &gnmiConnManager{}
