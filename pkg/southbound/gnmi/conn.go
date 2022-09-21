// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"time"

	"github.com/onosproject/onos-config/pkg/controller/utils"

	"github.com/google/uuid"
	"github.com/onosproject/onos-lib-go/pkg/uri"

	"google.golang.org/grpc"

	"github.com/onosproject/onos-lib-go/pkg/certs"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	"crypto/tls"

	"github.com/onosproject/onos-lib-go/pkg/logging"
	baseClient "github.com/openconfig/gnmi/client"
)

func init() {
	uuid.SetNodeID([]byte(utils.GetOnosConfigID()))
}

const defaultTimeout = 60 * time.Second

var log = logging.GetLogger("southbound", "gnmi")

// ConnID connection ID
type ConnID string

// Conn gNMI connection interface
type Conn interface {
	Client
	ID() ConnID
	TargetID() topoapi.ID
}

func newDestination(target *topoapi.Object) (*baseClient.Destination, error) {
	configurable := &topoapi.Configurable{}
	err := target.GetAspect(configurable)
	if err != nil {
		return nil, errors.NewInvalid("target entity %s must have 'onos.topo.Configurable' aspect to work with onos-config", target.ID)
	}

	tlsOptions := &topoapi.TLSOptions{}
	err = target.GetAspect(tlsOptions)
	if err != nil {
		return nil, errors.NewInvalid("topo entity %s must have 'onos.topo.TLSOptions' aspect to work with onos-config", target.ID)
	}

	timeout := defaultTimeout
	if configurable.Timeout != nil {
		timeout = *configurable.Timeout
	}

	destination := &baseClient.Destination{
		Addrs:   []string{configurable.Address},
		Target:  string(target.ID),
		Timeout: timeout,
	}

	if tlsOptions.Plain {
		log.Infof("Plain (non TLS) connection to %s", configurable.Address)
	} else {
		log.Infof("TLS connection to %s", configurable.Address)
		tlsConfig := &tls.Config{}
		if tlsOptions.Insecure {
			log.Infof("Skipping verification of TLS certificate chain for target %s", configurable.Address)
			tlsConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			log.Info("Enabled verification of TLS certificate chain for target %s", configurable.Address)
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
			log.Info("Loading default TLS certificates")
			clientCerts, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
			if err != nil {
				return nil, err
			}
			tlsConfig.Certificates = []tls.Certificate{clientCerts}
		} else if tlsOptions.Cert != "" && tlsOptions.Key != "" {
			// Load certs given for device
			log.Info("Loading the given TLS certs for the device")
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

func newConnID() ConnID {
	connID := ConnID(uri.NewURI(
		uri.WithScheme("uuid"),
		uri.WithOpaque(uuid.New().String())).String())
	return connID
}

func newConn(targetID topoapi.ID, client *client) Conn {
	return &conn{
		client:   client,
		id:       newConnID(),
		targetID: targetID,
	}
}

// conn gNMI Connection
type conn struct {
	*client
	clientConn *grpc.ClientConn
	id         ConnID
	targetID   topoapi.ID
}

// ID returns the gNMI connection ID
func (c *conn) ID() ConnID {
	return c.id
}

// TargetID returns target ID associated with this connection
func (c *conn) TargetID() topoapi.ID {
	return c.targetID
}

var _ Conn = &conn{}
