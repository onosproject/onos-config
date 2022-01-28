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
	"github.com/google/uuid"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	baseClient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"math"
	"sync"
	"time"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
)

const defaultKeepAliveInterval = 30 * time.Second

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
		targets:  make(map[topoapi.ID]*grpc.ClientConn),
		conns:    make(map[ConnID]Conn),
		watchers: make(map[uuid.UUID]chan<- Conn),
		eventCh:  make(chan Conn),
	}
	go mgr.processEvents()
	return mgr
}

type connManager struct {
	targets    map[topoapi.ID]*grpc.ClientConn
	conns      map[ConnID]Conn
	stateMu    sync.RWMutex
	watchers   map[uuid.UUID]chan<- Conn
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
	clientConn, ok := m.targets[target.ID]
	m.stateMu.RUnlock()
	if ok {
		return errors.NewAlreadyExists("target '%s' already exists", target.ID)
	}

	m.stateMu.Lock()
	defer m.stateMu.Unlock()

	clientConn, ok = m.targets[target.ID]
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
	m.targets[target.ID] = clientConn

	go func() {
		var conn Conn
		var keepAliveCancel context.CancelFunc

		state := clientConn.GetState()
		switch state {
		case connectivity.Connecting, connectivity.Ready, connectivity.Idle:
			conn = newConn(target.ID, gnmiClient)
			m.addConn(conn)
		}
		for clientConn.WaitForStateChange(context.Background(), state) {
			state = clientConn.GetState()

			// If the channel is idle, start the keep-alive goroutine.
			// Otherwise, shutdown the keep-alive goroutine while the channel is active.
			switch state {
			case connectivity.Idle:
				if conn == nil {
					conn = newConn(target.ID, gnmiClient)
					m.addConn(conn)
				}
				if keepAliveCancel == nil {
					ctx, cancel := context.WithCancel(context.Background())
					go func() {
						ticker := time.NewTicker(defaultKeepAliveInterval)
						for {
							select {
							case _, ok := <-ticker.C:
								if !ok {
									return
								}
								clientConn.Connect()
							case <-ctx.Done():
								ticker.Stop()
							}
						}
					}()
					keepAliveCancel = cancel
				}
			default:
				if keepAliveCancel != nil {
					keepAliveCancel()
					keepAliveCancel = nil
				}
			}

			// If the channel is active, ensure a connection is added to the manager.
			// If the channel failed, remove the connection from the manager.
			switch state {
			case connectivity.Connecting, connectivity.Ready, connectivity.Idle:
				if conn == nil {
					conn = newConn(target.ID, gnmiClient)
					m.addConn(conn)
				}
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
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	clientConn, ok := m.targets[targetID]
	if !ok {
		return errors.NewNotFound("target '%s' not found", targetID)
	}
	delete(m.targets, targetID)
	return clientConn.Close()
}

func (m *connManager) Watch(ctx context.Context, ch chan<- Conn) error {
	id := uuid.New()
	m.watchersMu.Lock()
	m.stateMu.Lock()
	m.watchers[id] = ch
	m.watchersMu.Unlock()

	go func() {
		for _, conn := range m.conns {
			ch <- conn
		}
		m.stateMu.Unlock()

		<-ctx.Done()
		m.watchersMu.Lock()
		delete(m.watchers, id)
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
