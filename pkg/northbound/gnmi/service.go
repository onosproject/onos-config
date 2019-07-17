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

// Package gnmi implements the northbound gNMI service for the configuration subsystem.
package gnmi

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/onosproject/onos-config/pkg/manager"
	"io/ioutil"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
)

// Service implements Service for GNMI
type Service struct {
	northbound.Service
}

// Register registers the GNMI server with grpc
func (s Service) Register(r *grpc.Server) {
	gnmi.RegisterGNMIServer(r, &Server{})
}

// Server implements the grpc GNMI service
type Server struct {
	mu sync.RWMutex
}

// Capabilities implements gNMI Capabilities
func (s *Server) Capabilities(ctx context.Context, req *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	v, _ := getGNMIServiceVersion()
	return &gnmi.CapabilityResponse{
		SupportedModels:    manager.GetManager().ModelRegistry.Capabilities(),
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_JSON},
		GNMIVersion:        *v,
	}, nil
}

// getGNMIServiceVersion returns a pointer to the gNMI service version string.
// The method is non-trivial because of the way it is defined in the proto file.
func getGNMIServiceVersion() (*string, error) {
	gzB, _ := (&gnmi.Update{}).Descriptor()
	r, err := gzip.NewReader(bytes.NewReader(gzB))
	if err != nil {
		return nil, fmt.Errorf("error in initializing gzip reader: %v", err)
	}
	defer r.Close()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("error in reading gzip data: %v", err)
	}
	desc := &descriptor.FileDescriptorProto{}
	if err := proto.Unmarshal(b, desc); err != nil {
		return nil, fmt.Errorf("error in unmarshaling proto: %v", err)
	}
	ver, err := proto.GetExtension(desc.Options, gnmi.E_GnmiService)
	if err != nil {
		return nil, fmt.Errorf("error in getting version from proto extension: %v", err)
	}
	return ver.(*string), nil
}
