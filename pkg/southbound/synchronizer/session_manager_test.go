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

package synchronizer

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	dispatcherpkg "github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	modelregistrypkg "github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store/stream"
	storemock "github.com/onosproject/onos-config/pkg/test/mocks/store"
	"gotest.tools/assert"
)

func createSessionManager(t *testing.T) *SessionManager {
	dispatcher := dispatcherpkg.NewDispatcher()
	models := new(modelregistrypkg.ModelRegistry)
	opstateCache := make(map[topodevice.ID]devicechange.TypedValueMap)
	opStateCacheLock := &sync.RWMutex{}
	ctrl := gomock.NewController(t)
	deviceChangeStore := storemock.NewMockDeviceChangesStore(ctrl)
	mastershipStore := storemock.NewMockMastershipStore(ctrl)
	deviceStore := storemock.NewMockDeviceStore(ctrl)

	deviceChangeStore.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(deviceID devicetype.VersionedID, c chan<- *devicechange.DeviceChange) (stream.Context, error) {
			ctx := stream.NewContext(func() {})
			return ctx, errors.NewNotFound("no Configuration found")
		}).AnyTimes()

	deviceStore.EXPECT().Watch(gomock.Any()).AnyTimes()
	mastershipStore.EXPECT().Watch(gomock.Any(), gomock.Any()).AnyTimes()
	mastershipStore.EXPECT().GetMastership(gomock.Any()).AnyTimes()
	mastershipStore.EXPECT().NodeID().AnyTimes()
	mastershipStore.EXPECT().Close().AnyTimes()

	topoChan := make(chan *topodevice.ListResponse)
	opstateChan := make(chan events.OperationalStateEvent)

	sessionManager, err := NewSessionManager(
		WithTopoChannel(topoChan),
		WithOpStateChannel(opstateChan),
		WithDispatcher(dispatcher),
		WithModelRegistry(models),
		WithOperationalStateCache(opstateCache),
		WithNewTargetFn(southbound.NewTarget),
		WithOperationalStateCacheLock(opStateCacheLock),
		WithDeviceChangeStore(deviceChangeStore),
		WithMastershipStore(mastershipStore),
		WithDeviceStore(deviceStore),
		WithSessions(make(map[topodevice.ID]*Session)),
	)

	assert.NilError(t, err)
	return sessionManager

}

/**
 * Check device is added as a synchronizer correctly, times out on no gRPC device
 * and then un-does everything
 */
func TestSessionManager(t *testing.T) {
	// TODO Fix this unit test or replace with a new one
	sessionManager := createSessionManager(t)
	_ = sessionManager.Start()
	t.Skip()

	/*timeout := time.Millisecond * 500
	device1NameStr := "factoryTd"
	device1 := topodevice.Device{
		ID:          topodevice.ID(device1NameStr),
		Revision:    0,
		Address:     "1.2.3.4:11161",
		Target:      "",
		Version:     "1.0.0",
		Timeout:     &timeout,
		Credentials: topodevice.Credentials{},
		TLS:         topodevice.TlsConfig{},
		Type:        "TestDevice",
		Role:        "leaf",
		Attributes:  make(map[string]string),
	}
	topoEvent := topodevice.ListResponse{
		Type:   topodevice.ListResponseADDED,
		Device: &device1,
	}

	sessionManager.topoChannel <- &topoEvent

	for resp := range sessionManager.southboundErrorChan {
		assert.Error(t, resp.Error(),
			"topo update event ignored type:UPDATED device:<id:\"factoryTd\" address:\"1.2.3.4:11161\" version:\"1.0.0\" timeout:<nanos:500000000 > credentials:<> tls:<> type:\"TestDevice\" role:\"spine\" > ", "after topo update")
		break
	}

	// Wait for gRPC connection to timeout
	time.Sleep(time.Millisecond * 1000) // Give it a moment for the event to take effect and for timeout to happen
	sessionManager.operationalStateCacheLock.RLock()
	opStateCacheUpdated, ok := sessionManager.operationalStateCache[device1.ID]
	sessionManager.operationalStateCacheLock.RUnlock()
	assert.Assert(t, ok, "Op state cache entry created")
	assert.Equal(t, len(opStateCacheUpdated), 0)

	time.Sleep(1 * time.Second)

	// Device removed from topo
	//device1.Attributes["t1"] = "test"
	device1Update := topodevice.Device{
		ID:          topodevice.ID(device1NameStr),
		Revision:    0,
		Address:     "1.2.3.4:11161",
		Target:      "",
		Version:     "1.0.0",
		Timeout:     &timeout,
		Credentials: topodevice.Credentials{},
		TLS:         topodevice.TlsConfig{},
		Type:        "TestDevice",
		Role:        "spine", // Role is changed - will be ignored
		Attributes:  make(map[string]string),
	}
	topoEventUpdated := topodevice.ListResponse{
		Type:   topodevice.ListResponseUPDATED,
		Device: &device1Update,
	}
	sessionManager.topoChannel <- &topoEventUpdated

	for resp := range sessionManager.southboundErrorChan {
		assert.Error(t, resp.Error(),
			"topo update event ignored type:UPDATED device:<id:\"factoryTd\" address:\"1.2.3.4:11161\" version:\"1.0.0\" timeout:<nanos:500000000 > credentials:<> tls:<> type:\"TestDevice\" role:\"spine\" > ", "after topo update")
		break
	}

	// Device removed from topo
	topoEventRemove := topodevice.ListResponse{
		Type:   topodevice.ListResponseREMOVED,
		Device: &device1,
	}

	sessionManager.topoChannel <- &topoEventRemove

	time.Sleep(1 * time.Second)

	sessionManager.operationalStateCacheLock.RLock()
	_, ok = sessionManager.operationalStateCache[device1.ID]
	sessionManager.operationalStateCacheLock.RUnlock()
	assert.Assert(t, !ok, "Expected Op state cache entry to have been removed")

	close(sessionManager.topoChannel)

	/*****************************************************************
	 * Now it should have cleaned up after itself
	 *****************************************************************/
	/*time.Sleep(time.Millisecond * 100) // Give it a second for the event to take effect
	listeners := sessionManager.dispatcher.GetListeners()
	assert.Equal(t, 0, len(listeners))*/

	// TODO: Retries recreate the op state in the cache
	//opStateCacheLock.RLock()
	//_, ok = opstateCache[device1.ID]
	//opStateCacheLock.RUnlock()
	//assert.Assert(t, !ok, "Op state cache entry deleted")*/
}
