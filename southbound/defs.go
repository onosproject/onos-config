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
	"time"

	"github.com/openconfig/gnmi/client"
)

// Device represents a target device
type Device struct {
	Addr, Target, Usr, Pwd, CaPath, CertPath, KeyPath string
	Timeout                                           time.Duration
}

// Key struct - can be extended, for now is ip:port
type Key struct {
	Key string
}

// Target common struct for connecting to gNMI/gNOI targets
type Target struct {
	Ctxt context.Context
}

// GnmiTarget struct for connecting to a gNMI target
type GnmiTarget struct {
	target      Target
	Destination client.Destination
	Clt         client.Impl
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

// SubscribeMode specifies the mode of a subscription request
type SubscribeMode int

// StreamMode specifies the mode of a streaming request
type StreamMode int

const (
	// Once mode
	Once SubscribeMode = iota
	// Poll mode
	Poll
	// Stream mode
	Stream
)

const (
	// OnChange stream mode
	OnChange StreamMode = iota
	// Sample stream mode
	Sample
	// TargetDefined stream mode
	TargetDefined
)

var targets = make(map[Key]interface{})
