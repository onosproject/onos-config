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

// Package admin implements the northbound administrative gRPC service for the configuration subsystem.
package admin

import (
	"context"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	v2 "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"github.com/onosproject/onos-config/pkg/store/transaction"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/onosproject/onos-lib-go/pkg/uri"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound", "admin")

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
	transactionsStore   transaction.Store
	configurationsStore configuration.Store
	pluginRegistry      *pluginregistry.PluginRegistry
}

// NewService allocates a Service struct with the given parameters
func NewService(transactionsStore transaction.Store, configurationsStore configuration.Store, pluginRegistry *pluginregistry.PluginRegistry) Service {
	return Service{
		transactionsStore:   transactionsStore,
		configurationsStore: configurationsStore,
		pluginRegistry:      pluginRegistry,
	}
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := Server{
		transactionsStore:   s.transactionsStore,
		configurationsStore: s.configurationsStore,
		pluginRegistry:      s.pluginRegistry}
	admin.RegisterConfigAdminServiceServer(r, server)
	admin.RegisterConfigurationServiceServer(r, server)
	admin.RegisterTransactionServiceServer(r, server)
}

// Server implements the gRPC service for administrative facilities.
type Server struct {
	transactionsStore   transaction.Store
	configurationsStore configuration.Store
	pluginRegistry      *pluginregistry.PluginRegistry
}

// ListRegisteredModels lists the registered models..
func (s Server) ListRegisteredModels(r *admin.ListModelsRequest, stream admin.ConfigAdminService_ListRegisteredModelsServer) error {
	if stream.Context() != nil {
		if md := metautils.ExtractIncoming(stream.Context()); md != nil && md.Get("name") != "" {
			log.Infof("admin ListSnapshots() called by '%s (%s)'. Groups [%v]. Token %s",
				md.Get("name"), md.Get("email"), md.Get("groups"), md.Get("at_hash"))
		}
	}
	log.Infow("ListRegisteredModels called with:",
		"ModelName", r.ModelName,
		"ModelVersion", r.ModelVersion,
		"Verbose", r.Verbose)

	// TODO support filters

	plugins := s.pluginRegistry.GetPlugins()
	for _, p := range plugins {
		log.Infow("Found plugin",
			"ID", p.ID,
			"Name", p.Info.Name,
			"Version", p.Info.Version,
		)
		msg := &admin.ModelPlugin{
			Id:       p.ID,
			Endpoint: p.Endpoint,
			Info:     &p.Info,
			Status:   p.Status.String(),
			Error:    p.Error,
		}
		err := stream.Send(msg)
		if err != nil {
			log.Errorf("error sending ModelInfo from plugin %v: %v", p.ID, err)
			return err
		}
	}
	return nil
}

// RollbackTransaction rolls back configuration change transaction with the specified index.
func (s Server) RollbackTransaction(ctx context.Context, req *admin.RollbackRequest) (*admin.RollbackResponse, error) {
	id := v2.TransactionID(uri.NewURI(uri.WithScheme("uuid"), uri.WithOpaque(uuid.New().String())).String())
	t := &v2.Transaction{
		ID: id,
		Transaction: &v2.Transaction_Rollback{
			Rollback: &v2.TransactionRollback{
				Index: req.Index,
			},
		},
	}
	if err := s.transactionsStore.Create(ctx, t); err != nil {
		log.Errorf("Unable to rollback transaction with index %d: %+v", req.Index, err)
		return nil, errors.Status(err).Err()
	}
	return &admin.RollbackResponse{ID: t.ID, Index: t.Index}, nil
}

// ListSnapshots lists snapshots for all devices
func (s Server) ListSnapshots(r *admin.ListSnapshotsRequest, stream admin.ConfigAdminService_ListSnapshotsServer) error {
	return errors.NewNotSupported("not implemented")
}

// CompactChanges takes a snapshot of all devices
func (s Server) CompactChanges(ctx context.Context, request *admin.CompactChangesRequest) (*admin.CompactChangesResponse, error) {
	return nil, errors.NewNotSupported("not implemented")
}

// UploadRegisterModel uploads and registers a new model plugin.
// Deprecated: models should only be loaded at startup
func (s Server) UploadRegisterModel(stream admin.ConfigAdminService_UploadRegisterModelServer) error {
	return errors.NewNotSupported("dynamic model registration has been deprecated")
}
