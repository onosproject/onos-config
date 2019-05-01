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

/* Package main is for profiling the key functions
Run the program from anywhere

	time go run github.com/opennetworkinglab/onos-config/profiling

and then run to view the output in a web browser (requires graphviz to be installed)

	go tool pprof -http=localhost:8080 /tmp/cpuProfile.out

 */
package main

import (
	"fmt"
	"github.com/opennetworkinglab/onos-config/store/change"
	"os"
	"runtime/pprof"
	"strconv"
)

func main() {
	cpuFile, err := os.Create("/tmp/cpuProfile.out")
	if err != nil {
		fmt.Println(err)
		return
	}
	pprof.StartCPUProfile(cpuFile)
	defer pprof.StopCPUProfile()


	changeValues := change.ChangeValueCollection{}
	iterations := 50000

	for i := 0; i < iterations; i++ {
		path := fmt.Sprintf("/test%d", i)
		cv, _ := change.CreateChangeValue(path, strconv.Itoa(i), false)
		changeValues = append(changeValues, cv)
	}

	change, err := change.CreateChange(changeValues, "Benchmarked Change")

	err = change.IsValid()
	if err != nil {
		fmt.Errorf("Invalid change %s", err)
	}

	fmt.Println("Finished after ", iterations, "iterations")

}
