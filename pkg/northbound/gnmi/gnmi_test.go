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

package gnmi

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/listener"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"os"
	"testing"
)

// TestMain should only contain static data.
// It is run once for all tests - each test is then run on its own thread, so if
// anything is shared the order of it's modification is not deterministic
// Also there can only be one TestMain per package
func TestMain(m *testing.M) {
	//var err error
	os.Exit(m.Run())
}

// setUp should not depend on any global variables
func setUp(broadcast bool) *Server {
	var mgr *manager.Manager
	var server = &Server{}

	cfgStore, err := store.LoadConfigStore("../../../configs/configStore-sample.json")
	if err != nil {
		fmt.Println("Unexpected config store loading error", err)
		os.Exit(-1)
	}
	if cfgStore.Storetype != "config" {
		fmt.Println("Expected Config store to be loaded")
		os.Exit(-1)
	}

	changeStore, err := store.LoadChangeStore("../../../configs/changeStore-sample.json")
	if err != nil {
		fmt.Println("Unexpected change store loading error", err)
		os.Exit(-1)
	}
	if changeStore.Storetype != "change" {
		fmt.Println("Expected Change store to be loaded")
		os.Exit(-1)
	}

	networkStore, err := store.LoadNetworkStore("../../../configs/networkStore-sample.json")
	if err != nil {
		fmt.Println("Unexpected network store loading error", err)
		os.Exit(-1)
	}
	if networkStore.Storetype != "network" {
		fmt.Println("Expected Network store to be loaded")
		os.Exit(-1)
	}

	mgr = manager.GetManager()
	mgr.Dispatcher = listener.NewDispatcher()
	mgr.TopoChannel = make(chan events.TopoEvent)
	go listenToTopoLoading(mgr.TopoChannel)
	mgr.ChangesChannel = make(chan events.ConfigEvent)
	go mgr.Dispatcher.Listen(mgr.ChangesChannel)

	if broadcast {
		go broadcastNotification()
	}

	deviceStore, err := topocache.LoadDeviceStore("../../../configs/deviceStore-sample.json", mgr.TopoChannel)
	if err != nil {
		fmt.Println("Unexpected device store loading error", err)
		os.Exit(-1)
	}
	if deviceStore.Storetype != "device" {
		fmt.Println("Expected device store to be loaded")
		os.Exit(-1)
	}

	mgr.ConfigStore = &cfgStore
	mgr.ChangeStore = &changeStore
	mgr.NetworkStore = networkStore
	mgr.DeviceStore = deviceStore
	fmt.Println("Finished setUp()")
	return server
}

func listenToTopoLoading(deviceChan <-chan events.TopoEvent) {
	for range deviceChan {
		// fmt.Printf("Ignoring event for testing %v\n", deviceConfigEvent)
	}
}
