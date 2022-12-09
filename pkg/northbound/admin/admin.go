// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/onosproject/onos-config/pkg/utils/tree"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/onosproject/onos-lib-go/pkg/uri"
	"google.golang.org/grpc"
	"strings"
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

func logContext(ctx context.Context, name string) {
	if ctx != nil {
		if md := metautils.ExtractIncoming(ctx); md != nil && md.Get("name") != "" {
			log.Infof("admin %s called by '%s (%s)'. Groups [%v]. Token %s", name,
				md.Get("name"), md.Get("email"), md.Get("groups"), md.Get("at_hash"))
		}
	}
}

// ListRegisteredModels lists the registered models..
func (s Server) ListRegisteredModels(r *admin.ListModelsRequest, stream admin.ConfigAdminService_ListRegisteredModelsServer) error {
	logContext(stream.Context(), "ListRegisteredModels()")
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
	logContext(ctx, "RollbackTransaction()")
	id := configapi.TransactionID(uri.NewURI(uri.WithScheme("uuid"), uri.WithOpaque(uuid.New().String())).String())
	t := &configapi.Transaction{
		ID: id,
		Details: &configapi.Transaction_Rollback{
			Rollback: &configapi.RollbackTransaction{
				RollbackIndex: req.Index,
			},
		},
		TransactionStrategy: configapi.TransactionStrategy{
			// TODO: Make synchronicity and isolation configurable for rollbacks
			Synchronicity: configapi.TransactionStrategy_SYNCHRONOUS,
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
	}
	return nil, ctx.Err()
}

// LeafSelectionQuery selects values allowable for leaf.
func (s Server) LeafSelectionQuery(ctx context.Context, req *admin.LeafSelectionQueryRequest) (*admin.LeafSelectionQueryResponse, error) {
	log.Debugf("Received LeafSelectionQuery %+v", req)
	logContext(ctx, "LeafSelectionQuery()")
	if req == nil {
		return nil, errors.Status(errors.NewInvalid("request is empty")).Err()
	}

	groups := make([]string, 0)
	if md := metautils.ExtractIncoming(ctx); md != nil && md.Get("name") != "" {
		groups = append(groups, strings.Split(md.Get("groups"), ";")...)
		log.Debugf("gNMI LeafSelectionQuery() called by '%s (%s)'. Groups %v. Token %s",
			md.Get("name"), md.Get("email"), groups, md.Get("at_hash"))
	}

	configType := configapi.TargetType(req.Type)
	configVersion := configapi.TargetVersion(req.Version)

	config, err := s.configurationsStore.Get(ctx,
		configuration.NewID(configapi.TargetID(req.Target), configType, configVersion))
	if err != nil {
		return nil, errors.Status(err).Err()
	}

	if req.ChangeContext != nil &&
		len(req.ChangeContext.GetUpdate())+len(req.ChangeContext.GetReplace())+len(req.ChangeContext.GetDelete()) > 0 {

		log.Warn("Ignoring change context for the moment")
		// TODO if there is something in the req.ChangeContext then
		//   a) convert it to Path-Value format like happens in gNMI Set (Updates, Replace and Delete elements)
		//   b) overlay it on to the Path-Values from 1) like happens in gNMI Set Proposal Controller
	}

	values := make([]*configapi.PathValue, 0, len(config.Values))
	for _, changeValue := range config.Values {
		values = append(values, changeValue)
	}

	jsonTree, err := tree.BuildTree(values, true)
	if err != nil {
		return nil, errors.Status(errors.NewInternal("error converting configuration to JSON %v", err)).Err()
	}

	modelPlugin, ok := s.pluginRegistry.GetPlugin(configType, configVersion)
	if !ok {
		return nil, errors.Status(errors.NewInvalid("error getting plugin for %s %s", configType, configVersion)).Err()
	}

	selection, err := modelPlugin.LeafValueSelection(ctx, req.SelectionPath, jsonTree)
	if err != nil {
		return nil, errors.Status(errors.NewInvalid("error getting leaf selection for '%s'. %v", req.SelectionPath, err)).Err()
	}

	return &admin.LeafSelectionQueryResponse{
		Selection: selection,
	}, nil
}
