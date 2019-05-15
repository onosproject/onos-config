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
	"github.com/openconfig/gnmi/proto/gnmi"
	"io"
	"log"
	"time"
)

//per each subscribe request we reiceve the map is upated with a channel coprresponding to the path.
var (
	PathToChannels map[*gnmi.Path]chan *gnmi.Update
)

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	//Create channel
	//Spawn go routine with the given channel
	//forever
	//read from channel
	//produce update message
	//send notification with update message
	//
	//Forever
	//read subscribe request message
	//add subscription to the subscriber DB for this session --> map update
	//	if this is subscribe once
	//	go spawn gatherer for specified paths to generate update events
	ch := make(chan *gnmi.Update)

	//this for loop handles each subsicribe request coming into the server
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

		subscribe := in.GetSubscribe()
		mode := subscribe.Mode
		stopped := make(chan struct{})
		//If the subscription mode is ONCE we immediately start a routine to collect the data
		if mode == gnmi.SubscriptionList_ONCE {
			go collector(ch, subscribe)
		}
		//This generate a subscribe response for one or more updates on the channel.
		// for Subscription_once messages also also closes the channel.
		go func() {
			update := <-ch
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
			//If the subscription mode is ONCE we read from the channel, build a response and issue it
			if mode == gnmi.SubscriptionList_ONCE {
				responseUpdate := &gnmi.SubscribeResponse_SyncResponse{
					SyncResponse: true,
				}
				response := &gnmi.SubscribeResponse{
					Response: responseUpdate,
				}
				sendResponse(response, stream)
				stopped <- struct{}{}
			}
		}()
		//If the subscription mode is ONCE the channel need to be closed immediately
		if mode == gnmi.SubscriptionList_ONCE {
			<-stopped
			return nil
		}
		//for each path we pair it to the the channel.
		subs := subscribe.Subscription
		for _, sub := range subs {
			PathToChannels[sub.Path] = ch
		}
	}

}

func sendResponse(response *gnmi.SubscribeResponse, stream gnmi.GNMI_SubscribeServer) {
	err := stream.Send(response)
	if err != nil {
		//TODO remove channel registrations
		log.Println("Error in sending response to client")
	}
}

func collector(ch chan *gnmi.Update, request *gnmi.SubscriptionList) {
	for _, sub := range request.Subscription {
		update, err := GetUpdate(sub.Path)
		if err != nil {
			log.Println("Error while collecting data for subscribe once", err)
		}
		ch <- update
	}
}
