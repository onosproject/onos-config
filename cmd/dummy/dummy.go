// Copyright 2020-present Open Networking Foundation.
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
	"fmt"
	_ "github.com/golang/protobuf/proto"
	"github.com/onosproject/config-models/cmd/empty"
	_ "github.com/onosproject/config-models/modelplugin/stratum-1.0.0/stratum_1_0_0"
	_ "github.com/openconfig/ygot/genutil"
	_ "github.com/openconfig/ygot/ygen"
	_ "github.com/openconfig/ygot/ytypes"
)

func main() {
	fmt.Println("Just a place holder to refer to plugins packages")
	fmt.Printf("Testing %s\n", empty.TestMe())
}
