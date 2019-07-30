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
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	log "k8s.io/klog"
	"regexp"
	"time"
)

//internal struct to handle return of methods
type result struct {
	success bool
	err     error
}

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
	hash := store.B64(h.Sum(nil))
	//Registering one listener per NB app/client on both change and opStateChan
	changesChan, err := mgr.Dispatcher.RegisterNbi(hash)
	if err != nil {
		log.Warning("Subscription present: ", err)
		return status.Error(codes.AlreadyExists, err.Error())
	}
	opStateChan, err := mgr.Dispatcher.RegisterOpState(hash)
	if err != nil {
		log.Warning("Subscription present: ", err)
		return status.Error(codes.AlreadyExists, err.Error())
	}
	resChan := make(chan result)
	//Handles each subscribe request coming into the server, blocks until a new request or an error comes in
	go listenOnChannel(stream, mgr, hash, resChan, subscribe, changesChan, opStateChan)

	res := <-resChan

	if !res.success {
		return status.Error(codes.Internal, res.err.Error())
	}
	return nil
}

func listenOnChannel(stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager, hash string,
	resChan chan result, subscribe *gnmi.SubscriptionList, changesChan chan events.ConfigEvent,
	opStateChan chan events.OperationalStateEvent) {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Info("Subscription Terminated EOF")
			//Ignoring Errors during removal
			mgr.Dispatcher.UnregisterNbi(hash)
			mgr.Dispatcher.UnregisterOperationalState(hash)
			resChan <- result{success: true, err: nil}
			break
		}

		if err != nil {
			code, ok := status.FromError(err)
			if ok && code.Code() == codes.Canceled {
				log.Info("Subscription Terminated, Canceled")
				mgr.Dispatcher.UnregisterNbi(hash)
				mgr.Dispatcher.UnregisterOperationalState(hash)
				resChan <- result{success: true, err: nil}
			} else {
				log.Error("Error in subscription ", err)
				//Ignoring Errors during removal
				mgr.Dispatcher.UnregisterNbi(hash)
				mgr.Dispatcher.UnregisterOperationalState(hash)
				resChan <- result{success: false, err: err}
			}
			break
		}

		var mode gnmi.SubscriptionList_Mode

		if in.GetPoll() != nil {
			mode = gnmi.SubscriptionList_POLL
		} else {
			subscribe = in.GetSubscribe()
			mode = subscribe.Mode
		}

		//If there are no paths in the request such request is ignored
		//TODO evaluate throwing error
		if subscribe.Subscription == nil {
			log.Warning("No subscription paths, ignoring request ", in)
			continue
		}

		//If the subscription mode is ONCE or POLL we immediately start a routine to collect the data
		if mode != gnmi.SubscriptionList_STREAM {
			go collector(stream, subscribe, resChan, mode)
		} else {
			subs := subscribe.Subscription
			//FAST way to identify if target and subscription is present
			subsStr := make([]*regexp.Regexp, 0)
			targets := make(map[string]struct{})
			for _, sub := range subs {
				subscriptionPathStr := utils.StrPath(sub.Path)
				subsStr = append(subsStr, utils.MatchWildcardRegexp(subscriptionPathStr))
				targets[sub.Path.Target] = struct{}{}
			}
			//Each subscription request spawns a go routing listening for related events for the target and the paths
			go listenForUpdates(changesChan, stream, mgr, targets, subsStr, resChan)
			go listenForOpStateUpdates(opStateChan, stream, targets, subsStr, resChan)
		}
	}
}

func collector(stream gnmi.GNMI_SubscribeServer, request *gnmi.SubscriptionList, resChan chan result, mode gnmi.SubscriptionList_Mode) {
	for _, sub := range request.Subscription {
		//We get the stated of the device, for each path we build an update and send it out.
		update, err := getUpdate(request.Prefix, sub.Path)
		if err != nil {
			log.Error("Error while collecting data for subscribe once or poll ", err)
			resChan <- result{success: false, err: err}
		}
		response := buildUpdateResponse(update)
		err = sendResponse(response, stream)
		if err != nil {
			log.Error("Error sending response ", err)
			resChan <- result{success: false, err: err}
		}
	}
	responseSync := buildSyncResponse()
	err := sendResponse(responseSync, stream)
	if err != nil {
		log.Error("Error sending sync response ", err)
		resChan <- result{success: false, err: err}
	} else if mode != gnmi.SubscriptionList_POLL {
		//Sending only if we need to finish listening because of ONCE
		// if POLL we need to keep the channel open
		resChan <- result{success: true, err: nil}
	}
}

