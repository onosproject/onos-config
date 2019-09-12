// Copyright 2019-present Open Networking Foundation.
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

// Package northbound houses implementations of various application-oriented interfaces
// for the ONOS configuration subsystem.
package northbound

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/onosproject/onos-config/pkg/certs"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	log "k8s.io/klog"
	"net"

	"google.golang.org/grpc"
)

// Service provides service-specific registration for grpc services.
type Service interface {
	Register(s *grpc.Server)
}

// Server provides NB gNMI server for onos-config.
type Server struct {
	cfg      *ServerConfig
	services []Service
}

// ServerConfig comprises a set of server configuration options.
type ServerConfig struct {
	CaPath   *string
	KeyPath  *string
	CertPath *string
	Port     int16
	Insecure bool
}

// NewServer initializes gNMI server using the supplied configuration.
func NewServer(cfg *ServerConfig) *Server {
	return &Server{
		services: []Service{},
		cfg:      cfg,
	}
}

// NewServerConfig creates a server config created with the specified end-point security details.
func NewServerConfig(caPath string, keyPath string, certPath string) *ServerConfig {
	return &ServerConfig{
		Port:     5150,
		Insecure: true,
		CaPath:   &caPath,
		KeyPath:  &keyPath,
		CertPath: &certPath,
	}
}

// AddService adds a Service to the server to be registered on Serve.
func (s *Server) AddService(r Service) {
	s.services = append(s.services, r)
}

// Serve starts the NB gNMI server.
func (s *Server) Serve(started func(string)) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return err
	}

	tlsCfg := &tls.Config{}

	if *s.cfg.CertPath == "" && *s.cfg.KeyPath == "" {
		// Load default Certificates
		clientCerts, err := tls.X509KeyPair([]byte(certs.DefaultLocalhostCrt), []byte(certs.DefaultLocalhostKey))
		if err != nil {
			log.Error("Error loading default certs")
			return err
		}
		tlsCfg.Certificates = []tls.Certificate{clientCerts}
	} else {
		log.Infof("Loading certs: %s %s", *s.cfg.CertPath, *s.cfg.KeyPath)
		clientCerts, err := tls.LoadX509KeyPair(*s.cfg.CertPath, *s.cfg.KeyPath)
		if err != nil {
			log.Info("Error loading default certs")
		}
		tlsCfg.Certificates = []tls.Certificate{clientCerts}
	}

	if s.cfg.Insecure {
		// RequestClientCert will ask client for a certificate but won't
		// require it to proceed. If certificate is provided, it will be
		// verified.
		tlsCfg.ClientAuth = tls.RequestClientCert
	} else {
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if *s.cfg.CaPath == "" {
		log.Info("Loading default CA onfca")
		tlsCfg.ClientCAs = getCertPoolDefault()
	} else {
		tlsCfg.ClientCAs = getCertPool(*s.cfg.CaPath)
	}

	opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}
	server := grpc.NewServer(opts...)
	for i := range s.services {
		s.services[i].Register(server)
	}
	started(lis.Addr().String())

	log.Infof("Starting RPC server on address: %s", lis.Addr().String())
	return server.Serve(lis)
}

func getCertPoolDefault() *x509.CertPool {
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM([]byte(certs.OnfCaCrt)); !ok {
		log.Warning("failed to append CA certificates")
	}
	return certPool
}

func getCertPool(CaPath string) *x509.CertPool {
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(CaPath)
	if err != nil {
		log.Warning("could not read ", CaPath, err)
	}
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		log.Warning("failed to append CA certificates")
	}
	return certPool
}
