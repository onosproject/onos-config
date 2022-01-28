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
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"sync"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
)

// ConnManager gNMI connection manager
type ConnManager interface {
	Get(ctx context.Context, connID ConnID) (Conn, bool)
	Connect(ctx context.Context, target *topoapi.Object) error
	Disconnect(ctx context.Context, targetID topoapi.ID) error
	Watch(ctx context.Context, ch chan<- Conn) error
}

// NewConnManager creates a new gNMI connection manager
func NewConnManager() ConnManager {
	mgr := &connManager{
		targets: make(map[topoapi.ID]Target),
		conns:   make(map[ConnID]Conn),
		eventCh: make(chan Conn),
	}
	go mgr.processEvents()
	return mgr
}

type connManager struct {
	targets    map[topoapi.ID]Target
	conns      map[ConnID]Conn
	stateMu    sync.RWMutex
	watchers   []chan<- Conn
	watchersMu sync.RWMutex
	eventCh    chan Conn
}

func (m *connManager) Get(ctx context.Context, connID ConnID) (Conn, bool) {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	conn, ok := m.conns[connID]
	return conn, ok
}

func (m *connManager) Connect(ctx context.Context, target *topoapi.Object) error {
	m.stateMu.RLock()
	connTarget, ok := m.targets[target.ID]
	m.stateMu.RUnlock()
	if ok {
		return errors.NewAlreadyExists("target '%s' already exists", target.ID)
	}

	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	connTarget, ok = m.targets[target.ID]
	if ok {
		return errors.NewAlreadyExists("target '%s' already exists", target.ID)
	}

	connTarget = newTarget(m, target)
	if err := connTarget.Connect(ctx); err != nil {
		return err
	}
	m.targets[target.ID] = connTarget
	return nil
}

func (m *connManager) Disconnect(ctx context.Context, targetID topoapi.ID) error {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	connTarget, ok := m.targets[targetID]
	if !ok {
		return errors.NewNotFound("target '%s' not found", targetID)
	}
	return connTarget.Close(ctx)
}

func (m *connManager) Watch(ctx context.Context, ch chan<- Conn) error {
	m.watchersMu.Lock()
	m.stateMu.Lock()
	m.watchers = append(m.watchers, ch)
	m.watchersMu.Unlock()

	go func() {
		for _, conn := range m.conns {
			ch <- conn
		}
		m.stateMu.Unlock()

		<-ctx.Done()
		m.watchersMu.Lock()
		watchers := make([]chan<- Conn, 0, len(m.watchers)-1)
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

func (m *connManager) addConn(conn Conn) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	m.conns[conn.ID()] = conn
	m.eventCh <- conn
}

func (m *connManager) removeConn(connID ConnID) {
	m.stateMu.Lock()
	if conn, ok := m.conns[connID]; ok {
		delete(m.conns, connID)
		m.stateMu.Unlock()
		m.eventCh <- conn
	} else {
		m.stateMu.Unlock()
	}
}

func (m *connManager) processEvents() {
	for conn := range m.eventCh {
		m.processEvent(conn)
	}
}

func (m *connManager) processEvent(conn Conn) {
	log.Infof("Notifying gNMI connection: %s", conn.ID())
	m.watchersMu.RLock()
	for _, watcher := range m.watchers {
		watcher <- conn
	}
	m.watchersMu.RUnlock()
}

var _ ConnManager = &connManager{}
