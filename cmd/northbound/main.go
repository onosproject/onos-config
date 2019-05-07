// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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
	"fmt"

	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
)

func main() {
	cfg := &gnmi.ServerConfig{
		Port:     5150,
		Insecure: true,
	}

	serv, err := gnmi.NewServer(cfg)
	if err != nil {
		fmt.Println("Can't start server ", err)
	}

	serv.Serve()
}
