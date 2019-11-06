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
	networkchangestore "github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	mockstore "github.com/onosproject/onos-config/pkg/test/mocks/store"
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/types/change/network"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
	devicetype "github.com/onosproject/onos-config/pkg/types/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"os"
	"sync"
	"testing"
	"time"
)

// AllMocks is a place to hold references to mock stores and caches. It can't be put into test/mocks/store
// because of circular dependencies.
type AllMocks struct {
	MockStores      *mockstore.MockStores
	MockDeviceCache *devicestore.MockCache
}

// TestMain should only contain static data.
// It is run once for all tests - each test is then run on its own thread, so if
// anything is shared the order of it's modification is not deterministic
// Also there can only be one TestMain per package
func TestMain(m *testing.M) {
	log.SetOutput(os.Stdout)
	os.Exit(m.Run())
}

type MockModelPlugin struct {
	schemaFn func() (map[string]*yang.Entry, error)
}

func (m MockModelPlugin) ModelData() (string, string, []*gnmi.ModelData, string) {
	panic("implement me")
}

func (m MockModelPlugin) UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error) {
	return nil, nil
}

func (m MockModelPlugin) Validate(*ygot.ValidatedGoStruct, ...ygot.ValidationOption) error {
	return nil
}

func (m MockModelPlugin) Schema() (map[string]*yang.Entry, error) {
	return m.schemaFn()
}

func (m MockModelPlugin) GetStateMode() int {
	panic("implement me")
}

func setUpWatchMock(mocks *AllMocks) {
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

	config1Value01, _ := devicechangetypes.NewChangeValue("/cont1a/cont2a/leaf4a", devicechangetypes.NewTypedValueUint64(14), false)
	config1Value02, _ := devicechangetypes.NewChangeValue("/cont1a/cont2a/leaf2a", devicechangetypes.NewTypedValueEmpty(), true)

	deviceChange := devicechangetypes.DeviceChange{
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
		NetworkChange: devicechangetypes.NetworkChangeRef{
			ID:    "",
			Index: 0,
		},
		Change: &devicechangetypes.Change{
			DeviceID:      "",
			DeviceVersion: "",
			Values: []*devicechangetypes.ChangeValue{
				config1Value01, config1Value02,
			},
		},
	}

	mocks.MockStores.NetworkChangesStore.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(c chan<- stream.Event, opts ...networkchangestore.WatchOption) (stream.Context, error) {
			go func() {
				c <- stream.Event{Object: &watchChange}
				close(c)
			}()
			return stream.NewContext(func() {

			}), nil
		}).AnyTimes()

	mocks.MockStores.DeviceChangesStore.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(deviceId devicetype.VersionedID, c chan<- stream.Event, opts ...networkchangestore.WatchOption) (stream.Context, error) {
			go func() {
				c <- stream.Event{Object: &deviceChange}
				close(c)
			}()
			return stream.NewContext(func() {

			}), nil
		}).AnyTimes()
}

func setUpListMock(mocks *AllMocks) {
	mocks.MockStores.DeviceChangesStore.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(device devicetype.VersionedID, c chan<- *devicechangetypes.DeviceChange) (stream.Context, error) {
			go func() {
				close(c)
			}()
			return stream.NewContext(func() {

			}), nil
		}).AnyTimes()
}

func setUpChangesMock(mocks *AllMocks) {
	configValue01, _ := devicechangetypes.NewChangeValue("/cont1a/cont2a/leaf2a", devicechangetypes.NewTypedValueUint64(13), false)

	change1 := devicechangetypes.Change{
		Values: []*devicechangetypes.ChangeValue{
			configValue01,
		},
		DeviceID:      devicetype.ID("Device1"),
		DeviceVersion: "1.0.0",
	}
	deviceChange1 := &devicechangetypes.DeviceChange{
		Change: &change1,
		ID:     "Change",
		Status: changetypes.Status{
			Phase:   changetypes.Phase_CHANGE,
			State:   changetypes.State_COMPLETE,
			Reason:  0,
			Message: "",
		},
	}
	mocks.MockStores.DeviceChangesStore.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(device devicetype.VersionedID, c chan<- *devicechangetypes.DeviceChange) (stream.Context, error) {
			go func() {
				c <- deviceChange1
				close(c)
			}()
			return stream.NewContext(func() {

			}), nil
		}).AnyTimes()
}

