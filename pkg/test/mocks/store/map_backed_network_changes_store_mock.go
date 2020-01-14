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

package store

import (
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/api/types"
	changetypes "github.com/onosproject/onos-config/api/types/change"
	"github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	networkstore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"sync"
)

//SetUpMapBackedNetworkChangesStore : creates a map backed store for the given mock
func SetUpMapBackedNetworkChangesStore(mockNetworkChangesStore *MockNetworkChangesStore) {
	mu := &sync.RWMutex{}
	networkChangesList := make([]*networkchange.NetworkChange, 0)
	mockNetworkChangesStore.EXPECT().Create(gomock.Any()).DoAndReturn(
		func(networkChange *networkchange.NetworkChange) error {
			mu.Lock()
			networkChangesList = append(networkChangesList, networkChange)
			mu.Unlock()
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Update(gomock.Any()).DoAndReturn(
		func(networkChange *networkchange.NetworkChange) error {
			mu.Lock()
			networkChangesList = append(networkChangesList, networkChange)
			mu.Unlock()
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(id networkchange.ID) (*networkchange.NetworkChange, error) {
			var found *networkchange.NetworkChange
			mu.RLock()
			for _, networkChange := range networkChangesList {
				if networkChange.ID == id {
					found = networkChange
				}
			}
			mu.RUnlock()
			if found != nil {
				return found, nil
			}
			return nil, nil
		}).AnyTimes()

	mockNetworkChangesStore.EXPECT().List(gomock.Any()).DoAndReturn(
		func(c chan<- *networkchange.NetworkChange) error {
			go func() {
				mu.RLock()
				defer mu.RUnlock()
				for _, networkChange := range networkChangesList {
					c <- networkChange
				}
				close(c)
			}()
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(c chan<- stream.Event, o ...networkstore.WatchOption) (stream.Context, error) {
			go func() {
				mu.RLock()
				defer mu.RUnlock()
				change := networkChangesList[len(networkChangesList)-1]
				if change == nil {
					close(c)
					return
				}

				change1 := *change
				if change1.Status.Phase == changetypes.Phase_ROLLBACK ||
					change1.Status.State == changetypes.State_PENDING {
					change1.Status.State = changetypes.State_COMPLETE
				}
				event := stream.Event{
					Type:   changetypes.ListResponseType_LISTNONE,
					Object: &change1,
				}
				c <- event

				change2 := change1
				refs := make([]*networkchange.DeviceChangeRef, len(change2.Changes))
				for i, change := range change2.Changes {
					refs[i] = &networkchange.DeviceChangeRef{
						DeviceChangeID: device.NewID(types.ID(change2.ID), change.DeviceID, change.DeviceVersion),
					}
				}
				change2.Refs = refs
				event = stream.Event{
					Type:   changetypes.ListResponseType_LISTNONE,
					Object: &change2,
				}
				c <- event
				close(c)
			}()
			return stream.NewContext(func() {}), nil
		}).AnyTimes()
}
