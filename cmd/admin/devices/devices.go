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
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"io"
	"log"
)

func main() {
	address := flag.String("address", ":5150", "address to which to send requests e.g. localhost:5150")
	keyPath := flag.String("keyPath", certs.Client1Key, "path to client private key")
	certPath := flag.String("certPath", certs.Client1Crt, "path to client certificate")

	deviceAddress := flag.String("deviceAddress", "", "device address:port")
	deviceTarget := flag.String("deviceTarget", "", "device target name")
	deviceUser := flag.String("deviceUser", "", "device login user")
	devicePassword := flag.String("devicePassword", "", "device login password")
	deviceCaPath := flag.String("deviceCaPath", "", "device CA certificate path")
	deviceCertPath := flag.String("deviceCertPath", "", "device certificate path")
	deviceKeyPath := flag.String("deviceKeyPath", "", "device key path")

	devicePlain := flag.Bool("devicePlain", false, "device communicates in plain text")
	deviceInsecure := flag.Bool("deviceInsecure", false, "device communicates insecurely")
	deviceTimeout := flag.Int64("deviceTimeout", 10000000000, "device communication timeout in nanos")

	flag.Parse()

	opts, err := certs.HandleCertArgs(keyPath, certPath)
	if err != nil {
		log.Fatal("Error loading cert", err)
	}

	conn := northbound.Connect(address, opts...)
	defer conn.Close()

	client := proto.NewDeviceInventoryServiceClient(conn)

	operation := flag.Arg(0)
	id := flag.Arg(1)

	if operation == "add" {
		_, err = client.AddDevice(context.Background(), &proto.DeviceInfo{
			Id:       id,
			Address:  *deviceAddress,
			Target:   *deviceTarget,
			User:     *deviceUser,
			Password: *devicePassword,
			CaPath:   *deviceCaPath,
			CertPath: *deviceCertPath,
			KeyPath:  *deviceKeyPath,
			Plain:    *devicePlain,
			Insecure: *deviceInsecure,
			Timeout:  *deviceTimeout,
		})
	} else if operation == "remove" {
		fmt.Printf("Removing device %s\n", id)
		_, err = client.RemoveDevice(context.Background(), &proto.DeviceInfo{Id: id})
	} else {
		stream, err := client.GetDevices(context.Background(), &proto.GetDevicesRequest{})
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
				fmt.Printf("%s: %s\n", in.Id, in.Address)
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
