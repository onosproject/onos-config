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
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"io"
	log "k8s.io/klog"
	"os"
	"strings"
)

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := Server{}
	RegisterConfigAdminServiceServer(r, server)
}

// Server implements the gRPC service for administrative facilities.
type Server struct {
}

// RegisterModel registers a model plugin already on the onos-configs file system.
func (s Server) RegisterModel(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	name, version, err := manager.GetManager().ModelRegistry.RegisterModelPlugin(req.SoFile)
	if err != nil {
		return nil, err
	}
	return &RegisterResponse{
		Name:    name,
		Version: version,
	}, nil
}

// UploadRegisterModel uploads and registers a new model plugin.
func (s Server) UploadRegisterModel(stream ConfigAdminService_UploadRegisterModelServer) error {
	response := RegisterResponse{Name: "Unknown"}
	soFileName := ""

	const TEMPFILE = "/tmp/uploaded_model_plugin.tmp"
	f, err := os.Create(TEMPFILE)
	if err != nil {
		return errors.Wrapf(err, "failed to create temporary file %s", TEMPFILE)
	}
	defer f.Close()

	// while there are messages coming
	i := 0
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}

			err = errors.Wrapf(err,
				"failed while reading chunks from stream")
			return err
		}
		n2, err := f.Write(chunk.Content)
		if err != nil {
			return errors.Wrapf(err, "failed to write chunk %d to %s", i, TEMPFILE)
		}
		log.Infof("Wrote %d bytes to file %s. Chunk %d", n2, TEMPFILE, i)
		soFileName = chunk.SoFile
		i++
	}
	f.Close()
	soFileName = "/tmp/" + soFileName
	log.Infof("File %s written from %d chunks. Renaming to %s", TEMPFILE, i, soFileName)
	err = os.Rename(TEMPFILE, soFileName)
	if err != nil {
		return errors.Wrapf(err, "Failed to rename temporary file")
	}

	name, version, err := manager.GetManager().ModelRegistry.RegisterModelPlugin(soFileName)
	if err != nil {
		return err
	}
	response.Name = name
	response.Version = version

	err = stream.SendAndClose(&response)
	if err != nil {
		return errors.Wrapf(err, "failed to send the close message %v", response)
	}

	return nil
}

// ListRegisteredModels lists the registered models..
func (s Server) ListRegisteredModels(req *ListModelsRequest, stream ConfigAdminService_ListRegisteredModelsServer) error {
	requestedModel := req.ModelName
	requestedVersion := req.ModelVersion

	for _, model := range manager.GetManager().ModelRegistry.ModelPlugins {
		name, version, md, plugin := model.ModelData()
		if requestedModel != "" && !strings.HasPrefix(name, requestedModel) {
			continue
		}
		if requestedVersion != "" && !strings.HasPrefix(version, requestedVersion) {
			continue
		}

		roPaths := make([]*ReadOnlyPath, 0)
		if req.Verbose {
			roPathsAndValues, ok := manager.GetManager().ModelRegistry.ModelReadOnlyPaths[utils.ToModelName(name, version)]
			if !ok {
				log.Warningf("no list of Read Only Paths found for %s %s\n", name, version)
			} else {
				for path, subpathList := range roPathsAndValues {
					subPathsPb := make([]*ReadOnlySubPath, 0)
					for subPath, subPathType := range subpathList {
						subPathPb := ReadOnlySubPath{
							SubPath:   subPath,
							ValueType: ChangeValueType(subPathType),
						}
						subPathsPb = append(subPathsPb, &subPathPb)
					}
					pathPb := ReadOnlyPath{
						Path:    path,
						SubPath: subPathsPb,
					}
					roPaths = append(roPaths, &pathPb)
				}
			}
		}

		rwPaths := make([]*ReadWritePath, 0)
		if req.Verbose {
			rwPathsAndValues, ok := manager.GetManager().ModelRegistry.ModelReadWritePaths[utils.ToModelName(name, version)]
			if !ok {
				log.Warningf("no list of Read Write Paths found for %s %s\n", name, version)
			} else {
				for path, rwObj := range rwPathsAndValues {
					rwObjProto := ReadWritePath{
						Path:        path,
						ValueType:   ChangeValueType(rwObj.ValueType),
						Description: rwObj.Description,
						Default:     rwObj.Default,
						Units:       rwObj.Units,
						Mandatory:   rwObj.Mandatory,
						Range:       rwObj.Range,
						Length:      rwObj.Length,
					}
					rwPaths = append(rwPaths, &rwObjProto)
				}
			}
		}

		// Build model message
		msg := &ModelInfo{
			Name:          name,
			Version:       version,
			ModelData:     md,
			Module:        plugin,
			ReadOnlyPath:  roPaths,
			ReadWritePath: rwPaths,
		}

		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetNetworkChanges provides a stream of submitted network changes.
func (s Server) GetNetworkChanges(r *NetworkChangesRequest, stream ConfigAdminService_GetNetworkChangesServer) error {
	for _, nc := range manager.GetManager().NetworkStore.Store {

		// Build net change message
		msg := &NetChange{
			Time:    &nc.Created,
			Name:    nc.Name,
			User:    nc.User,
			Changes: make([]*ConfigChange, 0),
		}

		// Build list of config change messages.
		for k, v := range nc.ConfigurationChanges {
			msg.Changes = append(msg.Changes, &ConfigChange{Id: string(k), Hash: store.B64(v)})
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
	ctx context.Context, req *RollbackRequest) (*RollbackResponse, error) {
	var networkConfig *store.NetworkConfiguration
	var ncIdx int

	if req.Name == "" {
		networkConfig =
			&manager.GetManager().NetworkStore.Store[len(manager.GetManager().NetworkStore.Store)-1]
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

	configNames := make(map[string][]string)
	// Check all are valid before we delete anything
	for configName, changeID := range networkConfig.ConfigurationChanges {
		configChangeIds := manager.GetManager().ConfigStore.Store[configName].Changes
		if store.B64(configChangeIds[len(configChangeIds)-1]) != store.B64(changeID) {
			return nil, fmt.Errorf(
				"the last change on %s is not %s as expected. Was %s",
				configName, store.B64(changeID), store.B64(configChangeIds[len(configChangeIds)-1]))
		}
		changeID, err := manager.GetManager().RollbackTargetConfig(configName)
		rollbackIDs := configNames[string(configName)]
		configNames[string(configName)] = append(rollbackIDs, store.B64(changeID))
		if err != nil {
			return nil, err
		}
	}
	_ = manager.GetManager().NetworkStore.RemoveEntry(networkConfig.Name)

	return &RollbackResponse{
		Message: fmt.Sprintf("Rolled back change '%s' Updated configs %s",
			networkConfig.Name, configNames),
	}, nil
}
