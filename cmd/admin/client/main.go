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

package main

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"google.golang.org/grpc"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(":5150", grpc.WithInsecure())
	if err != nil {
		fmt.Println("Can't connect", err)
	}
	defer conn.Close()

	client := proto.NewNorthboundClient(conn)
	_, err = client.SayHello(context.TODO(), &proto.HelloWorldRequest{})
	if err != nil {
		fmt.Println("Request failed", err)
	} else {
		fmt.Println("Request succeeded :-)")
	}
}
