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
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/types/change/network"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
	"os"
	"sync"
	"testing"
	"time"
)

// TestMain should only contain static data.
// It is run once for all tests - each test is then run on its own thread, so if
// anything is shared the order of it's modification is not deterministic
// Also there can only be one TestMain per package
func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	os.Exit(m.Run())
}

type MockStores struct {
	DeviceStore          *mockstore.MockDeviceStore
	NetworkChangesStore  *mockstore.MockNetworkChangesStore
	DeviceChangesStore   *mockstore.MockDeviceChangesStore
	NetworkSnapshotStore *mockstore.MockNetworkSnapshotStore
	DeviceSnapshotStore  *mockstore.MockDeviceSnapshotStore
	LeadershipStore      *mockstore.MockLeadershipStore
	MastershipStore      *mockstore.MockMastershipStore
}

func setUpWatchMock(mockStores *MockStores) {
	now := time.Now()
	watchChange := network.NetworkChange{
		ID:       "",
		Index:    0,
		Revision: 0,
		Status: changetypes.Status{
			Phase:   0,
			State:   changetypes.State_COMPLETE,
			Reason:  0,
			Message: "",
		},
		Created: now,
		Updated: now,
		Changes: nil,
		Refs:    nil,
		Deleted: false,
	}
	mockStores.NetworkChangesStore.EXPECT().Watch(gomock.Any()).DoAndReturn(
		func(c chan<- *network.NetworkChange) error {
			go func() {
				c <- &watchChange
				close(c)
			}()
			return nil
		}).AnyTimes()
}

// setUp should not depend on any global variables
func setUp(t *testing.T) (*Server, *manager.Manager, *MockStores) {
	var server = &Server{}

	ctrl := gomock.NewController(t)
	mockStores := MockStores{
		DeviceStore:          mockstore.NewMockDeviceStore(ctrl),
		NetworkChangesStore:  mockstore.NewMockNetworkChangesStore(ctrl),
		DeviceChangesStore:   mockstore.NewMockDeviceChangesStore(ctrl),
		NetworkSnapshotStore: mockstore.NewMockNetworkSnapshotStore(ctrl),
		DeviceSnapshotStore:  mockstore.NewMockDeviceSnapshotStore(ctrl),
		LeadershipStore:      mockstore.NewMockLeadershipStore(ctrl),
		MastershipStore:      mockstore.NewMockMastershipStore(ctrl),
	}

	mgr, err := manager.LoadManager(
		"../../../configs/configStore-sample.json",
		"../../../configs/changeStore-sample.json",
		"../../../configs/networkStore-sample.json",
		mockStores.LeadershipStore,
		mockStores.MastershipStore,
		mockStores.DeviceChangesStore,
		mockStores.NetworkChangesStore,
		mockStores.NetworkSnapshotStore,
		mockStores.DeviceSnapshotStore)

	if err != nil {
		log.Error("Expected manager to be loaded ", err)
		os.Exit(-1)
	}
	mgr.DeviceStore = mockStores.DeviceStore
	mgr.DeviceChangesStore = mockStores.DeviceChangesStore
	mgr.NetworkChangesStore = mockStores.NetworkChangesStore

	log.Infof("Dispatcher pointer %p", &mgr.Dispatcher)
	go listenToTopoLoading(mgr.TopoChannel)
	go mgr.Dispatcher.Listen(mgr.ChangesChannel)

	mockStores.DeviceChangesStore.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(device devicetopo.ID, c chan<- *devicechangetypes.DeviceChange) error {
			close(c)
			return nil
		}).AnyTimes()

	setUpWatchMock(&mockStores)
	log.Info("Finished setUp()")

	return server, mgr, &mockStores
}

func tearDown(mgr *manager.Manager, wg *sync.WaitGroup) {
	// `wg.Wait` blocks until `wg.Done` is called the same number of times
	// as the amount of tasks we have (in this case, 1 time)
	wg.Wait()

	mgr.Dispatcher = &dispatcher.Dispatcher{}
	log.Infof("Dispatcher Teardown %p", mgr.Dispatcher)

}

func listenToTopoLoading(deviceChan <-chan *devicetopo.ListResponse) {
	for deviceConfigEvent := range deviceChan {
		log.Info("Ignoring event for testing ", deviceConfigEvent)
	}
}
