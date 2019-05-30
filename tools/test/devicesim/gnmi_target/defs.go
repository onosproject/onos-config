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

package main

import (
	"flag"

	"github.com/onosproject/onos-config/tools/test/devicesim/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

var (
	bindAddr   = flag.String("bind_address", ":10161", "Bind to address:port or just :port")
	configFile = flag.String("config", "", "IETF JSON file for target startup config")
)

type server struct {
	*gnmi.Server
	Model        *gnmi.Model
	configStruct ygot.ValidatedGoStruct
}

type streamClient struct {
	target  string
	sr      *pb.SubscribeRequest
	stream  pb.GNMI_SubscribeServer
	errChan chan<- error
}
