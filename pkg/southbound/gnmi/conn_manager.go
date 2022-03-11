// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"math"
	"sync"

	"github.com/google/uuid"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	baseClient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

// ConnManager gNMI connection manager
type ConnManager interface {
	Get(ctx context.Context, connID ConnID) (Conn, bool)
	GetByTarget(ctx context.Context, targetID topoapi.ID) (Client, error)
	Connect(ctx context.Context, target *topoapi.Object) error
	Disconnect(ctx context.Context, targetID topoapi.ID) error
	Watch(ctx context.Context, ch chan<- Conn) error
}

// NewConnManager creates a new gNMI connection manager
func NewConnManager() ConnManager {
	mgr := &connManager{
		targets:  make(map[topoapi.ID]*client),
		conns:    make(map[ConnID]Conn),
		watchers: make(map[uuid.UUID]chan<- Conn),
		eventCh:  make(chan Conn),
	}
	go mgr.processEvents()
	return mgr
}

type connManager struct {
	targets    map[topoapi.ID]*client
	conns      map[ConnID]Conn
	connsMu    sync.RWMutex
	watchers   map[uuid.UUID]chan<- Conn
	watchersMu sync.RWMutex
	eventCh    chan Conn
}

func (m *connManager) GetByTarget(ctx context.Context, targetID topoapi.ID) (Client, error) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	if gnmiClient, ok := m.targets[targetID]; ok {
		return gnmiClient, nil
	}
	return nil, errors.NewNotFound("gnmi client for target %s not found", targetID)
}

func (m *connManager) Get(ctx context.Context, connID ConnID) (Conn, bool) {
	m.connsMu.RLock()
	defer m.connsMu.RUnlock()
	conn, ok := m.conns[connID]
	return conn, ok
}

func (m *connManager) Connect(ctx context.Context, target *topoapi.Object) error {
	m.connsMu.RLock()
	gnmiClient, ok := m.targets[target.ID]
	m.connsMu.RUnlock()
	if ok {
		return errors.NewAlreadyExists("target '%s' already exists", target.ID)
	}

	m.connsMu.Lock()
	defer m.connsMu.Unlock()

	gnmiClient, ok = m.targets[target.ID]
	if ok {
		return errors.NewAlreadyExists("target '%s' already exists", target.ID)
	}

	if target.Type != topoapi.Object_ENTITY {
		return errors.NewInvalid("object is not a topo entity %v+", target)
	}

	typeKindID := string(target.GetEntity().KindID)
	if len(typeKindID) == 0 {
		return errors.NewInvalid("target entity %s must have a 'kindID' to work with onos-config", target.ID)
	}

	destination, err := newDestination(target)
	if err != nil {
		log.Warnf("Failed to create a new target %s", err)
		return err
	}

	log.Infof("Connecting to gNMI target: %+v", destination)
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt32)),
	}
	gnmiClient, clientConn, err := connect(ctx, *destination, opts...)
	if err != nil {
		log.Warnf("Failed to connect to the gNMI target %s: %s", destination.Target, err)
		return err
	}

	m.targets[target.ID] = gnmiClient

	go func() {
		var conn Conn
		state := clientConn.GetState()
		switch state {
		case connectivity.Ready:
			conn = newConn(target.ID, gnmiClient)
			m.addConn(conn)
		}
		for clientConn.WaitForStateChange(context.Background(), state) {
			state = clientConn.GetState()
			log.Infof("Connection state changed for Target '%s': %s", target.ID, state)

			// If the channel is active, ensure a connection is added to the manager.
			// If the channel is idle, do not change its state (this can occur when
			// connected or disconnected).
			// In all other states, remove the connection from the manager.
			switch state {
			case connectivity.Ready:
				if conn == nil {
					conn = newConn(target.ID, gnmiClient)
					m.addConn(conn)
				}
			case connectivity.Idle:
				clientConn.Connect()
			default:
				if conn != nil {
					m.removeConn(conn.ID())
					conn = nil
				}
			}

			// If the channel is shutting down, exit the goroutine.
			switch state {
			case connectivity.Shutdown:
				return
			}
		}
	}()
	return nil
}

func (m *connManager) Disconnect(ctx context.Context, targetID topoapi.ID) error {
	m.connsMu.Lock()
	clientConn, ok := m.targets[targetID]
	if !ok {
		m.connsMu.Unlock()
		return errors.NewNotFound("target '%s' not found", targetID)
	}
	delete(m.targets, targetID)
	m.connsMu.Unlock()
	return clientConn.Close()
}

func (m *connManager) Watch(ctx context.Context, ch chan<- Conn) error {
	id := uuid.New()
	m.watchersMu.Lock()
	m.connsMu.Lock()
	m.watchers[id] = ch
	m.watchersMu.Unlock()

	go func() {
		for _, conn := range m.conns {
			ch <- conn
		}
		m.connsMu.Unlock()

		<-ctx.Done()
		m.watchersMu.Lock()
		delete(m.watchers, id)
		m.watchersMu.Unlock()
	}()
	return nil
}

func (m *connManager) addConn(conn Conn) {
	m.connsMu.Lock()
	m.conns[conn.ID()] = conn
	m.connsMu.Unlock()
	m.eventCh <- conn
}

func (m *connManager) removeConn(connID ConnID) {
	m.connsMu.Lock()
	if conn, ok := m.conns[connID]; ok {
		delete(m.conns, connID)
		m.connsMu.Unlock()
		m.eventCh <- conn
	} else {
		m.connsMu.Unlock()
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

func connect(ctx context.Context, d baseClient.Destination, opts ...grpc.DialOption) (*client, *grpc.ClientConn, error) {
	switch d.TLS {
	case nil:
		opts = append(opts, grpc.WithInsecure())
	default:
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(d.TLS)))
	}

	if d.Credentials != nil {
		secure := true
		if d.TLS == nil {
			secure = false
		}
		pc := newPassCred(d.Credentials.Username, d.Credentials.Password, secure)
		opts = append(opts, grpc.WithPerRPCCredentials(pc))
	}

	gCtx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	addr := ""
	if len(d.Addrs) != 0 {
		addr = d.Addrs[0]
	}
	conn, err := grpc.DialContext(gCtx, addr, opts...)
	if err != nil {
		return nil, nil, errors.NewInternal("Dialer(%s, %v): %v", addr, d.Timeout, err)
	}

	cl, err := gclient.NewFromConn(gCtx, conn, d)
	if err != nil {
		return nil, nil, err
	}

	gnmiClient := &client{
		client: cl,
	}

	return gnmiClient, conn, nil
}
