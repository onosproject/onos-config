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
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	"github.com/onosproject/onos-api/go/onos/config/snapshot"
	devicesnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/device"
	networksnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/network"
	nbutils "github.com/onosproject/onos-config/pkg/northbound/utils"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	devicesnap "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnap "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	streams "github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound", "admin")

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
	networkChangesStore  network.Store
	networkSnapshotStore networksnap.Store
	deviceSnapshotStore  devicesnap.Store
	pluginRegistry       *pluginregistry.PluginRegistry
}

// NewService allocates a Service struct with the given parameters
func NewService(networkChangesStore network.Store,
	networkSnapshotStore networksnap.Store,
	deviceSnapshotStore devicesnap.Store,
	pluginRegistry *pluginregistry.PluginRegistry) Service {
	return Service{
		networkChangesStore:  networkChangesStore,
		networkSnapshotStore: networkSnapshotStore,
		deviceSnapshotStore:  deviceSnapshotStore,
		pluginRegistry:       pluginRegistry,
	}
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := Server{
		networkChangesStore:  s.networkChangesStore,
		networkSnapshotStore: s.networkSnapshotStore,
		deviceSnapshotStore:  s.deviceSnapshotStore,
		pluginRegistry:       s.pluginRegistry}
	admin.RegisterConfigAdminServiceServer(r, server)
}

// Server implements the gRPC service for administrative facilities.
type Server struct {
	networkChangesStore  network.Store
	networkSnapshotStore networksnap.Store
	deviceSnapshotStore  devicesnap.Store
	pluginRegistry       *pluginregistry.PluginRegistry
}

// UploadRegisterModel uploads and registers a new model plugin.
// Deprecated: models should only be loaded at startup
func (s Server) UploadRegisterModel(stream admin.ConfigAdminService_UploadRegisterModelServer) error {
	return errors.NewNotSupported("not implemented")
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
			Id:     p.ID,
			Port:   uint32(p.Port),
			Info:   &p.Info,
			Status: p.Status.String(),
			Error:  p.Error,
		}
		err := stream.Send(msg)
		if err != nil {
			log.Errorf("error sending ModelInfor from plugin %v: %v", p.ID, err)
			return err
		}
	}
	return nil
}

// RollbackNetworkChange rolls back a named atomix-based network change.
func (s Server) RollbackNetworkChange(ctx context.Context, req *admin.RollbackRequest) (*admin.RollbackResponse, error) {
	if md := metautils.ExtractIncoming(ctx); md != nil && md.Get("name") != "" {
		log.Infof("admin RollbackNetworkChange() called by '%s (%s)'. Groups [%v]. Token %s",
			md.Get("name"), md.Get("email"), md.Get("groups"), md.Get("at_hash"))
		// TODO replace the following with fine grained RBAC using OpenPolicyAgent Regos
		if err := utils.TemporaryEvaluate(md); err != nil {
			return nil, err
		}
	}
	errRollback := nbutils.RollbackTargetConfig(networkchange.ID(req.Name), s.networkChangesStore)
	if errRollback != nil {
		return nil, errRollback
	}
	return &admin.RollbackResponse{
		Message: fmt.Sprintf("Rolled back change '%s'", req.Name),
	}, nil
}

// ListSnapshots lists snapshots for all devices
func (s Server) ListSnapshots(r *admin.ListSnapshotsRequest, stream admin.ConfigAdminService_ListSnapshotsServer) error {
	if stream.Context() != nil {
		if md := metautils.ExtractIncoming(stream.Context()); md != nil && md.Get("name") != "" {
			log.Infof("admin ListSnapshots() called by '%s (%s)'. Groups [%v]. Token %s",
				md.Get("name"), md.Get("email"), md.Get("groups"), md.Get("at_hash"))
		}
	}
	log.Infof("ListSnapshots called with %s. Subscribe %v", r.ID, r.Subscribe)

	// There may be a wildcard given - we only want to reply with changes that match
	matcher := utils.MatchWildcardChNameRegexp(string(r.ID), false)

	if r.Subscribe {
		eventCh := make(chan streams.Event)
		ctx, err := s.deviceSnapshotStore.WatchAll(eventCh)
		if err != nil {
			log.Errorf("error watching Network Changes %s", err)
			return err
		}
		defer ctx.Close()

		for {
			breakout := false
			select { // Blocks until one of the following are received
			case event, ok := <-eventCh:
				if !ok { // Will happen at the end of stream
					breakout = true
					break
				}

				change := event.Object.(*devicesnapshot.Snapshot)

				if matcher.MatchString(string(change.ID)) {
					msg := change
					log.Infof("Sending matching change %v", change.ID)
					err := stream.Send(msg)
					if err != nil {
						log.Errorf("error sending Snapshot %v %v", change.ID, err)
						return err
					}
				}
			case <-stream.Context().Done():
				log.Infof("ListSnapshots remote client closed connection")
				return nil
			}
			if breakout {
				break
			}
		}
	} else {
		changeCh := make(chan *devicesnapshot.Snapshot)
		ctx, err := s.deviceSnapshotStore.LoadAll(changeCh)
		if err != nil {
			log.Errorf("error ListSnapshots %s", err)
			return err
		}
		defer ctx.Close()

		for {
			breakout := false
			select { // Blocks until one of the following are received
			case change, ok := <-changeCh:
				if !ok { // Will happen at the end of stream
					breakout = true
					break
				}

				if matcher.MatchString(string(change.ID)) {
					msg := change
					log.Infof("Sending matching change %v", change.ID)
					err := stream.Send(msg)
					if err != nil {
						log.Errorf("error sending Snapshot %v %v", change.ID, err)
						return err
					}
				}
			case <-stream.Context().Done():
				log.Infof("ListSnapshots remote client closed connection")
				return nil
			}
			if breakout {
				break
			}
		}
	}
	log.Infof("Closing ListSnapshots for %s", r.ID)

	return nil
}

// CompactChanges takes a snapshot of all devices
func (s Server) CompactChanges(ctx context.Context, request *admin.CompactChangesRequest) (*admin.CompactChangesResponse, error) {
	if md := metautils.ExtractIncoming(ctx); md != nil && md.Get("name") != "" {
		log.Infof("admin CompactChanges() called by '%s (%s)'. Groups [%v]. Token %s",
			md.Get("name"), md.Get("email"), md.Get("groups"), md.Get("at_hash"))
		// TODO replace the following with fine grained RBAC using OpenPolicyAgent Regos
		if err := utils.TemporaryEvaluate(md); err != nil {
			return nil, err
		}
	}
	snap := &networksnapshot.NetworkSnapshot{
		Retention: snapshot.RetentionOptions{
			RetainWindow: request.RetentionPeriod,
		},
	}

	ch := make(chan streams.Event)
	stream, err := s.networkSnapshotStore.Watch(ch)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	if err := s.networkSnapshotStore.Create(snap); err != nil {
		return nil, err
	}

	for event := range ch {
		eventSnapshot := event.Object.(*networksnapshot.NetworkSnapshot)
		if snap.ID != "" && snap.ID == eventSnapshot.ID && eventSnapshot.Status.Phase == snapshot.Phase_DELETE && eventSnapshot.Status.State == snapshot.State_COMPLETE {
			return &admin.CompactChangesResponse{}, nil
		}
	}
	return nil, errors.NewInvalid("snapshot state unknown")
}
