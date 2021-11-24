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

	"google.golang.org/grpc"

	"google.golang.org/grpc/connectivity"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// ConnManager gNMI connection manager
type ConnManager interface {
	Get(ctx context.Context, id ConnID) (Conn, error)
	List(ctx context.Context) ([]Conn, error)
	Watch(ctx context.Context, ch chan<- Conn) error
	Connect(ctx context.Context, target *topoapi.Object) (Conn, error)
	remove(connID ConnID) error
}

// NewConnManager creates a new gNMI connection manager
func NewConnManager() ConnManager {
	mgr := &connManager{
		conns:       make(map[ConnID]Conn),
		clientConns: make(map[topoapi.ID]*grpc.ClientConn),
		eventCh:     make(chan Conn),
	}
	go mgr.processEvents()
	return mgr
}

type connManager struct {
	conns       map[ConnID]Conn
	clientConns map[topoapi.ID]*grpc.ClientConn
	connsMu     sync.RWMutex
	watchers    []chan<- Conn
	watchersMu  sync.RWMutex
	eventCh     chan Conn
}

// Connect connecting to a gNMI target and adding a new gNMI connection
func (m *connManager) Connect(ctx context.Context, target *topoapi.Object) (Conn, error) {
	log.Infof("Connecting to the gNMI target: %s", target.ID)
	conn, err := connect(target)
	if err != nil {
		log.Errorf("Failed to connect to the gNMI target: %s", target.ID)
		return nil, err
	}

	log.Infof("Adding gNMI connection %s for the target %s", conn.ID(), target.ID)
	m.connsMu.Lock()
	_, ok := m.conns[conn.ID()]
	if ok {
		m.connsMu.Unlock()
		return nil, errors.NewAlreadyExists("gNMI connection %s already exists", conn.ID())
	}

	m.conns[conn.ID()] = conn
	m.clientConns[target.ID] = conn.clientConn()
	m.connsMu.Unlock()
	m.eventCh <- conn

	go func() {
		clientConn := conn.clientConn()
		state := clientConn.GetState()
		for state != connectivity.Shutdown && clientConn.WaitForStateChange(ctx, state) {
			state := clientConn.GetState()
			log.Infof("Current state of the gNMI connection %s: is %s", conn.ID(), state.String())
			if state == connectivity.TransientFailure {
				log.Infof("Closing the gNMI connection for target %s:%s", conn.ID(), target.ID)
				m.connsMu.Lock()
				if conn, ok := m.conns[conn.ID()]; ok {
					delete(m.conns, conn.ID())
					m.eventCh <- conn
				}
				m.connsMu.Unlock()
				break
			}
			if state == connectivity.Ready {
				m.connsMu.Lock()
				if _, ok := m.conns[conn.ID()]; !ok {
					conn := newConn(target)
					log.Infof("Connection %s for target %s is established", conn.ID(), target.ID)
					m.conns[conn.ID()] = conn
					m.eventCh <- conn
				}
				m.connsMu.Unlock()
			}
		}

	}()

	return conn, nil
}

// Remove removing a gNMI connection
func (m *connManager) remove(connID ConnID) error {
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
func (m *connManager) Get(ctx context.Context, connID ConnID) (Conn, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conn, ok := m.conns[connID]
	if !ok {
		return nil, errors.NewNotFound("gNMI connection '%s' not found", connID)
	}
	return conn, nil
}

// List lists all  gNMI connections
func (m *connManager) List(ctx context.Context) ([]Conn, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conns := make([]Conn, 0, len(m.conns))
	for _, conn := range m.conns {
		conns = append(conns, conn)
	}
	return conns, nil
}

// Watch watches gNMI connection changes
func (m *connManager) Watch(ctx context.Context, ch chan<- Conn) error {
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
