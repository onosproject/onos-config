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
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/manager"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
	"os"
	"sync"
	"testing"
)

// TestMain should only contain static data.
// It is run once for all tests - each test is then run on its own thread, so if
// anything is shared the order of it's modification is not deterministic
// Also there can only be one TestMain per package
func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	os.Exit(m.Run())
}

// setUp should not depend on any global variables
func setUp(t *testing.T) (*Server, *manager.Manager, *MockStore, *mockstore.MockNetworkChangesStore) {
	var server = &Server{}

	ctrl := gomock.NewController(t)
	mockDeviceChangesStore := mockstore.NewMockDeviceChangesStore(ctrl)
	mockNetworkChangesStore := mockstore.NewMockNetworkChangesStore(ctrl)

	mgr, err := manager.LoadManager(
		"../../../configs/configStore-sample.json",
		"../../../configs/changeStore-sample.json",
		"../../../configs/networkStore-sample.json",
		mockDeviceChangesStore,
		mockNetworkChangesStore)

	if err != nil {
		log.Error("Expected manager to be loaded ", err)
		os.Exit(-1)
	}
	mockDeviceStore := NewMockStore(ctrl)
	mgr.DeviceStore = mockDeviceStore
	mgr.DeviceChangesStore = mockDeviceChangesStore
	mgr.NetworkChangesStore = mockNetworkChangesStore

	log.Infof("Dispatcher pointer %p", &mgr.Dispatcher)
	go listenToTopoLoading(mgr.TopoChannel)
	go mgr.Dispatcher.Listen(mgr.ChangesChannel)

	log.Info("Finished setUp()")
	//TODO return all mock stores here. it needs to be passed because otherwise we can't do expect calls
	return server, mgr, mockDeviceStore, mockNetworkChangesStore
}

func tearDown(mgr *manager.Manager, wg *sync.WaitGroup) {
	// `wg.Wait` blocks until `wg.Done` is called the same number of times
	// as the amount of tasks we have (in this case, 1 time)
	wg.Wait()

	mgr.Dispatcher = &dispatcher.Dispatcher{}
	log.Infof("Dispatcher Teardown %p", mgr.Dispatcher)

}

func listenToTopoLoading(deviceChan <-chan *devicepb.ListResponse) {
	for deviceConfigEvent := range deviceChan {
		log.Info("Ignoring event for testing ", deviceConfigEvent)
	}
}
