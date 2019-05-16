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
Package main of cmd/diags/changes is a command line client to list Changes.

It connects to the diags gRPC interface of the onos-config-manager and queries
the Change Store
*/
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"log"
	"time"
)

func main() {
	address := flag.String("address", ":5150", "address to which to send requests")
	changeID := flag.String("changeid", "", "a specific change ID")
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

	conn := northbound.Connect(address, opts...)
	defer conn.Close()

	client := proto.NewConfigDiagsClient(conn)

	changesReq := &proto.ChangesRequest{ChangeIds: make([]string, 0)}
	if *changeID != "" {
		changesReq.ChangeIds = append(changesReq.ChangeIds, *changeID)
	}

	stream, err := client.GetChanges(context.Background(), changesReq)
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
			fmt.Printf("%s\t(%s)\t%s\n", in.Id, in.Desc,
				time.Unix(in.Time.Seconds, int64(in.Time.Nanos)).Format(time.RFC3339))
			for _, cv := range in.Changevalues {
				fmt.Printf("\t%s\t%s\t%v\n", cv.Path, cv.Value, cv.Removed)
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