//For each update coming from the change channel we check if it's for a valid target and path then, if so, we send it NB
func listenForUpdates(changeChan chan events.ConfigEvent, stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager,
	targets map[string]struct{}, subs []*regexp.Regexp, resChan chan result) {
	for update := range changeChan {
		target, changeInternal := getChangeFromEvent(update, mgr)
		_, targetPresent := targets[target]
		if targetPresent && changeInternal != nil {
			for _, changeValue := range changeInternal.Config {
				if matchRegex(changeValue.Path, subs) {
					pathGnmi, err := utils.ParseGNMIElements(utils.SplitPath(changeValue.Path))
					if err != nil {
						log.Warning("Error in parsing path ", err)
						continue
					}
					err = buildAndSendUpdate(pathGnmi, target, &changeValue.TypedValue, stream)
					if err != nil {
						log.Error("Error in sending update path ", err)
						resChan <- result{success: false, err: err}
					}
				}
			}
		}
	}
}

//For each update coming from the state channel we check if it's for a valid target and path then, if so, we send it NB
func listenForOpStateUpdates(opStateChan chan events.OperationalStateEvent, stream gnmi.GNMI_SubscribeServer,
	targets map[string]struct{}, subs []*regexp.Regexp, resChan chan result) {
	for opStateChange := range opStateChan {
		target := events.Event(opStateChange).Subject()
		_, targetPresent := targets[target]
		if targetPresent {
			changeInternal := events.Event(opStateChange).Values()
			for pathStr, value := range *changeInternal {
				if matchRegex(pathStr, subs) {
					pathArr := utils.SplitPath(pathStr)
					pathGnmi, err := utils.ParseGNMIElements(pathArr)
					if err != nil {
						log.Warning("Error in parsing path", err)
						resChan <- result{success: true, err: nil}
						continue
					}

					// FIXME Change this to create a typed value of the type sent up
					strValue := change.CreateTypedValueString(value)

					err = buildAndSendUpdate(pathGnmi, target, strValue, stream)
					if err != nil {
						log.Error("Error in sending update path ", err)
						resChan <- result{success: false, err: err}
					}
				}
			}
		}
	}
}

func matchRegex(path string, subs []*regexp.Regexp) bool {
	for _, s := range subs {
		if s.MatchString(path) {
			return true
		}
	}
	return false
}

func buildAndSendUpdate(pathGnmi *gnmi.Path, target string, value *change.TypedValue, stream gnmi.GNMI_SubscribeServer) error {
	pathGnmi.Target = target
	var response *gnmi.SubscribeResponse
	//if value is empty it's a delete operation, thus we issue a delete notification
	//TODO can probably be moved to Value.Remove
	if len(value.Value) > 0 {
		valueGnmi, err := values.NativeTypeToGnmiTypedValue(value)
		if err != nil {
			log.Warning("Unable to convert native value to gnmiValue", err)
			return err
		}

		update := &gnmi.Update{
			Path: pathGnmi,
			Val:  valueGnmi,
		}
		response = buildUpdateResponse(update)
	} else {
		response = buildDeleteResponse(pathGnmi)
	}
	err := sendResponse(response, stream)
	if err != nil {
		return err
	}
	responseSync := buildSyncResponse()
	err = sendResponse(responseSync, stream)
	return err
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

func buildDeleteResponse(delete *gnmi.Path) *gnmi.SubscribeResponse {
	deleteArray := []*gnmi.Path{delete}
	notification := &gnmi.Notification{
		Timestamp: time.Now().Unix(),
		Delete:    deleteArray,
	}
	responseUpdate := &gnmi.SubscribeResponse_Update{
		Update: notification,
	}
	response := &gnmi.SubscribeResponse{
		Response: responseUpdate,
	}
	return response
}

func sendResponse(response *gnmi.SubscribeResponse, stream gnmi.GNMI_SubscribeServer) error {
	log.Info("Sending SubscribeResponse out to gNMI client ", response)
	err := stream.Send(response)
	if err != nil {
		log.Warning("Error in sending response to client ", err)
		return status.Error(codes.Internal, err.Error())
	}
	return nil
}

func getChangeFromEvent(update events.ConfigEvent, mgr *manager.Manager) (string, *change.Change) {
	target := events.Event(update).Subject()
	changeID := update.ChangeID()
	changeInternal, ok := mgr.ChangeStore.Store[changeID]
	if !ok {
		log.Warning("No change found for: ", changeID)
	}
	return target, changeInternal
}
