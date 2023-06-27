// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package gnmi implements the northbound gNMI service for the configuration subsystem.
package gnmi

import (
	"context"
	"os"
	"strconv"
	"sync"

	"github.com/onosproject/onos-lib-go/pkg/logging"

	proposal "github.com/onosproject/onos-config/pkg/store/v2/proposal"

	"github.com/golang/protobuf/proto"
	protobuf "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configuration "github.com/onosproject/onos-config/pkg/store/v2/configuration"

	"github.com/onosproject/onos-config/pkg/pluginregistry"

	"github.com/onosproject/onos-config/pkg/store/topo"
	transaction "github.com/onosproject/onos-config/pkg/store/v2/transaction"

	sb "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound", "gnmi")

// Service implements Service for GNMI
type Service struct {
	northbound.Service
	pluginRegistry pluginregistry.PluginRegistry
	topo           topo.Store
	transactions   transaction.Store
	proposals      proposal.Store
	configurations configuration.Store
	conns          sb.ConnManager
}

// NewService allocates a Service struct with the given parameters
func NewService(
	topo topo.Store,
	transactions transaction.Store,
	proposals proposal.Store,
	configurations configuration.Store,
	pluginRegistry pluginregistry.PluginRegistry, conns sb.ConnManager) Service {
	return Service{
		pluginRegistry: pluginRegistry,
		topo:           topo,
		transactions:   transactions,
		proposals:      proposals,
		configurations: configurations,
		conns:          conns,
	}
}

// Register registers the GNMI server with grpc
func (s Service) Register(r *grpc.Server) {
	setSizeLimitStr := os.Getenv("GNMI_SET_SIZE_LIMIT")
	var setSizeLimit int
	var err error
	if setSizeLimit, err = strconv.Atoi(setSizeLimitStr); err != nil {
		log.Errorf("Ignoring GNMI_SET_SIZE_LIMIT: %s %v", setSizeLimitStr, err)
	}

	gnmi.RegisterGNMIServer(r,
		&Server{
			pluginRegistry:   s.pluginRegistry,
			topo:             s.topo,
			transactions:     s.transactions,
			proposals:        s.proposals,
			configurations:   s.configurations,
			conns:            s.conns,
			gnmiSetSizeLimit: setSizeLimit,
		})
}

// Server implements the grpc GNMI service
type Server struct {
	mu               sync.RWMutex
	pluginRegistry   pluginregistry.PluginRegistry
	topo             topo.Store
	transactions     transaction.Store
	proposals        proposal.Store
	configurations   configuration.Store
	conns            sb.ConnManager
	gnmiSetSizeLimit int
}

// Capabilities implements gNMI Capabilities
func (s *Server) Capabilities(ctx context.Context, req *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	plugins := s.pluginRegistry.GetPlugins()

	supportedModels := make([]*gnmi.ModelData, 0)
	uniqueModels := make(map[string]*gnmi.ModelData)
	for _, plugin := range plugins {
		capabilities := plugin.Capabilities(ctx)
		for _, model := range capabilities.SupportedModels {
			modelKey := model.Name + "!" + model.Version
			if uniqueModels[modelKey] == nil {
				supportedModels = append(supportedModels, model)
				uniqueModels[modelKey] = model
			}
		}
	}

	v, err := getGNMIServiceVersion()
	if err != nil {
		return nil, errors.Status(err).Err()
	}
	return &gnmi.CapabilityResponse{
		SupportedModels:    supportedModels,
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_JSON, gnmi.Encoding_JSON_IETF, gnmi.Encoding_PROTO},
		GNMIVersion:        v,
	}, nil
}

// getGNMIServiceVersion returns a pointer to the gNMI service version string.
// The method is non-trivial because of the way it is defined in the proto file.
func getGNMIServiceVersion() (string, error) {
	parentFile := (&gnmi.Update{}).ProtoReflect().Descriptor().ParentFile()
	options := parentFile.Options()
	version := ""
	if fileOptions, ok := options.(*protobuf.FileOptions); ok {
		ver, err := proto.GetExtension(fileOptions, gnmi.E_GnmiService)
		if err != nil {
			return "", errors.NewInvalid(err.Error())
		}
		version = *ver.(*string)
	}
	return version, nil
}
