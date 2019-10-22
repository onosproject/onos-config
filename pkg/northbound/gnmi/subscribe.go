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
	changeTypes "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
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
	//TODO remove
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

// TODO listenOnChannel works on legacy, non-atomix stores, remove the non opstate stuff
func listenOnChannel(stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager, hash string,
	resChan chan result, subscribe *gnmi.SubscriptionList, changesChan chan events.ConfigEvent,
	opStateChan chan events.OperationalStateEvent) {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Info("Subscription Terminated EOF")
			//Ignoring Errors during removal
			//TODO remove NBI
			mgr.Dispatcher.UnregisterNbi(hash)
			mgr.Dispatcher.UnregisterOperationalState(hash)
			resChan <- result{success: true, err: nil}
			break
		}

		if err != nil {
			code, ok := status.FromError(err)
			if ok && code.Code() == codes.Canceled {
				log.Info("Subscription Terminated, Canceled")
				//TODO remove NBI
				mgr.Dispatcher.UnregisterNbi(hash)
				mgr.Dispatcher.UnregisterOperationalState(hash)
				resChan <- result{success: true, err: nil}
			} else {
				log.Error("Error in subscription ", err)
				//Ignoring Errors during removal
				//TODO remove NBI
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
			go listenForNewUpdates(stream, mgr, targets, subsStr, resChan)
			go listenForOpStateUpdates(opStateChan, stream, targets, subsStr, resChan)
		}
	}
}