// setUp should not depend on any global variables
func setUp(t *testing.T) (*Server, *manager.Manager, *AllMocks) {
	var server = &Server{}
	var allMocks AllMocks

	ctrl := gomock.NewController(t)
	mockStores := &mockstore.MockStores{
		DeviceStore:          mockstore.NewMockDeviceStore(ctrl),
		NetworkChangesStore:  mockstore.NewMockNetworkChangesStore(ctrl),
		DeviceChangesStore:   mockstore.NewMockDeviceChangesStore(ctrl),
		NetworkSnapshotStore: mockstore.NewMockNetworkSnapshotStore(ctrl),
		DeviceSnapshotStore:  mockstore.NewMockDeviceSnapshotStore(ctrl),
		LeadershipStore:      mockstore.NewMockLeadershipStore(ctrl),
		MastershipStore:      mockstore.NewMockMastershipStore(ctrl),
	}
	deviceCache := devicestore.NewMockCache(ctrl)
	allMocks.MockStores = mockStores
	allMocks.MockDeviceCache = deviceCache

	mgr, err := manager.LoadManager(
		mockStores.LeadershipStore,
		mockStores.MastershipStore,
		mockStores.DeviceChangesStore,
		deviceCache,
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
	//go mgr.Dispatcher.Listen(mgr.ChangesChannel)

	setUpWatchMock(&allMocks)
	log.Info("Finished setUp()")

	return server, mgr, &allMocks
}

func setUpBaseNetworkStore(store *mockstore.MockNetworkChangesStore) {
	mockstore.SetUpMapBackedNetworkChangesStore(store)
	now := time.Now()
	config1Value01, _ := devicechangetypes.NewChangeValue("/cont1a/cont2a/leaf4a", devicechangetypes.NewTypedValueUint64(14), false)
	config1Value02, _ := devicechangetypes.NewChangeValue("/cont1a/cont2a/leaf2a", devicechangetypes.NewTypedValueEmpty(), true)
	change1 := devicechangetypes.Change{
		Values: []*devicechangetypes.ChangeValue{
			config1Value01, config1Value02},
		DeviceID:      device1,
		DeviceVersion: deviceVersion1,
	}

	networkChange1 := &networkchangetypes.NetworkChange{
		ID:      networkChange1,
		Changes: []*devicechangetypes.Change{&change1},
		Updated: now,
		Created: now,
		Status:  changetypes.Status{State: changetypes.State_COMPLETE},
	}
	_ = store.Create(networkChange1)
}

func setUpBaseDevices(mockStores *mockstore.MockStores, deviceCache *devicestore.MockCache) {
	deviceCache.EXPECT().GetDevicesByID(gomock.Any()).Return([]*devicestore.Info{
		{
			DeviceID: "Device1",
			Version:  "1.0.0",
			Type:     "TestDevice",
		},
	}).AnyTimes()

	mockStores.DeviceStore.EXPECT().Get(gomock.Any()).Return(nil, status.Error(codes.NotFound, "device not found")).AnyTimes()
	mockStores.DeviceStore.EXPECT().List(gomock.Any()).DoAndReturn(
		func(c chan<- *devicetopo.Device) (stream.Context, error) {
			go func() {
				c <- &devicetopo.Device{
					ID:      "Device1 (1.0.0)",
					Version: "1.0.0",
				}
				c <- &devicetopo.Device{
					ID:      "Device2 (2.0.0)",
					Version: "2.0.0",
				}
				c <- &devicetopo.Device{
					ID:      "Device3",
					Version: "1.0.0",
				}
				close(c)
			}()
			return stream.NewContext(func() {
			}), nil
		}).AnyTimes()
}

func setUpForGetSetTests(t *testing.T) (*Server, []*gnmi.Path, []*gnmi.Update, []*gnmi.Update) {
	server, mgr, allMocks := setUp(t)
	allMocks.MockStores.NetworkChangesStore = mockstore.NewMockNetworkChangesStore(gomock.NewController(t))
	mgr.NetworkChangesStore = allMocks.MockStores.NetworkChangesStore
	setUpBaseNetworkStore(allMocks.MockStores.NetworkChangesStore)
	setUpBaseDevices(allMocks.MockStores, allMocks.MockDeviceCache)
	var deletePaths = make([]*gnmi.Path, 0)
	var replacedPaths = make([]*gnmi.Update, 0)
	var updatedPaths = make([]*gnmi.Update, 0)
	return server, deletePaths, replacedPaths, updatedPaths
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
