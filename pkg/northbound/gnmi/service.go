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
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/device/state"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"io/ioutil"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
)

// Service implements Service for GNMI
type Service struct {
	northbound.Service
	deviceChangesStore        device.Store
	modelRegistry             *modelregistry.ModelRegistry
	pluginRegistry            *pluginregistry.PluginRegistry
	deviceCache               cache.Cache
	networkChangesStore       network.Store
	dispatcher                *dispatcher.Dispatcher
	deviceStore               devicestore.Store
	deviceStateStore          state.Store
	operationalStateCache     *map[topodevice.ID]devicechange.TypedValueMap
	operationalStateCacheLock *sync.RWMutex
	allowUnvalidatedConfig    bool
}

// NewService allocates a Service struct with the given parameters
func NewService(modelRegistry *modelregistry.ModelRegistry,
	pluginRegistry *pluginregistry.PluginRegistry,
	deviceChangesStore device.Store,
	deviceCache cache.Cache,
	networkChangesStore network.Store,
	dispatcher *dispatcher.Dispatcher,
	deviceStore devicestore.Store,
	deviceStateStore state.Store,
	operationalStateCache *map[topodevice.ID]devicechange.TypedValueMap,
	operationalStateCacheLock *sync.RWMutex,
	allowUnvalidatedConfig bool) Service {
	return Service{
		deviceChangesStore:        deviceChangesStore,
		modelRegistry:             modelRegistry,
		pluginRegistry:            pluginRegistry,
		deviceCache:               deviceCache,
		networkChangesStore:       networkChangesStore,
		dispatcher:                dispatcher,
		deviceStore:               deviceStore,
		deviceStateStore:          deviceStateStore,
		operationalStateCache:     operationalStateCache,
		operationalStateCacheLock: operationalStateCacheLock,
		allowUnvalidatedConfig:    allowUnvalidatedConfig,
	}
}

// Register registers the GNMI server with grpc
func (s Service) Register(r *grpc.Server) {
	gnmi.RegisterGNMIServer(r,
		&Server{
			deviceChangesStore:        s.deviceChangesStore,
			modelRegistry:             s.modelRegistry,
			pluginRegistry:            s.pluginRegistry,
			deviceCache:               s.deviceCache,
			networkChangesStore:       s.networkChangesStore,
			dispatcher:                s.dispatcher,
			deviceStore:               s.deviceStore,
			deviceStateStore:          s.deviceStateStore,
			operationalStateCache:     s.operationalStateCache,
			operationalStateCacheLock: s.operationalStateCacheLock,
			allowUnvalidatedConfig:    s.allowUnvalidatedConfig,
		})
}

// Server implements the grpc GNMI service
type Server struct {
	mu                        sync.RWMutex
	lastWrite                 networkchange.Revision
	deviceChangesStore        device.Store
	modelRegistry             *modelregistry.ModelRegistry
	pluginRegistry            *pluginregistry.PluginRegistry
	deviceCache               cache.Cache
	networkChangesStore       network.Store
	deviceStore               devicestore.Store
	dispatcher                *dispatcher.Dispatcher
	deviceStateStore          state.Store
	operationalStateCache     *map[topodevice.ID]devicechange.TypedValueMap
	operationalStateCacheLock *sync.RWMutex
	allowUnvalidatedConfig    bool
}

// Capabilities implements gNMI Capabilities
func (s *Server) Capabilities(ctx context.Context, req *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	capabilities, err := s.modelRegistry.Capabilities()
	if err != nil {
		return nil, err
	}
	v, _ := getGNMIServiceVersion()
	return &gnmi.CapabilityResponse{
		SupportedModels:    capabilities,
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_JSON, gnmi.Encoding_JSON_IETF, gnmi.Encoding_PROTO},
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
