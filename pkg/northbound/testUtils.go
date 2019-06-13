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

package northbound

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/manager"
	"google.golang.org/grpc"
	"log"
)

var (
	// Address is a test server address as "127.0.0.1:port" string
	Address string

	// Opts is a set of gRPC connection options
	Opts []grpc.DialOption
)

// SetUpServer sets up a test manager and a gRPC end-point
// to which it registers the given service.
func SetUpServer(port int16, service Service) {
	var err error
	_, err = manager.LoadManager(
		"../../../configs/configStore-sample.json",
		"../../../configs/changeStore-sample.json",
		"../../../configs/deviceStore-sample.json",
		"../../../configs/networkStore-sample.json",
	)
	if err != nil {
		log.Fatal("Unable to load manager")
	}

	config := NewServerConfig("", "", "")
	config.Port = port
	s := NewServer(config)
	s.AddService(service)
	go func() {
		err := s.Serve()
		if err != nil {
			log.Fatal("Unable to serve", err)
		}
	}()

	empty := ""
	Address = fmt.Sprintf("127.0.0.1:%d", port)
	Opts, err = certs.HandleCertArgs(&empty, &empty)
	if err != nil {
		log.Fatal("Error loading cert", err)
	}
}
