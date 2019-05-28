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

package southbound

import (
	"context"
	"github.com/openconfig/gnmi/client"
	gclient "github.com/openconfig/gnmi/client/gnmi"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// GnmiClient : interface to hide struct dependency on gnmi.client. Can be overridden by tests.
type GnmiClient interface {
	Capabilities(ctx context.Context, r *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error)
	Get(ctx context.Context, r *gpb.GetRequest) (*gpb.GetResponse, error)
	Set(ctx context.Context, r *gpb.SetRequest) (*gpb.SetResponse, error)
	Subscribe(ctx context.Context, q client.Query) error
}

// GnmiClientFactory : Default GnmiClient creation.
var GnmiClientFactory = func(ctx context.Context, d client.Destination) (GnmiClient, error) {
	openconfigClient, err := gclient.New(ctx, d)
	if err != nil {
		return nil, err
	}
	c := openconfigClient.(*gclient.Client)
	return gnmiClientImpl{
		c: c,
	}, err
}

// GnmiClientImpl : Default implementation of GnmiClient based on the openconfig GNMI client.
type gnmiClientImpl struct {
	c *gclient.Client
}

// Capabilities : GNMI capabilities
func (client gnmiClientImpl) Capabilities(ctx context.Context, r *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error) {
	return client.c.Capabilities(ctx, r)
}

// Get : GNMI get
func (client gnmiClientImpl) Get(ctx context.Context, r *gpb.GetRequest) (*gpb.GetResponse, error) {
	return client.c.Get(ctx, r)
}

// Set : GNMI set
func (client gnmiClientImpl) Set(ctx context.Context, r *gpb.SetRequest) (*gpb.SetResponse, error) {
	return client.c.Set(ctx, r)
}

// Subscribe : GNMI subscribe
func (client gnmiClientImpl) Subscribe(ctx context.Context, q client.Query) error {
	return client.c.Subscribe(ctx, q)
}
