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
	changetypes "github.com/onosproject/onos-config/api/types/change"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store"
	streams "github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/values"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
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
	//Registering one listener for opStateChan
	opStateChan, err := mgr.Dispatcher.RegisterOpState(hash)
	if err != nil {
		log.Warn("Subscription present: ", err)
		return status.Error(codes.AlreadyExists, err.Error())
	}
	resChan := make(chan result)
	//Handles each subscribe request coming into the server, blocks until a new request or an error comes in
	go s.listenOnChannel(stream, mgr, hash, resChan, subscribe, opStateChan)

	res := <-resChan

	if !res.success {
		return status.Error(codes.Internal, res.err.Error())
	}
	return nil
}

func (s *Server) listenOnChannel(stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager, hash string,
	resChan chan result, subscribe *gnmi.SubscriptionList, opStateChan chan events.OperationalStateEvent) {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			log.Info("Subscription Terminated EOF")
			//Ignoring Errors during removal
			mgr.Dispatcher.UnregisterOperationalState(hash)
			resChan <- result{success: true, err: nil}
			break
		}

		if err != nil {
			code, ok := status.FromError(err)
			if ok && code.Code() == codes.Canceled {
				log.Info("Subscription Terminated, Canceled")
				mgr.Dispatcher.UnregisterOperationalState(hash)
				resChan <- result{success: true, err: nil}
			} else {
				log.Error("Error in subscription ", err)
				//Ignoring Errors during removal
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
		if subscribe.Subscription == nil {
			log.Error("No subscription paths, ignoring request ", in)
			resChan <- result{success: false, err: fmt.Errorf("no subscription paths in request")}
			break
		}

		//If the subscription mode is ONCE or POLL we immediately start a routine to collect the data
		version, err := extractSubscribeVersion(in)
		if mode != gnmi.SubscriptionList_STREAM {
			if err != nil {
				resChan <- result{success: false, err: err}
			} else {
				go s.collector(mgr, version, stream, subscribe, resChan, mode)
			}
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
			go listenForUpdates(stream, mgr, targets, version, subsStr, resChan)
			go listenForOpStateUpdates(opStateChan, stream, targets, subsStr, resChan)
		}
	}
}

func (s *Server) collector(mgr *manager.Manager, version devicetype.Version, stream gnmi.GNMI_SubscribeServer, request *gnmi.SubscriptionList, resChan chan result, mode gnmi.SubscriptionList_Mode) {
	for _, sub := range request.Subscription {
		_, version, err := mgr.CheckCacheForDevice(devicetype.ID(sub.GetPath().GetTarget()), devicetype.Type(""), version)
		if err != nil {
			log.Error("Error while collecting data from device cache ", err)
			resChan <- result{success: false, err: err}
		}
		//We get the stated of the device, for each path we build an update and send it out.
		update, err := s.getUpdate(version, request.Prefix, sub.Path)
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
func listenForUpdates(stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager,
	targets map[string]struct{}, version devicetype.Version, subs []*regexp.Regexp, resChan chan result) {
	for target := range targets {
		_, version, err := mgr.CheckCacheForDevice(devicetype.ID(target), devicetype.Type(""), version)
		if err != nil {
			log.Errorf("unable to get version from cache %s", err)
			return
		}
		go listenForDeviceUpdates(stream, mgr, devicetype.ID(target), version, subs, resChan)
	}
}

//For each update coming from the change channel we check if it's for a valid target and path then, if so, we send it NB
func listenForDeviceUpdates(stream gnmi.GNMI_SubscribeServer, mgr *manager.Manager,
	target devicetype.ID, version devicetype.Version, subs []*regexp.Regexp, resChan chan result) {
	eventCh := make(chan streams.Event)
	ctx, errWatch := mgr.DeviceChangesStore.Watch(devicetype.NewVersionedID(target, version), eventCh)
	if errWatch != nil {
		log.Errorf("Cant watch for changes on device %s. error %s", target, errWatch.Error())
		resChan <- result{success: false, err: errWatch}
		return
	}
	defer ctx.Close()
	for changeEvent := range eventCh {
		change, ok := changeEvent.Object.(*devicechange.DeviceChange)
		if !ok {
			log.Error("Could not convert event to DeviceChange")
			continue
		}
		if change.Status.State == changetypes.State_COMPLETE {
			for _, value := range change.Change.Values {
				if matchRegex(value.Path, subs) {
					pathGnmi, err := utils.ParseGNMIElements(utils.SplitPath(value.Path))
					if err != nil {
						log.Warn("Error in parsing path ", err)
						continue
					}
					log.Infof("Subscribe notification for %s on %s with value %s", pathGnmi, target, value.Value)
					err = buildAndSendUpdate(pathGnmi, string(target), value.Value, value.Removed, stream)
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
		target := opStateChange.Subject()
		_, targetPresent := targets[target]
		if targetPresent && matchRegex(opStateChange.Path(), subs) {
			pathArr := utils.SplitPath(opStateChange.Path())
			pathGnmi, err := utils.ParseGNMIElements(pathArr)
			if err != nil {
				log.Warn("Error in parsing path", err)
				resChan <- result{success: true, err: nil}
				continue
			}

			err = buildAndSendUpdate(pathGnmi, target, opStateChange.Value(), len(opStateChange.Value().Bytes) == 0, stream)
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

func buildAndSendUpdate(pathGnmi *gnmi.Path, target string, value *devicechange.TypedValue, removed bool,
	stream gnmi.GNMI_SubscribeServer) error {
	pathGnmi.Target = target
	var response *gnmi.SubscribeResponse
	var errGet error
	//if removed we issue a delete notification
	if removed {
		response, errGet = buildDeleteResponse(pathGnmi)
	} else {
		valueGnmi, err := values.NativeTypeToGnmiTypedValue(value)
		if err != nil {
			log.Warn("Unable to convert native value to gnmiValue", err)
			return err
		}

		update := &gnmi.Update{
			Path: pathGnmi,
			Val:  valueGnmi,
		}
		response, errGet = buildUpdateResponse(update)
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
	_, errDevice := manager.GetManager().DeviceStore.Get(topodevice.ID(target))
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
		log.Warn("Error in sending response to client ", err)
		return status.Error(codes.Internal, err.Error())
	}
	return nil
}

func extractSubscribeVersion(req *gnmi.SubscribeRequest) (devicetype.Version, error) {
	var version devicetype.Version
	for _, ext := range req.GetExtension() {
		if ext.GetRegisteredExt().GetId() == GnmiExtensionVersion {
			version = devicetype.Version(ext.GetRegisteredExt().GetMsg())
		} else {
			return "", status.Error(codes.InvalidArgument, fmt.Errorf("unexpected extension %d = '%s' in Subscribe()",
				ext.GetRegisteredExt().GetId(), ext.GetRegisteredExt().GetMsg()).Error())
		}
	}
	return version, nil
}
