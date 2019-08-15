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
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	log "k8s.io/klog"
	"sync"
	"time"
)

var (
	// Address is a test server address as "127.0.0.1:port" string
	Address string

	// Opts is a set of gRPC connection options
	Opts []grpc.DialOption
)

// SetUpServer sets up a test manager and a gRPC end-point
// to which it registers the given service.
func SetUpServer(port int16, service Service, waitGroup *sync.WaitGroup) {
	var err error
	_, err = manager.LoadManager(
		"../../../configs/configStore-sample.json",
		"../../../configs/changeStore-sample.json",
		"../../../configs/networkStore-sample.json",
	)
	if err != nil {
		log.Error("Unable to load manager")
	}

	opStateValuesDevice2 := make(map[string]*change.TypedValue)
	opStateValuesDevice2["/cont1a/cont2a/leaf2c"] = change.CreateTypedValueString("test1")
	opStateValuesDevice2["/cont1b-state/leaf2d"] = change.CreateTypedValueUint64(12345)

	manager.GetManager().OperationalStateCache[device.ID("Device2")] = opStateValuesDevice2
	go manager.GetManager().Dispatcher.ListenOperationalState(manager.GetManager().OperationalStateChannel)

	config := NewServerConfig("", "", "")
	config.Port = port
	s := NewServer(config)
	s.AddService(service)

	empty := ""
	Address = fmt.Sprintf(":%d", port)
	Opts, err = certs.HandleCertArgs(&empty, &empty)
	if err != nil {
		log.Error("Error loading cert ", err)
	}
	go func() {
		err := s.Serve(func(started string) {
			waitGroup.Done()
			fmt.Printf("Started %v", started)
		})
		if err != nil {
			log.Error("Unable to serve ", err)
		}
	}()

	go func() {
		time.Sleep(100 * time.Millisecond)
		updatedLeaf2d := make(map[string]string)
		updatedLeaf2d["/cont1a/cont2a/leaf2d"] = "testNotRelevant"
		opStateEventWrong := events.CreateOperationalStateEvent("Device1", updatedLeaf2d)
		manager.GetManager().OperationalStateChannel <- opStateEventWrong

		time.Sleep(100 * time.Millisecond)
		updatedLeaf2c := make(map[string]string)
		updatedLeaf2c["/cont1a/cont2a/leaf2c"] = "test2"
		opStateEvent := events.CreateOperationalStateEvent("Device2", updatedLeaf2c)
		manager.GetManager().OperationalStateChannel <- opStateEvent
	}()
}
