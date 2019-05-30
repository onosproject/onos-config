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

package main

import (
	"encoding/json"
	"time"

	log "github.com/golang/glog"
	"github.com/onosproject/onos-config/tools/test/devicesim/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newServer creates a new gNMI server.
func newServer(model *gnmi.Model, config []byte) (*server, error) {
	s, err := gnmi.NewServer(model, config, nil)

	if err != nil {
		return nil, err
	}

	newconfig, _ := model.NewConfigStruct(config)
	server := server{Server: s, Model: model, configStruct: newconfig}
	return &server, nil
}

// sendResponse sends an SubscribeResponse to a gNMI client.
func (s *server) sendResponse(response *pb.SubscribeResponse, stream pb.GNMI_SubscribeServer) {
	log.Info("Sending SubscribeResponse out to gNMI client: ", response)
	err := stream.Send(response)
	if err != nil {
		//TODO remove channel registrations
		log.Errorf("Error in sending response to client")
	}
}

// buildSubResponse builds a subscribeResponse based on the given Update message.
func buildSubResponse(update *pb.Update) (*pb.SubscribeResponse, error) {
	updateArray := make([]*pb.Update, 0)
	updateArray = append(updateArray, update)
	notification := &pb.Notification{
		Timestamp: time.Now().Unix(),
		Update:    updateArray,
	}
	responseUpdate := &pb.SubscribeResponse_Update{
		Update: notification,
	}
	response := &pb.SubscribeResponse{
		Response: responseUpdate,
	}

	return response, nil
}

// getUpdate finds the node in the tree, build the update message and return it back to the collector
func (s *server) getUpdate(subList *pb.SubscriptionList, path *pb.Path) (*pb.Update, error) {
	fullPath := path
	prefix := subList.GetPrefix()

	if prefix != nil {
		fullPath = gnmiFullPath(prefix, path)
	}
	if fullPath.GetElem() == nil && fullPath.GetElement() != nil {
		return nil, status.Error(codes.Unimplemented, "deprecated path element type is unsupported")
	}

	jsonConfigString, _ := ygot.EmitJSON(s.configStruct, nil)
	configMap := make(map[string]interface{})

	json.Unmarshal([]byte(jsonConfigString), &configMap)
	pathElements := path.GetElem()

	leafValue, _ := findLeaf(configMap, pathElements[len(pathElements)-1].GetName())
	val := &pb.TypedValue{
		Value: &pb.TypedValue_StringVal{
			StringVal: leafValue,
		},
	}
	update := pb.Update{Path: path, Val: val}
	return &update, nil

}

// collector collects the latest update from the config.
func (s *server) collector(ch chan *pb.Update, request *pb.SubscriptionList) {
	for _, sub := range request.Subscription {
		path := sub.GetPath()
		update, err := s.getUpdate(request, path)
		if err != nil {
			log.Info("Error while collecting data for subscribe once or poll", err)

		}
		if err == nil {
			ch <- update
		}

	}
}

// listenForUpdates reads update messages from the update channel, creates a
// subscribe response and send it to the gnmi client.
func (s *server) listenForUpdates(updateChan chan *pb.Update, stream pb.GNMI_SubscribeServer,
	mode pb.SubscriptionList_Mode, done chan struct{}) {
	for update := range updateChan {
		response, _ := buildSubResponse(update)
		s.sendResponse(response, stream)

		if mode != pb.SubscriptionList_ONCE {
			responseSync := &pb.SubscribeResponse_SyncResponse{
				SyncResponse: true,
			}
			response = &pb.SubscribeResponse{
				Response: responseSync,
			}
			s.sendResponse(response, stream)

		} else {
			//If the subscription mode is ONCE we read from the channel, build a response and issue it
			done <- struct{}{}
		}
	}
}
