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
	"io"
	"os"
	"strings"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-api/go/onos/config/snapshot"
	devicesnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/device"
	networksnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/network"
	"github.com/onosproject/onos-config/pkg/manager"
	streams "github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound", "admin")

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := Server{}
	admin.RegisterConfigAdminServiceServer(r, server)
}

// Server implements the gRPC service for administrative facilities.
type Server struct {
}

// UploadRegisterModel uploads and registers a new model plugin.
func (s Server) UploadRegisterModel(stream admin.ConfigAdminService_UploadRegisterModelServer) error {
	response := admin.RegisterResponse{Name: "Unknown"}
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
		_, err = f.Write(chunk.Content)
		if err != nil {
			return errors.Wrapf(err, "failed to write chunk %d to %s", i, TEMPFILE)
		}
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
func (s Server) ListRegisteredModels(req *admin.ListModelsRequest, stream admin.ConfigAdminService_ListRegisteredModelsServer) error {
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

		roPaths := make([]*admin.ReadOnlyPath, 0)
		if req.Verbose {
			roPathsAndValues, ok := manager.GetManager().ModelRegistry.ModelReadOnlyPaths[utils.ToModelName(devicetype.Type(name), devicetype.Version(version))]
			if !ok {
				log.Warnf("no list of Read Only Paths found for %s %s\n", name, version)
			} else {
				for path, subpathList := range roPathsAndValues {
					subPathsPb := make([]*admin.ReadOnlySubPath, 0)
					for subPath, subPathType := range subpathList {
						subPathPb := admin.ReadOnlySubPath{
							SubPath:   subPath,
							ValueType: subPathType.Datatype,
						}
						subPathsPb = append(subPathsPb, &subPathPb)
					}
					pathPb := admin.ReadOnlyPath{
						Path:    path,
						SubPath: subPathsPb,
					}
					roPaths = append(roPaths, &pathPb)
				}
			}
		}

		rwPaths := make([]*admin.ReadWritePath, 0)
		if req.Verbose {
			rwPathsAndValues, ok := manager.GetManager().ModelRegistry.ModelReadWritePaths[utils.ToModelName(devicetype.Type(name), devicetype.Version(version))]
			if !ok {
				log.Warnf("no list of Read Write Paths found for %s %s\n", name, version)
			} else {
				for path, rwObj := range rwPathsAndValues {
					rwObjProto := admin.ReadWritePath{
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
		msg := &admin.ModelInfo{
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

// RollbackNetworkChange rolls back a named atomix-based network change.
func (s Server) RollbackNetworkChange(ctx context.Context, req *admin.RollbackRequest) (*admin.RollbackResponse, error) {
	errRollback := manager.GetManager().RollbackTargetConfig(networkchange.ID(req.Name))
	if errRollback != nil {
		return nil, errRollback
	}
	return &admin.RollbackResponse{
		Message: fmt.Sprintf("Rolled back change '%s'", req.Name),
	}, nil
}

// ListSnapshots lists snapshots for all devices
func (s Server) ListSnapshots(r *admin.ListSnapshotsRequest, stream admin.ConfigAdminService_ListSnapshotsServer) error {
	log.Infof("ListSnapshots called with %s. Subscribe %v", r.ID, r.Subscribe)

	// There may be a wildcard given - we only want to reply with changes that match
	matcher := utils.MatchWildcardChNameRegexp(string(r.ID))

	if r.Subscribe {
		eventCh := make(chan streams.Event)
		ctx, err := manager.GetManager().DeviceSnapshotStore.WatchAll(eventCh)
		if err != nil {
			log.Errorf("Error watching Network Changes %s", err)
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
						log.Errorf("Error sending Snapshot %v %v", change.ID, err)
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
		ctx, err := manager.GetManager().DeviceSnapshotStore.LoadAll(changeCh)
		if err != nil {
			log.Errorf("Error ListSnapshots %s", err)
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
						log.Errorf("Error sending Snapshot %v %v", change.ID, err)
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
	snap := &networksnapshot.NetworkSnapshot{
		Retention: snapshot.RetentionOptions{
			RetainWindow: request.RetentionPeriod,
		},
	}

	ch := make(chan streams.Event)
	stream, err := manager.GetManager().NetworkSnapshotStore.Watch(ch)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	if err := manager.GetManager().NetworkSnapshotStore.Create(snap); err != nil {
		return nil, err
	}

	for event := range ch {
		eventSnapshot := event.Object.(*networksnapshot.NetworkSnapshot)
		if snap.ID != "" && snap.ID == eventSnapshot.ID && eventSnapshot.Status.Phase == snapshot.Phase_DELETE && eventSnapshot.Status.State == snapshot.State_COMPLETE {
			return &admin.CompactChangesResponse{}, nil
		}
	}
	return nil, errors.New("snapshot state unknown")
}
