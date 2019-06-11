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
	"encoding/json"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"io"
	"log"
	"time"
)

//per each subscribe request we receive the map is updated with a channel corresponding to the path.
//var (
//	PathToChannels = make(map[*gnmi.Path]chan events.ConfigEvent)
//)

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	//updateChan := make(chan *gnmi.Update)
	var subscribe *gnmi.SubscriptionList
	mgr := manager.GetManager()
	//TODO remove assumption of 1 target
	target := ""
	h := sha1.New()
	jsonstr, _ := json.Marshal(stream)
	_, err1 := io.WriteString(h, string(jsonstr))
	if err1 != nil {
		return err1
	}
	hash := string(h.Sum(nil))
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
		log.Println("Receiving", in)
		if err == io.EOF {
			log.Println("Subscription Terminated")
			mgr.Dispatcher.UnregisterNbi(target)
			mgr.Dispatcher.UnregisterOperationalState(target)
			return nil
		}

		if err != nil {
			log.Println("Error in subscription", err)
			if target != "" {
				//Ignoring Errors during removal
				mgr.Dispatcher.UnregisterNbi(target)
				mgr.Dispatcher.UnregisterOperationalState(target)
			}
			return err
		}

		var mode gnmi.SubscriptionList_Mode

		if in.GetPoll() != nil {
			mode = gnmi.SubscriptionList_POLL
		} else {
			subscribe = in.GetSubscribe()
			log.Println("Subscribe", subscribe)
			target = subscribe.Subscription[0].Path.Target
			mode = subscribe.Mode
		}
		//If the subscription mode is ONCE or POLL we immediately start a routine to collect the data
		if mode != gnmi.SubscriptionList_STREAM {
			go collector(stream, subscribe, mode)
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
			go listenForUpdates(changesChan, stream, mgr, targets, subsStr)
			go listenForOpStateUpdates(opStateChan, stream, targets, subsStr)
		}
	}

}

func collector(stream gnmi.GNMI_SubscribeServer, request *gnmi.SubscriptionList, mode gnmi.SubscriptionList_Mode) {
	for _, sub := range request.Subscription {
		log.Println("Sub", sub)
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

func listenForUpdates(changeChan chan events.ConfigEvent, stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager,
	targets map[string]struct{}, subs map[string]struct{}) {
	for update := range changeChan {
		target, changeInternal := getChangeFromEvent(update, mgr)
		_, targetPresent := targets[target]
		if targetPresent {
			for _, changeValue := range changeInternal.Config {
				log.Println("subPaths", subs)
				_, isPresent := subs[changeValue.Path]
				log.Println("pathStr from Event ", changeValue.Path)
				log.Println("isPresent", true)
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

func listenForOpStateUpdates(opStateChan chan events.OperationalStateEvent, stream gnmi.GNMI_SubscribeServer,
	targets map[string]struct{}, subs map[string]struct{}) {
	for opStateChange := range opStateChan {
		target := events.Event(opStateChange).Subject()
		_, targetPresent := targets[target]
		if targetPresent {
			changeInternal := events.Event(opStateChange).Values()
			for pathStr, value := range *changeInternal {
				log.Println("subPaths", subs)
				_, isPresent := subs[pathStr]
				log.Println("apthStr form Event ", pathStr)
				log.Println("isPresent", true)
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
