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

package certs

import (
	"crypto/tls"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// HandleCertArgs is a common function for clients like admin/net-changes for
// handling certificate args if given, or else loading defaults
func HandleCertArgs(keyPath *string, certPath *string) ([]grpc.DialOption, error) {

	var opts = []grpc.DialOption{}
	var cert tls.Certificate
	var err error
	if *keyPath != Client1Key && *keyPath != "" &&
		*certPath != Client1Crt && *certPath != "" {
		cert, err = tls.LoadX509KeyPair(*certPath, *keyPath)
		if err != nil {
			return nil, err
		}

	} else {
		// Load default Certificates
		cert, err = tls.X509KeyPair([]byte(DefaultClientCrt), []byte(DefaultClientKey))
		if err != nil {
			return nil, err
		}
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	return opts, nil
}
