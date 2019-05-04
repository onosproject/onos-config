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

	"github.com/opennetworkinglab/onos-config/southbound/topocache"

	"github.com/golang/protobuf/proto"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// GnmiTarget struct for connecting to gNMI
type GnmiTarget struct {
	target Target
	Clt    client.Impl
}

// ConnectTarget make the initial connection to the gnmi device
func (gnmiTarget *GnmiTarget) ConnectTarget(device topocache.Device) (Key, error) {
	dest, key := createDestination(device)
	ctx := context.Background()
	c, err := gclient.New(ctx, *dest)
	if err != nil {
		return Key{}, fmt.Errorf("could not create a gNMI client: %v", err)
	}

	target := Target{
		Ctxt: ctx,
	}

	gnmiTarget.target = target
	gnmiTarget.Clt = c

	//targets[key] = gnmiTarget
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
