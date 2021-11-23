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
	"time"

	"github.com/onosproject/onos-lib-go/pkg/certs"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/uri"

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
}

// conn gNMI Connection
type conn struct {
	*client
	id     ConnID
	cancel context.CancelFunc
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

// newGNMIConnection creates a new gNMI connection
func newGNMIConnection(target *topoapi.Object) (Conn, error) {
	connID := ConnID(uri.NewURI(
		uri.WithScheme("gnmi"),
		uri.WithOpaque(string(target.ID))).String())

	if target.Type != topoapi.Object_ENTITY {
		return nil, errors.NewInvalid("object is not a topo entity %v+", target)
	}

	typeKindID := string(target.GetEntity().KindID)
	if len(typeKindID) == 0 {
		return nil, errors.NewInvalid("target entity %s must have a 'kindID' to work with onos-config", target.ID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	destination, err := newDestination(target)
	if err != nil {
		log.Warnf("Failed to create a new target %s", err)
		cancel()
		return nil, err
	}
	log.Infof("Connecting to gNMI target: %+v", destination)
	gnmiClient, err := newGNMIClient(ctx, *destination)
	if err != nil {
		log.Warnf("Failed to connect to the gNMI target %s: %s", destination.Target, err)
		cancel()
		return nil, err
	}

	return &conn{
		client: gnmiClient,
		id:     connID,
		cancel: cancel,
	}, nil
}

// ID returns the gNMI connection ID
func (c *conn) ID() ConnID {
	return c.id
}

var _ Conn = &conn{}
