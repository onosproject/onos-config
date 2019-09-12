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
Package main of cmd/profiling is for profiling some key functions.

Run the program from anywhere

	time go run github.com/opennetworkinglab/onos-config/profiling

and then run to view the output in a web browser (requires graphviz to be installed)

	go tool pprof -http=localhost:8080 /tmp/cpuProfile.out

*/
package main

import (
	"fmt"
	log "k8s.io/klog"
	"os"
	"runtime/pprof"

	"github.com/onosproject/onos-config/pkg/store/change"
)

func main() {
	cpuFile, err := os.Create("/tmp/cpuProfile.out")
	if err != nil {
		fmt.Println(err)
		return
	}
	_ = pprof.StartCPUProfile(cpuFile)
	defer pprof.StopCPUProfile()

	changeValues := change.ValueCollections{}
	iterations := 50000

	for i := 0; i < iterations; i++ {
		path := fmt.Sprintf("/test%d", i)
		cv, _ := change.CreateChangeValue(path, (*change.TypedValue)(change.CreateTypedValueInt64(i)), false)
		changeValues = append(changeValues, cv)
	}

	newChange, err := change.CreateChange(changeValues, "Benchmarked Change")
	if err != nil {
		log.Error("Cannot create a newChange object from ChangeValues ", err)
	}

	err = newChange.IsValid()
	if err != nil {
		log.Error("Invalid newChange ", err)
	}

	log.Info("Finished after ", iterations, " iterations")

}
