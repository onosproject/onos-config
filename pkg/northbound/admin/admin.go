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

package admin

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/onosproject/onos-config/pkg/store"
	"google.golang.org/grpc"
	"strings"
)

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	proto.RegisterAdminServiceServer(r, Server{})
}

// Server implements the gRPC service for administrative facilities.
type Server struct {
}

// RegisterModel registers a new YANG model.
func (s Server) RegisterModel(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	return &proto.RegisterResponse{}, nil
}

// UnregisterModel unregisters the specified YANG model.
func (s Server) UnregisterModel(ctx context.Context, req *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	return &proto.RegisterResponse{}, nil
}

// GetNetworkChanges provides a stream of submitted network changes.
func (s Server) GetNetworkChanges(r *proto.NetworkChangesRequest, stream proto.AdminService_GetNetworkChangesServer) error {
	for _, nc := range manager.GetManager().NetworkStore.Store {

		// Build net change message
		msg := &proto.NetChange{
			Time:    &timestamp.Timestamp{Seconds: nc.Created.Unix(), Nanos: int32(nc.Created.Nanosecond())},
			Name:    nc.Name,
			User:    nc.User,
			Changes: make([]*proto.ConfigChange, 0),
		}

		// Build list of config change messages.
		for k, v := range nc.ConfigurationChanges {
			msg.Changes = append(msg.Changes, &proto.ConfigChange{Id: string(k), Hash: store.B64(v)})
		}

		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// RollbackNetworkChange rolls back a named network changes.
func (s Server) RollbackNetworkChange(
	ctx context.Context, req *proto.RollbackRequest) (*proto.RollbackResponse, error) {
	var networkConfig *store.NetworkConfiguration
	var ncIdx int

	if req.Name == "" {
		networkConfig =
			&manager.GetManager().NetworkStore.Store[len(manager.GetManager().NetworkStore.Store) - 1]
	} else {
		for idx, nc := range manager.GetManager().NetworkStore.Store {
			if nc.Name == req.Name {
				networkConfig = &nc
				ncIdx = idx
			}
		}
		if networkConfig == nil {
			return nil, fmt.Errorf("Rollback aborted. Network change %s not found", req.Name)
		}

		// Look to see if any of the devices have been updated in later NCs
		if len(manager.GetManager().NetworkStore.Store) > ncIdx+1 {
			for _, nc := range manager.GetManager().NetworkStore.Store[ncIdx+1:] {
				for k := range nc.ConfigurationChanges {
					if len(networkConfig.ConfigurationChanges[k]) > 0 {
						return nil, fmt.Errorf(
							"network change %s cannot be rolled back because change %s "+
								"subsequently modifies %s", req.Name, nc.Name, k)
					}
				}
			}
		}
	}

	configNames := make([]string, 0)
	// Check all are valid before we delete anything
	for k, v := range networkConfig.ConfigurationChanges {
		configChangeIds := manager.GetManager().ConfigStore.Store[k].Changes
		if store.B64(configChangeIds[len(configChangeIds)-1]) != store.B64(v) {
			return nil, fmt.Errorf(
				"the last change on %s is not %s as expected. Was %s",
				k, v, store.B64(configChangeIds[len(configChangeIds)-1]))
		}
		configNames = append(configNames, string(k))
	}

	for k := range networkConfig.ConfigurationChanges {
		manager.GetManager().ConfigStore.RemoveLastChangeEntry(k)
	}
	manager.GetManager().NetworkStore.RemoveEntry(networkConfig.Name)

	return &proto.RollbackResponse{
		Message: fmt.Sprintf("Rolled back change '%s' Updated configs %s",
			networkConfig.Name, strings.Join(configNames, ",")),
	}, nil
}
