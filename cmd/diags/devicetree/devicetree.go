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
Package main of cmd/diags/devicetree is a command line client to list devices
and their configuration in tree format.
*/
package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	address := flag.String("address", ":5150", "address to which to send requests")
	deviceName := flag.String("devicename", "", "The hostname and port of a configured device")
	version := flag.Int("version", 0, "verision of the configuration to retrieve - 0 is the latest. -1 is the previous")
	keyPath := flag.String("keyPath", "", "path to client private key")
	certPath := flag.String("certPath", "", "path to client certificate")
	flag.Parse()

	var opts = []grpc.DialOption{}
	if *keyPath != "" && *certPath != "" {
		cert, err := tls.LoadX509KeyPair(*certPath, *keyPath)
		if err != nil {
			log.Println("Error loading certs", err)
		} else {
			log.Println("Loaded key and cert")
		}

		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		log.Println("No key/cert configured")
		opts = append(opts, grpc.WithInsecure())
	}

	if *version > 0 {
		fmt.Println("Version must be 0 (latest) or negative to count backwards")
		os.Exit(-1)
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

	configurations := make([]store.Configuration, 0)

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

			configuration := store.Configuration{
				Name:        store.ConfigName(in.Name),
				Device:      in.Deviceid,
				Created:     time.Unix(in.Updated.Seconds, int64(in.Updated.Nanos)),
				Updated:     time.Unix(in.Updated.Seconds, int64(in.Updated.Nanos)),
				User:        in.User,
				Description: in.Desc,
				Changes:     make([]change.ID, 0),
			}
			for _, cid := range in.ChangeIDs {
				idBytes, _ := base64.StdEncoding.DecodeString(cid)
				configuration.Changes = append(configuration.Changes, change.ID(idBytes))
			}

			configurations = append(configurations, configuration)
		}
	}()
	err = stream.CloseSend()
	if err != nil {
		log.Fatalf("Failed to close: %v", err)
	}
	<-waitc

	changes := make(map[string]*change.Change)

	changesReq := &proto.ChangesRequest{ChangeIds: make([]string, 0)}
	if *deviceName != "" {
		// Only add the changes for a specific device
		for _, ch := range configurations[0].Changes {
			changesReq.ChangeIds = append(changesReq.ChangeIds, base64.StdEncoding.EncodeToString(ch))
		}
	}

	stream2, err := client.GetChanges(context.Background(), changesReq)
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	waitc2 := make(chan struct{})
	go func() {
		for {
			in, err := stream2.Recv()
			if err == io.EOF {
				// read done.
				close(waitc2)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive response : %v", err)
			}
			fmt.Println("Received change", in.Id)
			idBytes, _ := base64.StdEncoding.DecodeString(in.Id)
			changeObj := change.Change{
				ID:          change.ID(idBytes),
				Description: in.Desc,
				Created:     time.Unix(in.Time.Seconds, int64(in.Time.Nanos)),
				Config:      make([]*change.Value, 0),
			}
			for _, cv := range in.Changevalues {
				value, _ := change.CreateChangeValue(cv.Path, cv.Value, cv.Removed)
				changeObj.Config = append(changeObj.Config, value)
			}
			changes[in.Id] = &changeObj
		}
	}()
	err = stream2.CloseSend()
	if err != nil {
		log.Fatalf("Failed to close: %v", err)
	}
	<-waitc2

	for _, configuration := range configurations {
		fmt.Println("Config", configuration.Name, "(Device:", configuration.Device, ")")
		fullDeviceConfigValues := configuration.ExtractFullConfig(changes, *version)
		jsonTree, _ := manager.BuildTree(fullDeviceConfigValues, true)
		fmt.Println(string(jsonTree))
	}
}
