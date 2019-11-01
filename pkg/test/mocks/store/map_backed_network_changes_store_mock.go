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
	networkstore "github.com/onosproject/onos-config/pkg/store/change/network"
	"github.com/onosproject/onos-config/pkg/store/stream"
	changetypes "github.com/onosproject/onos-config/pkg/types/change"
	networkchangetypes "github.com/onosproject/onos-config/pkg/types/change/network"
)

//SetUpMapBackedNetworkChangesStore : creates a map backed store for the given mock
func SetUpMapBackedNetworkChangesStore(mockNetworkChangesStore MockNetworkChangesStore) {
	networkChangesList := make([]*networkchangetypes.NetworkChange, 0)
	mockNetworkChangesStore.EXPECT().Create(gomock.Any()).DoAndReturn(
		func(networkChange *networkchangetypes.NetworkChange) error {
			networkChangesList = append(networkChangesList, networkChange)
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Update(gomock.Any()).DoAndReturn(
		func(networkChange *networkchangetypes.NetworkChange) error {
			networkChangesList = append(networkChangesList, networkChange)
			return nil
		}).AnyTimes()
	mockNetworkChangesStore.EXPECT().Get(gomock.Any()).DoAndReturn(
		func(id networkchangetypes.ID) (*networkchangetypes.NetworkChange, error) {
			var found *networkchangetypes.NetworkChange
			for _, networkChange := range networkChangesList {
				if networkChange.ID == id {
					found = networkChange
				}
			}
			if found != nil {
				return found, nil
			}
			return nil, nil
		}).AnyTimes()

	mockNetworkChangesStore.EXPECT().List(gomock.Any()).DoAndReturn(
		func(c chan<- *networkchangetypes.NetworkChange) error {
			go func() {
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
				lastChange := networkChangesList[len(networkChangesList)-1]
				if lastChange.Status.Phase == changetypes.Phase_ROLLBACK {
					lastChange.Status.State = changetypes.State_COMPLETE
				}
				event := stream.Event{
					Type:   "",
					Object: lastChange,
				}
				c <- event
				close(c)
			}()
			return stream.NewContext(func() {}), nil
		}).AnyTimes()
}
