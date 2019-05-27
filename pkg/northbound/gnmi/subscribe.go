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
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"io"
	"log"
	"strings"
	"time"
)

//per each subscribe request we receive the map is updated with a channel corresponding to the path.
var (
	PathToChannels = make(map[*gnmi.Path]chan *gnmi.Update)
)

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	updateChan := make(chan *gnmi.Update)
	var subscribe *gnmi.SubscriptionList
	//this for loop handles each subscribe request coming into the server
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Println("Subscription Terminated")
			return nil
		}

		if err != nil {
			log.Println("Error in subscription", err)
			return err
		}

		var mode gnmi.SubscriptionList_Mode

		if in.GetPoll() != nil {
			mode = gnmi.SubscriptionList_POLL
		} else {
			subscribe = in.GetSubscribe()
			mode = subscribe.Mode
		}
		stopped := make(chan struct{})
		//If the subscription mode is ONCE or POLL we immediately start a routine to collect the data
		if mode != gnmi.SubscriptionList_STREAM {
			go collector(updateChan, subscribe)
		}

		//This generate a subscribe response for one or more updates on the channel.
		// for Subscription_once messages also also closes the channel.
		go listenForUpdates(updateChan, stream, mode, stopped)
		//If the subscription mode is ONCE the channel need to be closed immediately
		if mode == gnmi.SubscriptionList_ONCE {
			<-stopped
			close(updateChan)
			return nil
		} else if mode == gnmi.SubscriptionList_STREAM {
			//for each path we pair it to the the channel.
			subs := subscribe.Subscription
			for _, sub := range subs {
				PathToChannels[sub.Path] = updateChan
			}
		}
	}

}

func listenForUpdates(updateChan chan *gnmi.Update, stream gnmi.GNMI_SubscribeServer,
	mode gnmi.SubscriptionList_Mode, stopped chan struct{}) {
	for update := range updateChan {
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
		sendResponse(response, stream)
		//For stream and Poll we also send a Sync Response
		//TODO make sure that for stream sending this every time adheres to spec.
		// see section #3.5.1.4 of gnmi-specification.md
		if mode != gnmi.SubscriptionList_ONCE {
			responseSync := &gnmi.SubscribeResponse_SyncResponse{
				SyncResponse: true,
			}
			response = &gnmi.SubscribeResponse{
				Response: responseSync,
			}
			sendResponse(response, stream)
		} else {
			//If the subscription mode is ONCE we read from the channel, build a response and issue it
			stopped <- struct{}{}
		}
	}
}

func sendResponse(response *gnmi.SubscribeResponse, stream gnmi.GNMI_SubscribeServer) {
	log.Println("Sending SubscribeResponse out to gNMI client", response)
	err := stream.Send(response)
	if err != nil {
		//TODO propagate error
		log.Println("Error in sending response to client", err)
	}
}

func collector(updateChan chan<- *gnmi.Update, request *gnmi.SubscriptionList) {
	for _, sub := range request.Subscription {
		update, err := getUpdate(request.Prefix, sub.Path)
		if err != nil {
			//TODO propagate error
			log.Println("Error while collecting data for subscribe once or poll", err)
		}
		updateChan <- update
	}
}

func broadcastConfigNotification() {
	mgr := manager.GetManager()
	changesChan, err := mgr.Dispatcher.Register("GnmiSubscribeNorthBound", false)
	if err != nil {
		log.Println("Error while subscribing to updates", err)
	}
	for update := range changesChan {
		target, changeInternal := getChange(update, mgr)
		//For every channel we cycle over the paths of the config change and if somebody is subscribed to it we send out
		for subscriptionPath, subscriptionChan := range PathToChannels {
			for _, changeValue := range changeInternal.Config {
				//FIXME this might prove expensive, find better way to store subscriptionPath and target in channels map
				subscriptionPathStr := utils.StrPath(subscriptionPath)
				if subscriptionPath.Target == target && strings.HasPrefix(changeValue.Path, subscriptionPathStr) {
					pathGnmi, err := utils.ParseGNMIElements(utils.SplitPath(changeValue.Path))
					if err != nil {
						log.Println("Error in parsing path", err)
						continue
					}
					pathGnmi.Target = subscriptionPath.Target
					sendUpdate(subscriptionChan, pathGnmi, changeValue.Value)
				}
			}
		}
	}
}

func broadcastOperationalNotification() {
	mgr := manager.GetManager()
	opStateChan, err := mgr.Dispatcher.RegisterOpState("GnmiSubscribeNorthBoundOpState")
	if err != nil {
		log.Println("Error while subscribing to updates", err)
	}
	for opStateChange := range opStateChan {
		target := events.Event(opStateChange).Subject()
		changeInternal := events.Event(opStateChange).Values()
		//For every channel we cycle over the paths of the config change and if somebody is subscribed to it we send out
		for subscriptionPath, subscriptionChan := range PathToChannels {
			for pathStr, value := range *changeInternal {
				//FIXME this might prove expensive, find better way to store subscriptionPath and target in channels map
				subscriptionPathStr := utils.StrPath(subscriptionPath)
				//TODO add method to do this
				pathArr := strings.Split(pathStr, "/")[1:]
				path, err := utils.ParseGNMIElements(pathArr)
				if err != nil {
					log.Println("Error in parsing path", pathStr)
				}
				if subscriptionPath.Target == target && strings.HasPrefix(pathStr, subscriptionPathStr) {
					sendUpdate(subscriptionChan, path, value)
				}
			}
		}
	}
}

func getChange(update events.ConfigEvent, mgr *manager.Manager) (string, *change.Change) {
	target := events.Event(update).Subject()
	changeID := update.ChangeID()
	changeInternal := mgr.ChangeStore.Store[changeID]
	return target, changeInternal
}

func sendUpdate(updateChan chan<- *gnmi.Update, path *gnmi.Path, value string) {
	typedValue := gnmi.TypedValue_StringVal{StringVal: value}
	valueGnmi := &gnmi.TypedValue{Value: &typedValue}

	update := &gnmi.Update{
		Path: path,
		Val:  valueGnmi,
	}
	updateChan <- update
}