func collector(stream gnmi.GNMI_SubscribeServer, request *gnmi.SubscriptionList, resChan chan result, mode gnmi.SubscriptionList_Mode) {
	for _, sub := range request.Subscription {
		//We get the stated of the device, for each path we build an update and send it out.
		//Already based on the new atomix based store
		update, err := getUpdate(request.Prefix, sub.Path)
		if err != nil {
			log.Error("Error while collecting data for subscribe once or poll ", err)
			resChan <- result{success: false, err: err}
		}
		response, errGet := buildUpdateResponse(update)
		if errGet != nil {
			log.Error("Error Retrieving Device", err)
			resChan <- result{success: false, err: err}
		}
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
// Deprecated: computeChange works on legacy, non-atomix stores
func listenForUpdates(changeChan chan events.ConfigEvent, stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager,
	targets map[string]struct{}, subs []*regexp.Regexp, resChan chan result) {
	for update := range changeChan {
		target, changeInternal := getChangeFromEvent(update, mgr)
		_, targetPresent := targets[target]
		if targetPresent && changeInternal != nil {
			//if the device is registered it has a listener, if not we assume the device is not in the system and
			// send an immediate response
			respChan, ok := mgr.Dispatcher.GetResponseListener(devicetopo.ID(target))
			if ok {
				go listenForDeviceUpdates(respChan, changeInternal, subs, target, stream, resChan)
			} else {
				log.Infof("Device %s not registered, reporting update to subscribers immediately ", target)
				sendSubscribeResponse(changeInternal, subs, target, stream, resChan)
			}
		}
	}
}

//Listens for an update from the device after a config has been changed, if update does not arrive after 5 secs
// (same as set.go) we consider the device as un-responsive and do not send the update to subscribers.
// Also if there is an error we do not send it up.
// Deprecated: computeChange works on legacy, non-atomix stores
func listenForDeviceUpdates(respChan <-chan events.DeviceResponse, changeInternal *change.Change,
	subs []*regexp.Regexp, target string, stream gnmi.GNMI_SubscribeServer, resChan chan result) {
	select {
	case response := <-respChan:
		switch eventType := response.EventType(); eventType {
		case events.EventTypeSubscribeNotificationSetConfig:
			log.Info("Set is properly configured ", response.ChangeID())
			sendSubscribeResponse(changeInternal, subs, target, stream, resChan)
		case events.EventTypeSubscribeErrorNotificationSetConfig:
			log.Error("Set is not properly configured, not sending subscribe update ", response.ChangeID())
		default:
			log.Errorf("Unrecognized reply, not sending subscribe update for %s. error %s", response.ChangeID(),
				response.Error())
		}
	case <-time.After(5 * time.Second):
		log.Error("Timeout on waiting for device reply ", target)
	}
}

//For each update coming from the change channel we check if it's for a valid target and path then, if so, we send it NB
func listenForNewUpdates(stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager,
	targets map[string]struct{}, subs []*regexp.Regexp, resChan chan result) {
	for target := range targets {
		go listenForNewDeviceUpdates(stream, mgr, target, subs, resChan)
	}
}

//For each update coming from the change channel we check if it's for a valid target and path then, if so, we send it NB
func listenForNewDeviceUpdates(stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager,
	target string, subs []*regexp.Regexp, resChan chan result) {
	changeChan := make(chan *devicechangetypes.DeviceChange)
	errWatch := mgr.DeviceChangesStore.Watch(devicetopo.ID(target), changeChan)
	if errWatch != nil {
		log.Errorf("Cant watch for changes on device %s. error %s", target, errWatch.Error())
		resChan <- result{success: false, err: errWatch}
	}
	for changeEvent := range changeChan {
		if changeEvent.Status.State == changeTypes.State_COMPLETE {
			for _, value := range changeEvent.Change.Values {
				if matchRegex(value.Path, subs) {
					pathGnmi, err := utils.ParseGNMIElements(utils.SplitPath(value.Path))
					if err != nil {
						log.Warning("Error in parsing path ", err)
						continue
					}
					log.Infof("NEW - Subscribe notification for %s on %s with value %s", pathGnmi, target, value.Value)
					//TODO uncomment in swap patch
					//err = buildAndSendUpdate(pathGnmi, target, value.Value, stream)
					//if err != nil {
					//	log.Error("Error in sending update path ", err)
					//	resChan <- result{success: false, err: err}
					//}
				}
			}
		}
	}
}

// Deprecated: computeChange works on legacy, non-atomix stores
func sendSubscribeResponse(changeInternal *change.Change, subs []*regexp.Regexp, target string,
	stream gnmi.GNMI_SubscribeServer, resChan chan result) {
	for _, changeValue := range changeInternal.Config {
		if matchRegex(changeValue.Path, subs) {
			pathGnmi, err := utils.ParseGNMIElements(utils.SplitPath(changeValue.Path))
			if err != nil {
				log.Warning("Error in parsing path ", err)
				continue
			}
			log.Infof("OLD - Subscribe notification for %s on %s with value %s", pathGnmi, target, changeValue.GetValue())
			err = buildAndSendUpdate(pathGnmi, target, changeValue.GetValue(), stream)
			if err != nil {
				log.Error("Error in sending update path ", err)
				resChan <- result{success: false, err: err}
			}
		}
	}
}

//For each update coming from the state channel we check if it's for a valid target and path then, if so, we send it NB
func listenForOpStateUpdates(opStateChan chan events.OperationalStateEvent, stream gnmi.GNMI_SubscribeServer,
	targets map[string]struct{}, subs []*regexp.Regexp, resChan chan result) {
	for opStateChange := range opStateChan {
		target := opStateChange.Subject()
		_, targetPresent := targets[target]
		if targetPresent && matchRegex(opStateChange.Path(), subs) {
			pathArr := utils.SplitPath(opStateChange.Path())
			pathGnmi, err := utils.ParseGNMIElements(pathArr)
			if err != nil {
				log.Warning("Error in parsing path", err)
				resChan <- result{success: true, err: nil}
				continue
			}

			err = buildAndSendUpdate(pathGnmi, target, opStateChange.Value(), stream)
			if err != nil {
				log.Error("Error in sending update path ", err)
				resChan <- result{success: false, err: err}
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

func buildAndSendUpdate(pathGnmi *gnmi.Path, target string, value *devicechangetypes.TypedValue, stream gnmi.GNMI_SubscribeServer) error {
	pathGnmi.Target = target
	var response *gnmi.SubscribeResponse
	var errGet error
	//if value is empty it's a delete operation, thus we issue a delete notification
	//TODO can probably be moved to Value.Remove
	if len(value.Bytes) > 0 {
		valueGnmi, err := values.NativeTypeToGnmiTypedValue(value)
		if err != nil {
			log.Warning("Unable to convert native value to gnmiValue", err)
			return err
		}

		update := &gnmi.Update{
			Path: pathGnmi,
			Val:  valueGnmi,
		}
		response, errGet = buildUpdateResponse(update)
	} else {
		response, errGet = buildDeleteResponse(pathGnmi)
	}
	if errGet != nil {
		return errGet
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

func buildUpdateResponse(update *gnmi.Update) (*gnmi.SubscribeResponse, error) {
	updateArray := make([]*gnmi.Update, 0)
	updateArray = append(updateArray, update)
	notification := &gnmi.Notification{
		Timestamp: time.Now().Unix(),
		Update:    updateArray,
	}
	return buildSubscribeResponse(notification, update.Path.Target)
}

func buildDeleteResponse(delete *gnmi.Path) (*gnmi.SubscribeResponse, error) {
	deleteArray := []*gnmi.Path{delete}
	notification := &gnmi.Notification{
		Timestamp: time.Now().Unix(),
		Delete:    deleteArray,
	}
	return buildSubscribeResponse(notification, delete.Target)
}

func buildSubscribeResponse(notification *gnmi.Notification, target string) (*gnmi.SubscribeResponse, error) {
	responseUpdate := &gnmi.SubscribeResponse_Update{
		Update: notification,
	}
	response := &gnmi.SubscribeResponse{
		Response: responseUpdate,
	}
	_, errDevice := manager.GetManager().DeviceStore.Get(devicetopo.ID(target))
	if errDevice != nil && status.Convert(errDevice).Code() == codes.NotFound {
		response.Extension = []*gnmi_ext.Extension{
			{
				Ext: &gnmi_ext.Extension_RegisteredExt{
					RegisteredExt: &gnmi_ext.RegisteredExtension{
						Id:  GnmiExtensionDevicesNotConnected,
						Msg: []byte(target),
					},
				},
			},
		}
	} else if errDevice != nil {
		//handling gRPC errors
		return nil, errDevice
	}
	return response, nil
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

// Deprecated: computeChange works on legacy, non-atomix stores
func getChangeFromEvent(update events.ConfigEvent, mgr *manager.Manager) (string, *change.Change) {
	target := update.Subject()
	changeID := update.ChangeID()
	changeInternal, ok := mgr.ChangeStore.Store[changeID]
	if !ok {
		log.Warning("No change found for: ", changeID)
	}
	return target, changeInternal
}
