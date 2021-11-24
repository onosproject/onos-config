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
	"math"
	"sync"

	gclient "github.com/openconfig/gnmi/client/gnmi"

	"github.com/google/uuid"

	"github.com/onosproject/onos-lib-go/pkg/uri"

	"google.golang.org/grpc"

	"google.golang.org/grpc/connectivity"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// ConnManager gNMI connection manager
type ConnManager interface {
	Get(ctx context.Context, targetID topoapi.ID) (Conn, error)
	List(ctx context.Context) ([]Conn, error)
	Watch(ctx context.Context, ch chan<- Conn) error
	Connect(ctx context.Context, target *topoapi.Object) (Conn, error)
}

// NewConnManager creates a new gNMI connection manager
func NewConnManager() ConnManager {
	mgr := &connManager{
		conns:       make(map[topoapi.ID]Conn),
		clientConns: make(map[topoapi.ID]*grpc.ClientConn),
		eventCh:     make(chan Conn),
	}
	go mgr.processEvents()
	return mgr
}

type connManager struct {
	conns       map[topoapi.ID]Conn
	clientConns map[topoapi.ID]*grpc.ClientConn
	connsMu     sync.RWMutex
	watchers    []chan<- Conn
	watchersMu  sync.RWMutex
	eventCh     chan Conn
}

func newConnID() ConnID {
	connID := ConnID(uri.NewURI(
		uri.WithScheme("uuid"),
		uri.WithOpaque(uuid.New().String())).String())
	return connID
}

// newConn creates a new gNMI connection
func (m *connManager) connect(ctx context.Context, target *topoapi.Object) (Conn, error) {
	m.connsMu.Lock()
	currentConn, ok := m.conns[target.ID]
	if ok {
		m.connsMu.Unlock()
		return nil, errors.NewAlreadyExists("gNMI connection %s already exists for target %s", currentConn.ID(), target.ID)
	}
	m.connsMu.Unlock()
	if target.Type != topoapi.Object_ENTITY {
		return nil, errors.NewInvalid("object is not a topo entity %v+", target)
	}

	typeKindID := string(target.GetEntity().KindID)
	if len(typeKindID) == 0 {
		return nil, errors.NewInvalid("target entity %s must have a 'kindID' to work with onos-config", target.ID)
	}

	destination, err := newDestination(target)
	if err != nil {
		log.Warnf("Failed to create a new target %s", err)
		return nil, err
	}
	log.Infof("Connecting to gNMI target: %+v", destination)
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
	}
	gnmiClient, clientConn, err := newGNMIClient(ctx, *destination, opts)
	if err != nil {
		log.Warnf("Failed to connect to the gNMI target %s: %s", destination.Target, err)
		return nil, err
	}

	connID := newConnID()
	log.Infof("Adding gNMI connection %s for the target %s", connID, target.ID)
	conn := &conn{
		id:     connID,
		client: gnmiClient,
	}
	m.connsMu.Lock()
	m.conns[target.ID] = conn
	m.clientConns[target.ID] = clientConn
	m.connsMu.Unlock()
	m.eventCh <- conn

	return conn, nil
}

// Connect connecting to a gNMI target and adding a new gNMI connection
func (m *connManager) Connect(ctx context.Context, target *topoapi.Object) (Conn, error) {
	newConn, err := m.connect(ctx, target)
	if err != nil {
		return nil, err
	}

	go func() {
		clientConn := m.clientConns[target.ID]
		state := clientConn.GetState()
		log.Infof("Initial state of the gNMI connection %s: is %s", newConn.ID(), state.String())
		for state != connectivity.Shutdown && clientConn.WaitForStateChange(ctx, state) {
			state = clientConn.GetState()
			log.Infof("Current state of the gNMI connection %s: is %s", newConn.ID(), state.String())
			if state == connectivity.TransientFailure {
				m.connsMu.Lock()
				if newConn, ok := m.conns[target.ID]; ok {
					log.Infof("Closing the gNMI connection for target %s:%s", newConn.ID(), target.ID)
					delete(m.conns, target.ID)
					m.eventCh <- newConn
				}
				m.connsMu.Unlock()
			}
			if state == connectivity.Ready {
				m.connsMu.Lock()
				if _, ok := m.conns[target.ID]; !ok {
					log.Infof("Connection %s for target %s is established", newConn.ID(), target.ID)
					d, err := newDestination(target)
					if err != nil {
						log.Warnf("Failed to create new destination for target %v: %s", target, err)
						continue
					}
					cl, err := gclient.NewFromConn(ctx, clientConn, *d)
					if err != nil {
						log.Warnf("Failed to create new connection instance: %s", err)
						continue
					}
					newConn = &conn{
						id: newConnID(),
						client: &client{
							client: cl,
						},
					}
					m.conns[target.ID] = newConn
					m.eventCh <- newConn
				}
				m.connsMu.Unlock()
			}
		}

	}()

	return newConn, nil
}

// Get returns a gNMI connection based on a given target ID
func (m *connManager) Get(ctx context.Context, targetID topoapi.ID) (Conn, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conn, ok := m.conns[targetID]
	if !ok {
		return nil, errors.NewNotFound("gNMI connection for target '%s' not found", targetID)
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
