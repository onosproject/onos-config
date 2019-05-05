// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

package southbound

import (
	"context"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// GetTarget returns a target based on the given key.
func GetTarget(key Key) (*GnmiTarget, error) {
	_, ok := targets[key]
	if ok {
		return targets[key].(*GnmiTarget), nil
	}
	return &GnmiTarget{}, fmt.Errorf("Client for %v does not exist, create first", key)

}

// ConnectTarget make the initial connection to the gnmi device
func (gnmiTarget *GnmiTarget) ConnectTarget(device Device) (Key, error) {
	//TODO make asyc
	//TODO lock channel to allow one request to device at each time
	dest, key := createDestination(device)
	ctx := context.Background()
	c, err := gclient.New(ctx, *dest)
	//c.handler := client.NotificationHandler{}
	if err != nil {
		return Key{}, fmt.Errorf("could not create a gNMI client: %v", err)
	}

	target := Target{
		Ctxt: ctx,
	}

	gnmiTarget.target = target
	gnmiTarget.Clt = c

	targets[key] = gnmiTarget

	return key, err
}

// CapabilitiesWithString allows a request for the capabilities by a string - can be empty
func (gnmiTarget GnmiTarget) CapabilitiesWithString(request string) (*gpb.CapabilityResponse, error) {
	r := &gpb.CapabilityRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.CapabilityRequest from %q : %v", *reqProto, err)
	}
	return gnmiTarget.Capabilities(r)
}

// Capabilities get capabilities according to a formatted request
func (gnmiTarget GnmiTarget) Capabilities(request *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	response, err := gnmiTarget.Clt.(*gclient.Client).Capabilities(gnmiTarget.target.Ctxt, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Capabilities(%q): %v", request.String(), err)
	}
	return response, nil
}

// GetWithString can make a get request according by a string - can be empty
func (gnmiTarget GnmiTarget) GetWithString(request string) (*gpb.GetResponse, error) {
	if request == "" {
		return nil, errors.New("cannot get and empty request")
	}
	r := &gpb.GetRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.GetRequest from %q : %v", *reqProto, err)
	}
	return gnmiTarget.Get(r)
}

// Get can make a get request according to a formatted request
func (gnmiTarget GnmiTarget) Get(request *gpb.GetRequest) (*gpb.GetResponse, error) {
	response, err := gnmiTarget.Clt.(*gclient.Client).Get(gnmiTarget.target.Ctxt, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Get(%q) : %v", request.String(), err)
	}
	return response, nil
}

// SetWithString can make a set request according by a string
func (gnmiTarget GnmiTarget) SetWithString(request string) (*gpb.SetResponse, error) {
	//TODO modify with key that gets target from map
	if request == "" {
		return nil, errors.New("cannot get and empty request")
	}
	r := &gpb.SetRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, fmt.Errorf("unable to parse gnmi.SetRequest from %q : %v", *reqProto, err)
	}
	return gnmiTarget.Set(r)
}

// Set can make a set request according to a formatted request
func (gnmiTarget GnmiTarget) Set(request *gpb.SetRequest) (*gpb.SetResponse, error) {
	response, err := gnmiTarget.Clt.(*gclient.Client).Set(gnmiTarget.target.Ctxt, request)
	if err != nil {
		return nil, fmt.Errorf("target returned RPC error for Set(%q) : %v", request.String(), err)
	}
	return response, nil
}

// Subscribe allows a client to request the target to stream it values of particular paths within the data tree.
func (gnmiTarget GnmiTarget) Subscribe(request *gpb.SubscribeRequest, handler client.NotificationHandler) error {
	//TODO currently establishing a throwaway client per each subscription request
	//this is due to the face that 1 NotificationHandler is allowed per client (1:1)
	//alternatively we could handle every connection request with one NotificationHandler
	//returing to the caller only the desired results.
	q, err := client.NewQuery(request)
	if err != nil {
		//TODO handle
	}
	q.Addrs = gnmiTarget.Destination.Addrs
	q.Timeout = gnmiTarget.Destination.Timeout
	q.Target = gnmiTarget.Destination.Target
	q.Credentials = gnmiTarget.Destination.Credentials
	q.TLS = gnmiTarget.Destination.TLS
	q.NotificationHandler = handler
	c := client.New()
	err = c.Subscribe(gnmiTarget.target.Ctxt, q, "")
	if err != nil {
		return fmt.Errorf("could not create a gNMI for subscription: %v", err)
	}
	gnmiTarget.Clt.(*gclient.Client).Subscribe(gnmiTarget.target.Ctxt, q)
	return nil

}

// NewSubscribeRequest returns a SubscribeRequest for the given paths
func NewSubscribeRequest(subscribeOptions *SubscribeOptions) (*gpb.SubscribeRequest, error) {
	var mode gpb.SubscriptionList_Mode
	switch subscribeOptions.Mode {
	case Once.String():
		mode = gpb.SubscriptionList_ONCE
	case Poll.String():
		mode = gpb.SubscriptionList_POLL
	case "":
		fallthrough
	case Stream.String():
		mode = gpb.SubscriptionList_STREAM
	default:
		return nil, fmt.Errorf("subscribe mode (%s) invalid", subscribeOptions.Mode)
	}

	var streamMode gpb.SubscriptionMode
	switch subscribeOptions.StreamMode {
	case OnChange.String():
		streamMode = gpb.SubscriptionMode_ON_CHANGE
	case Sample.String():
		streamMode = gpb.SubscriptionMode_SAMPLE
	case "":
		fallthrough
	case TargetDefined.String():
		streamMode = gpb.SubscriptionMode_TARGET_DEFINED
	default:
		return nil, fmt.Errorf("subscribe stream mode (%s) invalid", subscribeOptions.StreamMode)
	}

	prefixPath, err := ParseGNMIElements(SplitPath(subscribeOptions.Prefix))
	if err != nil {
		return nil, err
	}
	subList := &gpb.SubscriptionList{
		Subscription: make([]*gpb.Subscription, len(subscribeOptions.Paths)),
		Mode:         mode,
		UpdatesOnly:  subscribeOptions.UpdatesOnly,
		Prefix:       prefixPath,
	}
	for i, p := range subscribeOptions.Paths {
		gnmiPath, err := ParseGNMIElements(p)
		if err != nil {
			return nil, err
		}
		gnmiPath.Origin = subscribeOptions.Origin
		subList.Subscription[i] = &gpb.Subscription{
			Path:              gnmiPath,
			Mode:              streamMode,
			SampleInterval:    subscribeOptions.SampleInterval,
			HeartbeatInterval: subscribeOptions.HeartbeatInterval,
		}
	}
	return &gpb.SubscribeRequest{Request: &gpb.SubscribeRequest_Subscribe{
		Subscribe: subList}}, nil
}
