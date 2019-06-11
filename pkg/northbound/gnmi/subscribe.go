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
	"crypto/sha1"
	"fmt"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"io"
	"log"
	"time"
)

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	//updateChan := make(chan *gnmi.Update)
	var subscribe *gnmi.SubscriptionList
	mgr := manager.GetManager()
	//hashing the stream to obtain a unique identifier of the client
	h := sha1.New()
	_, err1 := io.WriteString(h, fmt.Sprintf("%v", stream))
	if err1 != nil {
		return err1
	}
	hash := string(h.Sum(nil))
	//Registering one listener per NB app/client on both change and opStateChan
	changesChan, err := mgr.Dispatcher.RegisterNbi(hash)
	if err != nil {
		log.Println("Subscription present: ", err)
		return err
	}
	opStateChan, err := mgr.Dispatcher.RegisterOpState(hash)
	if err != nil {
		log.Println("Subscription present: ", err)
		return err
	}
	//this for loop handles each subscribe request coming into the server
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Println("Subscription Terminated")
			mgr.Dispatcher.UnregisterNbi(hash)
			mgr.Dispatcher.UnregisterOperationalState(hash)
			return nil
		}

		if err != nil {
			log.Println("Error in subscription", err)
			//Ignoring Errors during removal
			mgr.Dispatcher.UnregisterNbi(hash)
			mgr.Dispatcher.UnregisterOperationalState(hash)
			return err
		}

		var mode gnmi.SubscriptionList_Mode

		if in.GetPoll() != nil {
			mode = gnmi.SubscriptionList_POLL
		} else {
			subscribe = in.GetSubscribe()
			mode = subscribe.Mode
		}
		//If the subscription mode is ONCE or POLL we immediately start a routine to collect the data
		if mode != gnmi.SubscriptionList_STREAM {
			go collector(stream, subscribe)
		} else {
			subs := subscribe.Subscription
			//FAST way to identify if target and subscription is present
			subsStr := make(map[string]struct{})
			targets := make(map[string]struct{})
			for _, sub := range subs {
				subscriptionPathStr := utils.StrPath(sub.Path)
				subsStr[subscriptionPathStr] = struct{}{}
				targets[sub.Path.Target] = struct{}{}
			}
			//Each subscription request spawns a go routing listening for related events for the target and the paths
			go listenForUpdates(changesChan, stream, mgr, targets, subsStr)
			go listenForOpStateUpdates(opStateChan, stream, targets, subsStr)
		}
	}

}

func collector(stream gnmi.GNMI_SubscribeServer, request *gnmi.SubscriptionList) {
	for _, sub := range request.Subscription {
		//We get the stated of the device, for each path we build an update and send it out.
		update, err := getUpdate(request.Prefix, sub.Path)
		if err != nil {
			//TODO propagate error
			log.Println("Error while collecting data for subscribe once or poll", err)
		}
		response := buildUpdateResponse(update)
		sendResponse(response, stream)
	}
	responseSync := buildSyncResponse()
	sendResponse(responseSync, stream)
}

//For each update coming from the change channel we check if it's for a valid target and path then, if so, we send it NB
func listenForUpdates(changeChan chan events.ConfigEvent, stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager,
	targets map[string]struct{}, subs map[string]struct{}) {
	for update := range changeChan {
		target, changeInternal := getChangeFromEvent(update, mgr)
		_, targetPresent := targets[target]
		if targetPresent {
			for _, changeValue := range changeInternal.Config {
				_, isPresent := subs[changeValue.Path]
				if isPresent {
					pathGnmi, err := utils.ParseGNMIElements(utils.SplitPath(changeValue.Path))
					if err != nil {
						log.Println("Error in parsing path", err)
						continue
					}
					buildAndSendUpdate(pathGnmi, target, changeValue.Value, stream)
				}
			}
		}
	}
}

//For each update coming from the state channel we check if it's for a valid target and path then, if so, we send it NB
func listenForOpStateUpdates(opStateChan chan events.OperationalStateEvent, stream gnmi.GNMI_SubscribeServer,
	targets map[string]struct{}, subs map[string]struct{}) {
	for opStateChange := range opStateChan {
		target := events.Event(opStateChange).Subject()
		_, targetPresent := targets[target]
		if targetPresent {
			changeInternal := events.Event(opStateChange).Values()
			for pathStr, value := range *changeInternal {
				_, isPresent := subs[pathStr]
				if isPresent {
					pathArr := utils.SplitPath(pathStr)
					pathGnmi, err := utils.ParseGNMIElements(pathArr)
					if err != nil {
						log.Println("Error in parsing path", err)
						continue
					}
					buildAndSendUpdate(pathGnmi, target, value, stream)
				}
			}
		}
	}
}

func buildAndSendUpdate(pathGnmi *gnmi.Path, target string, value string, stream gnmi.GNMI_SubscribeServer) {
	pathGnmi.Target = target
	typedValue := gnmi.TypedValue_StringVal{StringVal: value}
	valueGnmi := &gnmi.TypedValue{Value: &typedValue}
	update := &gnmi.Update{
		Path: pathGnmi,
		Val:  valueGnmi,
	}
	response := buildUpdateResponse(update)
	sendResponse(response, stream)
	responseSync := buildSyncResponse()
	sendResponse(responseSync, stream)
}

func buildSyncResponse() *gnmi.SubscribeResponse {
	responseSync := &gnmi.SubscribeResponse_SyncResponse{
		SyncResponse: true,
	}
	return &gnmi.SubscribeResponse{
		Response: responseSync,
	}
}

func buildUpdateResponse(update *gnmi.Update) *gnmi.SubscribeResponse {
	updateArray := make([]*gnmi.Update, 0)
	updateArray = append(updateArray, update)
	notification := &gnmi.Notification{
		Timestamp: time.Now().Unix(),
		Update:    updateArray,
	}
	responseUpdate := &gnmi.SubscribeResponse_Update{
		Update: notification,
	}
	response := &gnmi.SubscribeResponse{
		Response: responseUpdate,
	}
	return response
}

func sendResponse(response *gnmi.SubscribeResponse, stream gnmi.GNMI_SubscribeServer) {
	log.Println("Sending SubscribeResponse out to gNMI client", response)
	err := stream.Send(response)
	if err != nil {
		//TODO propagate error
		log.Println("Error in sending response to client", err)
	}
}

func getChangeFromEvent(update events.ConfigEvent, mgr *manager.Manager) (string, *change.Change) {
	target := events.Event(update).Subject()
	changeID := update.ChangeID()
	changeInternal := mgr.ChangeStore.Store[changeID]
	return target, changeInternal
}
