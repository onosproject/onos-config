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

package gnmi

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/models"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"time"
)

// Service implements Service for GNMI
type Service struct {
	northbound.Service
}

// Register registers the GNMI server with grpc
func (s Service) Register(r *grpc.Server) {
	gnmi.RegisterGNMIServer(r, &Server{
		models: models.NewModels(),
	})
}

// Server implements the grpc GNMI service
type Server struct {
	models *models.Models
}

// Capabilities implements gNMI Capabilities
func (s *Server) Capabilities(ctx context.Context, req *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	v, _ := getGNMIServiceVersion()
	return &gnmi.CapabilityResponse{
		SupportedModels:     s.models.Get(),
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_JSON},
		GNMIVersion:           *v,
	}, nil
}

// Get implements gNMI Get
func (s *Server) Get(ctx context.Context, req *gnmi.GetRequest) (*gnmi.GetResponse, error) {

	updates := make([]*gnmi.Update,0)

	for _, path := range req.Path {
		target := path.Target
		pathAsString := utils.StrPath(path)
		configValues, err := manager.GetManager().GetNetworkConfig(target, "",
			pathAsString, 0)
		if err != nil {
			return nil, err
		}

		var value *gnmi.TypedValue
		if len(configValues) == 0 {
			value = nil
		} else if len(configValues) == 1 {
			typedValue := &gnmi.TypedValue_AsciiVal{AsciiVal:configValues[0].Value}
			value = &gnmi.TypedValue{
				Value: typedValue,
			}
		} else {
			json, err := manager.BuildTree(configValues)
			if err != nil {

			}
			typedValue := &gnmi.TypedValue_JsonVal{
				JsonVal: json,
			}
			value = &gnmi.TypedValue{
				Value: typedValue,
			}
		}

		update := &gnmi.Update{
			Path: path,
			Val:  value,
		}
		updates = append(updates, update)
	}

	notification := &gnmi.Notification{
		Timestamp: time.Now().Unix(),
		Update: updates,

	}
	notifications := make([]*gnmi.Notification,0)
	notifications = append(notifications, notification)
	response := gnmi.GetResponse{
		Notification: notifications,
	}
	return &response, nil
}

// Set implements gNMI Set
func (s *Server) Set(ctx context.Context, req *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	//updates := make(map[string]string)
	targetUpdates := make(map[string]map[string]string)
	targetRemoves := make(map[string][]string)

	//TODO consolidate these lines into methods
	//Update
	for _, u := range req.Update {
		target := u.Path.Target
		updates, ok := targetUpdates[target]
		if !ok {
			updates = make(map[string]string)
		}
		path := utils.StrPath(u.Path)
		updates[path] = utils.StrVal(u.Val)
		targetUpdates[target] = updates
	}

	//Replace
	for _, u := range req.Replace {
		target := u.Path.Target
		updates, ok := targetUpdates[target]
		if !ok {
			updates = make(map[string]string)
		}
		path := utils.StrPath(u.Path)
		updates[path] = utils.StrVal(u.Val)
		targetUpdates[target] = updates
	}

	//Delete
	for _, u := range req.Delete {
		target := u.Target
		deletes, ok := targetRemoves[target]
		if !ok {
			deletes = make([]string, 0)
		}
		path := utils.StrPath(u)
		deletes = append(deletes, path)
		targetRemoves[target] = deletes
	}

	for target, updates := range targetUpdates{
		changeID, err := manager.GetManager().SetNetworkConfig(target, "test", updates, targetRemoves[target])
		if err != nil {
			log.Println("Error in setting config:", changeID, "for target", target)
			//TODO save error and stop proccess and initiate rollback
		}
	}
	//TODO consolidate across devices and create a network change
	//TODO Create the SetResponse
	return &gnmi.SetResponse{}, nil
}

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	return nil
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
