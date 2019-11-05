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
	"github.com/onosproject/onos-config/pkg/store/stream"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	devicetype "github.com/onosproject/onos-config/pkg/types/device"
	snapshottype "github.com/onosproject/onos-config/pkg/types/snapshot"
	"github.com/onosproject/onos-config/pkg/types/snapshot/device"
	networksnaptypes "github.com/onosproject/onos-config/pkg/types/snapshot/network"
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

// ConfigAdminClientFactory : Default ConfigAdminClient creation.
var ConfigAdminClientFactory = func(cc *grpc.ClientConn) ConfigAdminServiceClient {
	return NewConfigAdminServiceClient(cc)
}

// CreateConfigAdminServiceClient creates and returns a new config admin client
func CreateConfigAdminServiceClient(cc *grpc.ClientConn) ConfigAdminServiceClient {
	return ConfigAdminClientFactory(cc)
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
			roPathsAndValues, ok := manager.GetManager().ModelRegistry.ModelReadOnlyPaths[utils.ToModelName(devicetype.Type(name), devicetype.Version(version))]
			if !ok {
				log.Warningf("no list of Read Only Paths found for %s %s\n", name, version)
			} else {
				for path, subpathList := range roPathsAndValues {
					subPathsPb := make([]*ReadOnlySubPath, 0)
					for subPath, subPathType := range subpathList {
						subPathPb := ReadOnlySubPath{
							SubPath:   subPath,
							ValueType: subPathType.Datatype,
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
			rwPathsAndValues, ok := manager.GetManager().ModelRegistry.ModelReadWritePaths[utils.ToModelName(devicetype.Type(name), devicetype.Version(version))]
			if !ok {
				log.Warningf("no list of Read Write Paths found for %s %s\n", name, version)
			} else {
				for path, rwObj := range rwPathsAndValues {
					rwObjProto := ReadWritePath{
						Path:        path,
						ValueType:   rwObj.ValueType,
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
			GetStateMode:  uint32(model.GetStateMode()),
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

<<<<<<< Updated upstream
// RollbackNetworkChange rolls back a named atomix-based network change.
func (s Server) RollbackNetworkChange(
=======
// RollbackNewNetworkChange rolls back a named new (atomix-based)netwoget.gok changes.
func (s Server) RollbackNewNetworkChange(
>>>>>>> Stashed changes
	ctx context.Context, req *RollbackRequest) (*RollbackResponse, error) {
	errRollback := manager.GetManager().RollbackTargetConfig(networkchangetypes.ID(req.Name))
	if errRollback != nil {
		return nil, errRollback
	}
	return &RollbackResponse{
		Message: fmt.Sprintf("Rolled back change '%s'", req.Name),
	}, nil
}

// GetSnapshot gets a snapshot for a specific device
func (s Server) GetSnapshot(ctx context.Context, request *GetSnapshotRequest) (*device.Snapshot, error) {
	return manager.GetManager().DeviceSnapshotStore.Load(devicetype.NewVersionedID(request.DeviceID, request.DeviceVersion))
}

// ListSnapshots lists snapshots for all devices
func (s Server) ListSnapshots(request *ListSnapshotsRequest, stream ConfigAdminService_ListSnapshotsServer) error {
	ch := make(chan *device.Snapshot)
	ctx, err := manager.GetManager().DeviceSnapshotStore.LoadAll(ch)
	if err != nil {
		return err
	}
	defer ctx.Close()

	for snapshot := range ch {
		if err := stream.SendMsg(snapshot); err != nil {
			return err
		}
	}
	return nil
}

// CompactChanges takes a snapshot of all devices
func (s Server) CompactChanges(ctx context.Context, request *CompactChangesRequest) (*CompactChangesResponse, error) {
	snapshot := &networksnaptypes.NetworkSnapshot{
		Retention: snapshottype.RetentionOptions{
			RetainWindow: request.RetentionPeriod,
		},
	}

	ch := make(chan stream.Event)
	stream, err := manager.GetManager().NetworkSnapshotStore.Watch(ch)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	if err := manager.GetManager().NetworkSnapshotStore.Create(snapshot); err != nil {
		return nil, err
	}

	for event := range ch {
		eventSnapshot := event.Object.(*networksnaptypes.NetworkSnapshot)
		if snapshot.ID != "" && snapshot.ID == eventSnapshot.ID && eventSnapshot.Status.Phase == snapshottype.Phase_DELETE && eventSnapshot.Status.State == snapshottype.State_COMPLETE {
			return &CompactChangesResponse{}, nil
		}
	}
	return nil, errors.New("snapshot state unknown")
}
