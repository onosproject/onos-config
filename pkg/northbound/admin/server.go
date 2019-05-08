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

package admin

import (
	"context"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"

	"github.com/onosproject/onos-config/pkg/certs"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/onosproject/onos-config/pkg/models"
	pb "github.com/onosproject/onos-config/pkg/northbound/proto"
)

// Server provides NB gNMI server for onos-config
type Server struct {
	lis    net.Listener
	grpc   *grpc.Server
	mu     sync.Mutex
	models *models.Models
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
func NewServer(cfg *ServerConfig) (*Server, error) {
	var err error

	s := &Server{
		models: models.NewModels(),
	}

	s.grpc = grpc.NewServer()
	reflection.Register(s.grpc)

	s.lis, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to open listener port %d: %v", cfg.Port, err)
	}

	pb.RegisterNorthboundServer(s.grpc, s)

	return s, nil
}

// Serve starts the NB gNMI server
func (s *Server) Serve() error {
	glog.Infof("Starting RPC server on address: %s", s.lis.Addr().String())
	return s.grpc.Serve(s.lis)
}

func (s *Server) SayHello(ctx context.Context, req *pb.HelloWorldRequest) (*pb.HelloWorldResponse, error) {
	return &pb.HelloWorldResponse{}, nil
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
