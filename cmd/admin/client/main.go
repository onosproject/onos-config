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
	"context"
	"crypto/tls"
	"fmt"
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/northbound/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"log"
)

func main() {
	cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
	if err != nil {
		log.Println("Error loading default certs")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	// Set up a connection to the server.
	conn, err := grpc.Dial(":5150", grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
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

	stream, err := client.Shell(context.Background())
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
			fmt.Println(in.Str)
		}
	}()
	if err := stream.Send(&proto.Line{Str: "Hi there bozo..."}); err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	stream.CloseSend()
	<-waitc
}
