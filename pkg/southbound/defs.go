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
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/client"
	gpb "github.com/openconfig/gnmi/proto/gnmi"
)

// DeviceID Target key; can be extended, for now is ip:port
type DeviceID struct {
	DeviceID string
}

// TargetIf defines the API for Target
type TargetIf interface {
	ConnectTarget(ctx context.Context, device devicepb.Device) (DeviceID, error)
	//Capabilities(ctx context.Context, request *gpb.CapabilityRequest) (*gpb.CapabilityResponse, error)
	CapabilitiesWithString(ctx context.Context, request string) (*gpb.CapabilityResponse, error)
	Get(ctx context.Context, request *gpb.GetRequest) (*gpb.GetResponse, error)
	GetWithString(ctx context.Context, request string) (*gpb.GetResponse, error)
	Set(ctx context.Context, request *gpb.SetRequest) (*gpb.SetResponse, error)
	SetWithString(ctx context.Context, request string) (*gpb.SetResponse, error)
	Subscribe(ctx context.Context, request *gpb.SubscribeRequest, handler client.ProtoHandler) error
}

// Target struct for connecting to gNMI
type Target struct {
	Destination client.Destination
	Clt         GnmiClient
	Ctx         context.Context
}

// NewTarget is a method for constructing a target
func NewTarget() TargetIf {
	return &Target{}
}

// SubscribeOptions is the gNMI subscription request options
type SubscribeOptions struct {
	UpdatesOnly       bool
	Prefix            string
	Mode              string
	StreamMode        string
	SampleInterval    uint64
	HeartbeatInterval uint64
	Paths             [][]string
	Origin            string
}
