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

/*/
Package main of cmd/admin/device is a command line utility that allows adding
and removing devices from the device inventory.
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/northbound"
	devices "github.com/onosproject/onos-config/pkg/northbound/proto"
	"io"
	"log"
)

func main() {
	address := flag.String("address", ":5150", "address to which to send requests e.g. localhost:5150")
	keyPath := flag.String("keyPath", certs.Client1Key, "path to client private key")
	certPath := flag.String("certPath", certs.Client1Crt, "path to client certificate")

	addDevice := flag.String("addDevice", "", "protobuf encoding of device to add")
	removeID := flag.String("removeID", "", "id of device to remove")

	flag.Parse()

	opts, err := certs.HandleCertArgs(keyPath, certPath)
	if err != nil {
		log.Fatal("Error loading cert", err)
	}

	conn := northbound.Connect(address, opts...)
	defer conn.Close()

	client := devices.NewDeviceInventoryServiceClient(conn)

	if addDevice != nil && len(*addDevice) > 0 {
		deviceInfo := &devices.DeviceInfo{}
		if err := proto.UnmarshalText(*addDevice, deviceInfo); err == nil {
			fmt.Printf("Adding device %s\n", deviceInfo.Id)
			_, err = client.AddDevice(context.Background(), deviceInfo)
		} else {
			log.Fatalf("Unable to parse device info from %s : %v", *addDevice, err)
		}

	} else if removeID != nil && len(*removeID) > 0 {
		fmt.Printf("Removing device %s\n", *removeID)
		_, err = client.RemoveDevice(context.Background(), &devices.DeviceInfo{Id: *removeID})

	} else {
		stream, err := client.GetDevices(context.Background(), &devices.GetDevicesRequest{})
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
				fmt.Printf("%s: %s (%s)\n", in.Id, in.Address, in.Version)
			}
		}()
		err = stream.CloseSend()
		if err != nil {
			log.Fatalf("Failed to close: %v", err)
		}
		<-waitc

	}

	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
}
