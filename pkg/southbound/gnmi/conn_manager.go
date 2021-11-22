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

package gnmi

import (
	"context"
	"sync"

	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// ConnManager gNMI connection manager
type ConnManager interface {
	Get(ctx context.Context, id ConnID) (*Conn, error)
	List(ctx context.Context) ([]*Conn, error)
	Watch(ctx context.Context, ch chan<- *Conn) error
	Add(conn *Conn) error
	Remove(connID ConnID) error
}

// NewConnManager creates a new gNMI connection manager
func NewConnManager() ConnManager {
	mgr := &connManager{
		conns:   make(map[ConnID]*Conn),
		eventCh: make(chan *Conn),
	}
	go mgr.processEvents()
	return mgr
}

type connManager struct {
	conns      map[ConnID]*Conn
	connsMu    sync.RWMutex
	watchers   []chan<- *Conn
	watchersMu sync.RWMutex
	eventCh    chan *Conn
}

// Add adding a new gNMI connection
func (m *connManager) Add(conn *Conn) error {
	log.Infof("Adding gNMI connection %s", conn.ID)
	m.connsMu.Lock()
	_, ok := m.conns[conn.ID]
	if ok {
		m.connsMu.Unlock()
		return errors.NewAlreadyExists("gNMI connection %s already exists", conn.ID)
	}
	m.conns[conn.ID] = conn
	m.connsMu.Unlock()
	m.eventCh <- conn
	go func() {
		<-conn.Context().Done()
		log.Infof("Context is cancelled, removing gNMI connection %s", conn.ID)
		m.connsMu.Lock()
		delete(m.conns, conn.ID)
		m.connsMu.Unlock()
		m.eventCh <- conn
	}()
	return nil
}

// Remove removing a gNMI connection
func (m *connManager) Remove(connID ConnID) error {
	log.Infof("Removing gNMI connection %s", connID)
	m.connsMu.Lock()
	defer m.connsMu.Unlock()
	_, ok := m.conns[connID]
	if !ok {
		return errors.NewNotFound("gNMI connection %s not found", connID)
	}
	delete(m.conns, connID)
	return nil
}

// Get returns a gNMI connection based on a given connection ID
func (m *connManager) Get(ctx context.Context, connID ConnID) (*Conn, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conn, ok := m.conns[connID]
	if !ok {
		return nil, errors.NewNotFound("gNMI connection '%s' not found", connID)
	}
	return conn, nil
}

// List lists all  gNMI connections
func (m *connManager) List(ctx context.Context) ([]*Conn, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conns := make([]*Conn, 0, len(m.conns))
	for _, conn := range m.conns {
		conns = append(conns, conn)
	}
	return conns, nil
}

// Watch watches gNMI connection changes
func (m *connManager) Watch(ctx context.Context, ch chan<- *Conn) error {
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
		watchers := make([]chan<- *Conn, 0, len(m.watchers)-1)
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

func (m *connManager) processEvents() {
	for conn := range m.eventCh {
		m.processEvent(conn)
	}
}

func (m *connManager) processEvent(conn *Conn) {
	log.Infof("Notifying gNMI connection: %s", conn.ID)
	m.watchersMu.RLock()
	for _, watcher := range m.watchers {
		watcher <- conn
	}
	m.watchersMu.RUnlock()
}

var _ ConnManager = &connManager{}
