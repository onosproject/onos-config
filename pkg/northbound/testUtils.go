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
	"github.com/golang/mock/gomock"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/certs"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	topodevice "github.com/onosproject/onos-topo/api/device"
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
func SetUpServer(port int16, service Service, waitGroup *sync.WaitGroup) *manager.Manager {
	ctrl := gomock.NewController(nil)
	mgrTest, err := manager.LoadManager(
		mockstore.NewMockLeadershipStore(ctrl),
		mockstore.NewMockMastershipStore(ctrl),
		mockstore.NewMockDeviceChangesStore(ctrl),
		devicestore.NewMockCache(ctrl),
		mockstore.NewMockNetworkChangesStore(ctrl),
		mockstore.NewMockNetworkSnapshotStore(ctrl),
		mockstore.NewMockDeviceSnapshotStore(ctrl))
	if err != nil {
		log.Error("Unable to load manager")
	}

	opStateValuesDevice2 := make(map[string]*devicechange.TypedValue)
	opStateValuesDevice2["/cont1a/cont2a/leaf2c"] = devicechange.NewTypedValueString("test1")
	opStateValuesDevice2["/cont1b-state/leaf2d"] = devicechange.NewTypedValueUint64(12345)

	manager.GetManager().OperationalStateCache[topodevice.ID("Device2")] = opStateValuesDevice2
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
		opStateEventWrong := events.NewOperationalStateEvent("Device1",
			"/cont1a/cont2a/leaf2d",
			devicechange.NewTypedValueString("testNotRelevant"),
			events.EventItemUpdated)
		manager.GetManager().OperationalStateChannel <- opStateEventWrong

		time.Sleep(100 * time.Millisecond)
		opStateEvent := events.NewOperationalStateEvent("Device2",
			"/cont1a/cont2a/leaf2c",
			devicechange.NewTypedValueString("test2"),
			events.EventItemUpdated)
		manager.GetManager().OperationalStateChannel <- opStateEvent
	}()
	return mgrTest
}
