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
Package main of cmd/admin/rollback is a command line utility that enables a named
network change to be rolled back.

If no name is given that last network change is rolled back
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
	"log"
)

func main() {
	address := flag.String("address", ":5150", "address to which to send requests")
	changeName := flag.String("changename", "", "")
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
			Certificates: []tls.Certificate{cert},
			InsecureSkipVerify: true,
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		log.Println("No key/cert configured")
		opts = append(opts, grpc.WithInsecure())
	}

	conn := northbound.Connect(address, opts...)
	defer conn.Close()

	client := proto.NewAdminServiceClient(conn)

	rbResp, err := client.RollbackNetworkChange(
		context.Background(), &proto.RollbackRequest{Name: *changeName})
	if err != nil {
		log.Fatalf("Failed to send request: %v", err)
	}
	fmt.Println("Rollback success", rbResp.Message)

}
