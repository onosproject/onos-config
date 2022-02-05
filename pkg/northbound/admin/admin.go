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
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
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
	pluginRegistry      pluginregistry.PluginRegistry
}

// NewService allocates a Service struct with the given parameters
func NewService(transactionsStore transaction.Store, configurationsStore configuration.Store, pluginRegistry pluginregistry.PluginRegistry) Service {
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
	pluginRegistry      pluginregistry.PluginRegistry
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
	for _, plugin := range plugins {
		p := plugin.GetInfo()
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
	log.Debugf("Received RollbackRequest %+v", req)
	id := configapi.TransactionID(uri.NewURI(uri.WithScheme("uuid"), uri.WithOpaque(uuid.New().String())).String())
	t := &configapi.Transaction{
		ID: id,
		Details: &configapi.Transaction_Rollback{
			Rollback: &configapi.RollbackTransaction{
				RollbackIndex: req.Index,
			},
		},
	}
	if err := s.transactionsStore.Create(ctx, t); err != nil {
		log.Errorf("Unable to rollback transaction with index %d: %+v", req.Index, err)
		return nil, errors.Status(err).Err()
	}
	eventCh := make(chan configapi.TransactionEvent)
	err := s.transactionsStore.Watch(ctx, eventCh, transaction.WithReplay(), transaction.WithTransactionID(t.ID))
	if err != nil {
		return nil, errors.Status(err).Err()
	}
	for transactionEvent := range eventCh {
		if (transactionEvent.Transaction.TransactionStrategy.Synchronicity == configapi.TransactionStrategy_ASYNCHRONOUS &&
			transactionEvent.Transaction.Status.State == configapi.TransactionStatus_COMMITTED) ||
			(transactionEvent.Transaction.TransactionStrategy.Synchronicity == configapi.TransactionStrategy_SYNCHRONOUS &&
				transactionEvent.Transaction.Status.State == configapi.TransactionStatus_APPLIED) {
			response := &admin.RollbackResponse{ID: t.ID, Index: t.Index}
			log.Debugf("Sending RollbackResponse %+v", response)
			return response, nil
		} else if transactionEvent.Transaction.Status.State == configapi.TransactionStatus_FAILED {
			var err error
			if transactionEvent.Transaction.Status.Failure != nil {
				switch transactionEvent.Transaction.Status.Failure.Type {
				case configapi.Failure_UNKNOWN:
					err = errors.NewUnknown(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_CANCELED:
					err = errors.NewCanceled(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_NOT_FOUND:
					err = errors.NewNotFound(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_ALREADY_EXISTS:
					err = errors.NewAlreadyExists(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_UNAUTHORIZED:
					err = errors.NewUnauthorized(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_FORBIDDEN:
					err = errors.NewForbidden(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_CONFLICT:
					err = errors.NewConflict(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_INVALID:
					err = errors.NewInvalid(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_UNAVAILABLE:
					err = errors.NewUnavailable(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_NOT_SUPPORTED:
					err = errors.NewNotSupported(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_TIMEOUT:
					err = errors.NewTimeout(transactionEvent.Transaction.Status.Failure.Description)
				case configapi.Failure_INTERNAL:
					err = errors.NewInternal(transactionEvent.Transaction.Status.Failure.Description)
				default:
					err = errors.NewUnknown(transactionEvent.Transaction.Status.Failure.Description)
				}
			} else {
				err = errors.NewUnknown("unknown failure occurred")
			}
			log.Errorf("Transaction failed", err)
			return nil, errors.Status(err).Err()
		}
		switch transactionEvent.Transaction.Status.State {
		case configapi.TransactionStatus_APPLIED:
			response := &admin.RollbackResponse{ID: t.ID, Index: t.Index}
			log.Debugf("Sending RollbackResponse %+v", response)
			return response, nil
		case configapi.TransactionStatus_FAILED:
			err := getErrorFromFailure(transactionEvent.Transaction.Status.Failure)
			log.Errorf("Transaction failed", err)
			return nil, errors.Status(err).Err()
		}
	}
	return nil, ctx.Err()
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

func getErrorFromFailure(failure *configapi.Failure) error {
	if failure == nil {
		return errors.NewUnknown("unknown failure occurred")
	}

	switch failure.Type {
	case configapi.Failure_UNKNOWN:
		return errors.NewUnknown(failure.Description)
	case configapi.Failure_CANCELED:
		return errors.NewCanceled(failure.Description)
	case configapi.Failure_NOT_FOUND:
		return errors.NewNotFound(failure.Description)
	case configapi.Failure_ALREADY_EXISTS:
		return errors.NewAlreadyExists(failure.Description)
	case configapi.Failure_UNAUTHORIZED:
		return errors.NewUnauthorized(failure.Description)
	case configapi.Failure_FORBIDDEN:
		return errors.NewForbidden(failure.Description)
	case configapi.Failure_CONFLICT:
		return errors.NewConflict(failure.Description)
	case configapi.Failure_INVALID:
		return errors.NewInvalid(failure.Description)
	case configapi.Failure_UNAVAILABLE:
		return errors.NewUnavailable(failure.Description)
	case configapi.Failure_NOT_SUPPORTED:
		return errors.NewNotSupported(failure.Description)
	case configapi.Failure_TIMEOUT:
		return errors.NewTimeout(failure.Description)
	case configapi.Failure_INTERNAL:
		return errors.NewInternal(failure.Description)
	default:
		return errors.NewUnknown(failure.Description)
	}
}
