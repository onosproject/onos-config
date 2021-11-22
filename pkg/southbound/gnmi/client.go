// Copyright 2021-present Open Networking Foundation.
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
	"context"

	"github.com/golang/protobuf/proto"

	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// Client gNMI client interface
type Client interface {
	Capabilities(ctx context.Context, r *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error)
	CapabilitiesWithString(ctx context.Context, request string) (*gpb.CapabilityResponse, error)
	Get(ctx context.Context, r *gpb.GetRequest) (*gpb.GetResponse, error)
	GetWithString(ctx context.Context, request string) (*gpb.GetResponse, error)
	Set(ctx context.Context, r *gpb.SetRequest) (*gpb.SetResponse, error)
	SetWithString(ctx context.Context, request string) (*gpb.SetResponse, error)
	Subscribe(ctx context.Context, q client.Query) error
	Close() error
}

// GNMIClient gnmi client
type GNMIClient struct {
	client *gclient.Client
}

// Subscribe calls gNMI subscription based on a given query
func (c *GNMIClient) Subscribe(ctx context.Context, q client.Query) error {
	err := c.client.Subscribe(ctx, q)
	return errors.FromGRPC(err)
}

// newGNMIClient creates a new gnmi client
func newGNMIClient(ctx context.Context, destination client.Destination) (*GNMIClient, error) {
	client, err := gclient.New(ctx, destination)
	if err != nil {
		return nil, err
	}

	gnmiClient := &GNMIClient{
		client: client.(*gclient.Client),
	}

	return gnmiClient, nil

}

// Capabilities returns the capabilities of the target
func (c *GNMIClient) Capabilities(ctx context.Context, req *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	capResponse, err := c.client.Capabilities(ctx, req)
	return capResponse, errors.FromGRPC(err)
}

// Get calls gnmi Get RPC
func (c *GNMIClient) Get(ctx context.Context, req *gpb.GetRequest) (*gpb.GetResponse, error) {
	getResponse, err := c.client.Get(ctx, req)
	return getResponse, errors.FromGRPC(err)
}

// Set calls gnmi Set RPC
func (c *GNMIClient) Set(ctx context.Context, req *gpb.SetRequest) (*gpb.SetResponse, error) {
	setResponse, err := c.client.Set(ctx, req)
	return setResponse, errors.FromGRPC(err)
}

// CapabilitiesWithString allows a request for the capabilities by a string - can be empty
func (c *GNMIClient) CapabilitiesWithString(ctx context.Context, request string) (*gpb.CapabilityResponse, error) {
	r := &gpb.CapabilityRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, errors.NewInvalid("unable to unmarshal gnmi.CapabilityRequest from %v : %v", *reqProto, err)
	}
	capResponse, err := c.client.Capabilities(ctx, r)
	return capResponse, errors.FromGRPC(err)
}

// GetWithString can make a get request based on a given a string request - can be empty
func (c *GNMIClient) GetWithString(ctx context.Context, request string) (*gpb.GetResponse, error) {
	if request == "" {
		return nil, errors.NewInvalid("cannot get an empty request")
	}
	r := &gpb.GetRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, errors.NewInvalid("unable to unmarshal gnmi getRequest from %v : %v", *reqProto, err)
	}
	getResponse, err := c.client.Get(ctx, r)
	return getResponse, errors.FromGRPC(err)
}

// SetWithString can make a set request based on a given string request
func (c *GNMIClient) SetWithString(ctx context.Context, request string) (*gpb.SetResponse, error) {
	if request == "" {
		return nil, errors.NewInvalid("cannot set an empty request")
	}
	r := &gpb.SetRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, errors.NewInvalid("unable to unmarshal gnmi set request from %v: %v", *reqProto, err)
	}
	setResponse, err := c.client.Set(ctx, r)
	return setResponse, errors.FromGRPC(err)
}

// Close closes the gnmi client
func (c *GNMIClient) Close() error {
	return c.client.Close()
}

var _ Client = &GNMIClient{}
