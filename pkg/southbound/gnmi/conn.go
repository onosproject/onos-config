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
	"time"

	"google.golang.org/grpc/connectivity"

	"google.golang.org/grpc"

	"github.com/onosproject/onos-lib-go/pkg/certs"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"crypto/tls"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	baseClient "github.com/openconfig/gnmi/client"
)

var log = logging.GetLogger("southbound", "gnmi")

// ConnID connection ID
type ConnID string

// Conn gNMI connection interface
type Conn interface {
	Client
	ID() ConnID
	TargetID() topoapi.ID
	GetClientConn() *grpc.ClientConn
	WatchConnState(ctx context.Context, ch chan<- connectivity.State) error
}

// conn gNMI Connection
type conn struct {
	*client
	clientConn          *grpc.ClientConn
	id                  ConnID
	targetID            topoapi.ID
	connStateWatchers   []chan<- connectivity.State
	connStateWatchersMu sync.RWMutex
	connStateEventCh    chan connectivity.State
}

func newDestination(target *topoapi.Object) (*baseClient.Destination, error) {
	asset := &topoapi.Asset{}
	err := target.GetAspect(asset)
	if err != nil {
		return nil, errors.NewInvalid("target entity %s must have 'onos.topo.Asset' aspect to work with onos-config", target.ID)
	}

	configurable := &topoapi.Configurable{}
	err = target.GetAspect(configurable)
	if err != nil {
		return nil, errors.NewInvalid("target entity %s must have 'onos.topo.Configurable' aspect to work with onos-config", target.ID)
	}

	mastership := &topoapi.MastershipState{}
	err = target.GetAspect(mastership)
	if err != nil {
		return nil, errors.NewInvalid("topo entity %s must have 'onos.topo.MastershipState' aspect to work with onos-config", target.ID)
	}

	tlsOptions := &topoapi.TLSOptions{}
	err = target.GetAspect(tlsOptions)
	if err != nil {
		return nil, errors.NewInvalid("topo entity %s must have 'onos.topo.TLSOptions' aspect to work with onos-config", target.ID)
	}

	destination := &baseClient.Destination{
		Addrs:   []string{configurable.Address},
		Target:  string(target.ID),
		Timeout: time.Duration(configurable.Timeout),
	}

	if tlsOptions.Plain {
		log.Info("Plain (non TLS) connection to ", configurable.Address)
	} else {
		tlsConfig := &tls.Config{}
		if tlsOptions.Insecure {
			log.Info("Insecure TLS connection to ", configurable.Address)
			tlsConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			log.Info("Secure TLS connection to ", configurable.Address)
		}
		if tlsOptions.CaCert == "" {
			log.Info("Loading default CA onfca")
			defaultCertPool, err := certs.GetCertPoolDefault()
			if err != nil {
				return nil, err
			}
			tlsConfig.RootCAs = defaultCertPool
		} else {
			certPool, err := certs.GetCertPool(tlsOptions.CaCert)
			if err != nil {
				return nil, err
			}
			tlsConfig.RootCAs = certPool
		}
		if tlsOptions.Cert == "" && tlsOptions.Key == "" {
			log.Info("Loading default certificates")
			clientCerts, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{clientCerts}
		} else if tlsOptions.Cert != "" && tlsOptions.Key != "" {
			// Load certs given for device
			tlsConfig.Certificates = []tls.Certificate{setCertificate(tlsOptions.Cert, tlsOptions.Key)}
		} else {
			log.Errorf("Can't load Ca=%s , Cert=%s , key=%s for %v, trying with insecure connection",
				tlsOptions.CaCert, tlsOptions.Cert, tlsOptions.Key, configurable.Address)
			tlsConfig = &tls.Config{InsecureSkipVerify: true}
		}
		destination.TLS = tlsConfig
	}

	err = destination.Validate()
	if err != nil {
		return nil, err
	}

	return destination, nil
}

// ID returns the gNMI connection ID
func (c *conn) ID() ConnID {
	return c.id
}

// TargetID returns target ID associated with this connection
func (c *conn) TargetID() topoapi.ID {
	return c.targetID
}

// GetClientConn gets client connection
func (c *conn) GetClientConn() *grpc.ClientConn {
	return c.clientConn
}

func (c *conn) WatchConnState(ctx context.Context, ch chan<- connectivity.State) error {
	c.connStateWatchersMu.Lock()
	c.connStateWatchers = append(c.connStateWatchers, ch)
	c.connStateWatchersMu.Unlock()

	go func() {
		<-ctx.Done()
		c.connStateWatchersMu.Lock()
		connStateWatchers := make([]chan<- connectivity.State, 0, len(c.connStateWatchers)-1)
		for _, connStateWatcher := range connStateWatchers {
			if connStateWatcher != ch {
				connStateWatchers = append(connStateWatchers, connStateWatcher)
			}
		}
		c.connStateWatchers = connStateWatchers
		c.connStateWatchersMu.Unlock()
	}()
	return nil

}

func (c *conn) processConnStateEvents() {
	log.Infof("Starting processing of connection state events for connection: %s", c.id)
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		state := c.clientConn.GetState()
		log.Infof("Initial connection state for connection %s is %s", c.id, state.String())
		c.processConnStateEvent(state)
		for c.clientConn.WaitForStateChange(ctx, state) {
			state = c.clientConn.GetState()
			log.Infof("Connection state is changed for connection: %s, current state: %s", c.id, state.String())
			c.processConnStateEvent(state)
		}
	}()

}

func (c *conn) processConnStateEvent(state connectivity.State) {
	log.Infof("Notifying connection state for connection: %s", c.id)
	c.connStateWatchersMu.RLock()
	for _, connStateWatcher := range c.connStateWatchers {
		connStateWatcher <- state
	}
	c.connStateWatchersMu.RUnlock()
}

var _ Conn = &conn{}
