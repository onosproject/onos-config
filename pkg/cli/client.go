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

package cli

import (
	"crypto/tls"
	"github.com/onosproject/onos-config/pkg/certs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defaultAddress = "onos-config:5150"
)

// getConnection returns a gRPC client connection to the config service
func getConnection() *grpc.ClientConn {
	address := getConfigOrDefault("address", defaultAddress).(string)
	certPath := getConfigString("tls.certPath")
	keyPath := getConfigString("tls.keyPath")
	var opts []grpc.DialOption
	if certPath != "" && keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			ExitWithError(ExitBadArgs, err)
		}
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
			})),
		}
	} else {
		// Load default Certificates
		cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
		if err != nil {
			ExitWithError(ExitBadArgs, err)
		}
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
			})),
		}
	}

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		ExitWithError(ExitBadConnection, err)
	}
	return conn
}
