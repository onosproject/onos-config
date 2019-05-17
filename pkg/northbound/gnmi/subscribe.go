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
	"github.com/onosproject/onos-config/pkg/listener"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"io"
	"log"
	"time"
)

//per each subscribe request we receive the map is updated with a channel corresponding to the path.
var (
	PathToChannels = make(map[*gnmi.Path]chan *gnmi.Update)
)

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	ch := make(chan *gnmi.Update)
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

		subscribe := in.GetSubscribe()
		mode := subscribe.Mode
		stopped := make(chan struct{})
		//If the subscription mode is ONCE we immediately start a routine to collect the data
		if mode == gnmi.SubscriptionList_ONCE {
			go collector(ch, subscribe)
		}
		//TODO for POLL type spawn a routine that periodically checks for updates
		//This generate a subscribe response for one or more updates on the channel.
		// for Subscription_once messages also also closes the channel.
		go func() {
			for update := range ch {
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
	log.Println("Sending SubscribeResponse out to gNMI client", response)
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

func broadcastNotification() {
	mgr := manager.GetManager()
	changes, err := listener.Register("GnmiSubscribeNorthBound", false)
	if err != nil {
		log.Println("Error while subscribing to updates", err)
	}
	for update := range changes {
		//TODO needs to be filtered for appropriate paths in the change
		// currently broadcasting to everybody
		for _, ch := range PathToChannels {
			values := update.Values()
			target := update.Subject()
			changeID := (*values)[events.ChangeID]
			changeInternal := mgr.ChangeStore.Store[changeID]
			err := sendUpdate(target, changeInternal, ch)
			if err != nil {
				log.Println("Error while parsing path ", err)
			}
		}
	}
}

func sendUpdate(target string, c *change.Change, ch chan *gnmi.Update) error {
	for _, changeValue := range c.Config {
		elems := utils.SplitPath(changeValue.Path)
		pathElemsRefs, parseError := utils.ParseGNMIElements(elems)

		if parseError != nil {
			return parseError
		}
		//TODO use proper type of value, re-use code in get
		typedValue := gnmi.TypedValue_StringVal{StringVal: changeValue.Value}
		value := &gnmi.TypedValue{Value: &typedValue}
		updatePath := &gnmi.Path{Elem: pathElemsRefs.Elem, Target: target}
		update := &gnmi.Update{
			Path: updatePath,
			Val:  value,
		}
		ch <- update
	}
	return nil
}
