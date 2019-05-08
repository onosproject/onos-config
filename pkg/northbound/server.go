// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

package northbound

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"net"
	"sync"

	"github.com/onosproject/onos-config/pkg/certs"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

// Service provides service-specific registration for grpc services
type Service interface {
	Register(s *grpc.Server)
}

// Server provides NB gNMI server for onos-config
type Server struct {
	cfg      *ServerConfig
	services []Service
	mu       sync.Mutex
}

// ServerConfig comprises a set of server configuration options
type ServerConfig struct {
	CaPath   string
	KeyPath  string
	CertPath string
	Port     int16
	Insecure bool
}

// NewServer initializes gNMI server using the supplied configuration
func NewServer(cfg *ServerConfig) *Server {
	return &Server{
		services: []Service{},
		cfg:      cfg,
	}
}

// AddService adds a Service to the server to be registered on Serve
func (s *Server) AddService(r Service) {
	s.services = append(s.services, r)
}

// Serve starts the NB gNMI server
func (s *Server) Serve() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return err
	}

	tlsCfg := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	if s.cfg.CertPath == "" && s.cfg.KeyPath == "" {
		// Load default Certificates
		clientCerts, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
		if err != nil {
			log.Println("Error loading default certs")
		}
		tlsCfg.Certificates = []tls.Certificate{clientCerts}
	} else {
		clientCerts, err := tls.LoadX509KeyPair(s.cfg.CertPath, s.cfg.KeyPath)
		if err != nil {
			log.Println("Error loading default certs")
		}
		tlsCfg.Certificates = []tls.Certificate{clientCerts}
	}

	if s.cfg.Insecure {
		// RequestClientCert will ask client for a certificate but won't
		// require it to proceed. If certificate is provided, it will be
		// verified.
		tlsCfg.ClientAuth = tls.RequestClientCert
	}

	if s.cfg.CaPath == "" {
		log.Println("Loading default CA onfca")
		tlsCfg.ClientCAs = getCertPoolDefault()
	} else {
		tlsCfg.ClientCAs = getCertPool(s.cfg.CaPath)
	}

	opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}
	grpc := grpc.NewServer(opts...)
	for i := range s.services {
		s.services[i].Register(grpc)
	}

	glog.Infof("Starting RPC server on address: %s", lis.Addr().String())
	return grpc.Serve(lis)
}

func getCertPoolDefault() *x509.CertPool {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(certs.OnfCaCrt)); !ok {
		log.Println("failed to append CA certificates")
	}
	return certPool
}

func getCertPool(CaPath string) *x509.CertPool {
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(CaPath)
	if err != nil {
		log.Println("could not read", CaPath, err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Println("failed to append CA certificates")
	}
	return certPool
}
