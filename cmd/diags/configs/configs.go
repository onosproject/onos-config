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

/*
Package main of cmd/diags/configs is a command line client to list Configs.

It connects to the diags gRPC interface of the onos-config and queries
the Config Store
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"io"
	"log"
	"time"
)

func main() {
	address := flag.String("address", ":5150", "address to which to send requests e.g. localhost:5150")
	deviceName := flag.String("devicename", "", "hostname and port of a configured device")
	keyPath := flag.String("keyPath", certs.Client1Key, "path to client private key")
	certPath := flag.String("certPath", certs.Client1Crt, "path to client certificate")
	flag.Parse()

	opts, err := certs.HandleCertArgs(keyPath, certPath)
	if err != nil {
		log.Fatal("Error loading cert", err)
	}

	conn := northbound.Connect(address, opts...)
	defer conn.Close()

	client := proto.NewConfigDiagsClient(conn)

	configReq := &proto.ConfigRequest{DeviceIds: make([]string, 0)}
	if *deviceName != "" {
		configReq.DeviceIds = append(configReq.DeviceIds, *deviceName)
	}
	stream, err := client.GetConfigurations(context.Background(), configReq)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	waitc := make(chan struct{})
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				// read done.
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive response : %v", err)
			}
			fmt.Printf("%s\t(%s)\t%s\t%s\n", in.Name, in.Deviceid, in.Desc,
				time.Unix(in.Updated.Seconds, int64(in.Updated.Nanos)).Format(time.RFC3339))
			for _, cid := range in.ChangeIDs {
				fmt.Printf("\t%s", cid)
			}
			fmt.Println()
		}
	}()
	err = stream.CloseSend()
	if err != nil {
		log.Fatalf("Failed to close: %v", err)
	}
	<-waitc
}
