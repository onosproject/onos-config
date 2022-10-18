// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"github.com/golang/protobuf/proto"
	"io"

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
	Poll() error
}

// client gnmi client
type client struct {
	client *gclient.Client
}

// Subscribe calls gNMI subscription bacc
// sed on a given query
func (c *client) Subscribe(ctx context.Context, q baseClient.Query) error {
	err := c.client.Subscribe(ctx, q)
	go c.run(ctx)
	return errors.FromGRPC(err)
}

// Poll issues a poll request using the backing client
func (c *client) Poll() error {
	return c.client.Poll()
}

func (c *client) run(ctx context.Context) {
	log.Infof("Subscription response monitor started")
	for {
		err := c.client.Recv()
		if err != nil {
			log.Infof("Subscription response monitor stopped due to %v", err)
			return
		}
	}
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
