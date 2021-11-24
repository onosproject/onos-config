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
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/golang/protobuf/proto"

	"github.com/onosproject/onos-lib-go/pkg/errors"
	baseClient "github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"

	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// Client gNMI client interface
type Client interface {
	io.Closer
	Capabilities(ctx context.Context, r *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error)
	CapabilitiesWithString(ctx context.Context, request string) (*gpb.CapabilityResponse, error)
	Get(ctx context.Context, r *gpb.GetRequest) (*gpb.GetResponse, error)
	GetWithString(ctx context.Context, request string) (*gpb.GetResponse, error)
	Set(ctx context.Context, r *gpb.SetRequest) (*gpb.SetResponse, error)
	SetWithString(ctx context.Context, request string) (*gpb.SetResponse, error)
	Subscribe(ctx context.Context, q baseClient.Query) error
}

// client gnmi client
type client struct {
	client *gclient.Client
}

// Subscribe calls gNMI subscription based on a given query
func (c *client) Subscribe(ctx context.Context, q baseClient.Query) error {
	err := c.client.Subscribe(ctx, q)
	return errors.FromGRPC(err)
}

// newGNMIClient creates a new gnmi client
func newGNMIClient(ctx context.Context, d baseClient.Destination, opts []grpc.DialOption) (*client, *grpc.ClientConn, error) {
	switch d.TLS {
	case nil:
		opts = append(opts, grpc.WithInsecure())
	default:
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(d.TLS)))
	}

	if d.Credentials != nil {
		secure := true
		if d.TLS == nil {
			secure = false
		}
		pc := newPassCred(d.Credentials.Username, d.Credentials.Password, secure)
		opts = append(opts, grpc.WithPerRPCCredentials(pc))
	}

	gCtx, cancel := context.WithTimeout(ctx, d.Timeout)
	defer cancel()

	addr := ""
	if len(d.Addrs) != 0 {
		addr = d.Addrs[0]
	}
	conn, err := grpc.DialContext(gCtx, addr, opts...)
	if err != nil {
		return nil, nil, errors.NewInternal("Dialer(%s, %v): %v", addr, d.Timeout, err)
	}

	cl, err := gclient.NewFromConn(gCtx, conn, d)
	if err != nil {
		return nil, nil, err
	}

	gnmiClient := &client{
		client: cl,
	}

	return gnmiClient, conn, nil

}

// Capabilities returns the capabilities of the target
func (c *client) Capabilities(ctx context.Context, req *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	capResponse, err := c.client.Capabilities(ctx, req)
	return capResponse, errors.FromGRPC(err)
}

// Get calls gnmi Get RPC
func (c *client) Get(ctx context.Context, req *gpb.GetRequest) (*gpb.GetResponse, error) {
	getResponse, err := c.client.Get(ctx, req)
	return getResponse, errors.FromGRPC(err)
}

// Set calls gnmi Set RPC
func (c *client) Set(ctx context.Context, req *gpb.SetRequest) (*gpb.SetResponse, error) {
	setResponse, err := c.client.Set(ctx, req)
	return setResponse, errors.FromGRPC(err)
}

// CapabilitiesWithString allows a request for the capabilities by a string - can be empty
func (c *client) CapabilitiesWithString(ctx context.Context, request string) (*gpb.CapabilityResponse, error) {
	r := &gpb.CapabilityRequest{}
	reqProto := &request
	if err := proto.UnmarshalText(*reqProto, r); err != nil {
		return nil, errors.NewInvalid("unable to unmarshal gnmi.CapabilityRequest from %v : %v", *reqProto, err)
	}
	capResponse, err := c.client.Capabilities(ctx, r)
	return capResponse, errors.FromGRPC(err)
}

// GetWithString can make a get request based on a given a string request - can be empty
func (c *client) GetWithString(ctx context.Context, request string) (*gpb.GetResponse, error) {
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
func (c *client) SetWithString(ctx context.Context, request string) (*gpb.SetResponse, error) {
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
func (c *client) Close() error {
	return c.client.Close()
}

var _ Client = &client{}
