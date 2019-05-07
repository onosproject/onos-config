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

package gnmi

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/certs"

	glog "github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/onosproject/onos-config/pkg/models"
	pb "github.com/openconfig/gnmi/proto/gnmi"
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

	tlsCfg := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	if cfg.CertPath == "" && cfg.KeyPath == "" {
		// Load default Certificates
		clientCerts, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
		if err != nil {
			log.Println("Error loading default certs")
		}
		tlsCfg.Certificates = []tls.Certificate{clientCerts}
	} else {
		clientCerts, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
		if err != nil {
			log.Println("Error loading default certs")
		}
		tlsCfg.Certificates = []tls.Certificate{clientCerts}
	}

	if cfg.Insecure {
		// RequestClientCert will ask client for a certificate but won't
		// require it to proceed. If certificate is provided, it will be
		// verified.
		tlsCfg.ClientAuth = tls.RequestClientCert
	}

	if cfg.CaPath == "" {
		log.Println("Loading default CA onfca")
		tlsCfg.ClientCAs = getCertPoolDefault()
	} else {
		tlsCfg.ClientCAs = getCertPool(cfg.CaPath)
	}

	opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsCfg))}
	s.grpc = grpc.NewServer(opts...)
	reflection.Register(s.grpc)

	s.lis, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to open listener port %d: %v", cfg.Port, err)
	}

	pb.RegisterGNMIServer(s.grpc, s)

	return s, nil
}

// Serve starts the NB gNMI server
func (s *Server) Serve() error {
	glog.Infof("Starting RPC server on address: %s", s.lis.Addr().String())
	return s.grpc.Serve(s.lis)
}

// Capabilities implements gNMI Capabilities
func (s *Server) Capabilities(ctx context.Context, req *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
	v, _ := getGNMIServiceVersion()
	return &pb.CapabilityResponse{
		SupportedModels:    s.models.Get(),
		SupportedEncodings: []pb.Encoding{pb.Encoding_JSON},
		GNMIVersion:        *v,
	}, nil
}

// Get mplements gNMI Get
func (s *Server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	return &pb.GetResponse{}, nil
}

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *pb.SetRequest) (*pb.SetResponse, error) {
	return &pb.SetResponse{}, nil
}

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream pb.GNMI_SubscribeServer) error {
	return nil
}

// getGNMIServiceVersion returns a pointer to the gNMI service version string.
// The method is non-trivial because of the way it is defined in the proto file.
func getGNMIServiceVersion() (*string, error) {
	gzB, _ := (&pb.Update{}).Descriptor()
	r, err := gzip.NewReader(bytes.NewReader(gzB))
	if err != nil {
		return nil, fmt.Errorf("error in initializing gzip reader: %v", err)
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error in reading gzip data: %v", err)
	}
	desc := &dpb.FileDescriptorProto{}
	if err := proto.Unmarshal(b, desc); err != nil {
		return nil, fmt.Errorf("error in unmarshaling proto: %v", err)
	}
	ver, err := proto.GetExtension(desc.Options, pb.E_GnmiService)
	if err != nil {
		return nil, fmt.Errorf("error in getting version from proto extension: %v", err)
	}
	return ver.(*string), nil
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
