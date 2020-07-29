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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	dispatcherpkg "github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	modelregistrypkg "github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	storemock "github.com/onosproject/onos-config/pkg/test/mocks/store"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"github.com/onosproject/onos-topo/api/topo"
	"gotest.tools/assert"
)

func factorySetUp(t *testing.T) (chan *topo.SubscribeResponse, chan<- events.OperationalStateEvent,
	chan events.DeviceResponse, *dispatcherpkg.Dispatcher,
	*modelregistrypkg.ModelRegistry, map[topo.ID]devicechange.TypedValueMap, *sync.RWMutex, device.Store, error) {

	dispatcher := dispatcherpkg.NewDispatcher()
	modelregistry := new(modelregistrypkg.ModelRegistry)
	opStateCache := make(map[topo.ID]devicechange.TypedValueMap)
	opStateCacheLock := &sync.RWMutex{}
	ctrl := gomock.NewController(t)
	deviceChangeStore := storemock.NewMockDeviceChangesStore(ctrl)
	deviceChangeStore.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(deviceID devicetype.VersionedID, c chan<- *devicechange.DeviceChange) (stream.Context, error) {
			ctx := stream.NewContext(func() {})
			return ctx, errors.New("no Configuration found")
		}).AnyTimes()
	return make(chan *topo.SubscribeResponse),
		make(chan events.OperationalStateEvent),
		make(chan events.DeviceResponse),
		dispatcher, modelregistry, opStateCache, opStateCacheLock, deviceChangeStore, nil
}

/**
 * Check device is added as a synchronizer correctly, times out on no gRPC device
 * and then un-does everything
 */
func TestFactory_Revert(t *testing.T) {
	topoChan, opstateChan, responseChan, dispatcher, models, opstateCache, opStateCacheLock, deviceChangeStore, err := factorySetUp(t)
	assert.NilError(t, err, "Error in factorySetUp(t)")
	assert.Assert(t, topoChan != nil)
	assert.Assert(t, opstateChan != nil)
	assert.Assert(t, responseChan != nil)
	assert.Assert(t, dispatcher != nil)
	assert.Assert(t, models != nil)
	assert.Assert(t, opstateCache != nil)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		Factory(topoChan, opstateChan, responseChan, dispatcher, models, opstateCache, southbound.NewTarget,
			opStateCacheLock, deviceChangeStore)
		wg.Done()
	}()

	//timeout := time.Millisecond * 500
	device1NameStr := "factoryTd"
	device1 := topo.Object{
		ID:   topo.ID(device1NameStr),
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{Protocols: []*topodevice.ProtocolState{}},
		},
		Attributes: map[string]string{
			topo.Address: "1.2.3.4:11161",
			topo.Target:  "",
			topo.Version: "1.0.0",
			/*
				topo.Timeout:     &timeout,
				Credentials: topodevice.Credentials{},
				TLS:         topodevice.TlsConfig{},
			*/
			topo.Type: "TestDevice",
			topo.Role: "leaf",
		},
	}
	topoEvent := topo.SubscribeResponse{
		Update: &topo.Update{
			Type:   topo.Update_INSERT,
			Object: &device1,
		},
	}

	topoChan <- &topoEvent

	// Wait for gRPC connection to timeout
	time.Sleep(time.Millisecond * 600) // Give it a moment for the event to take effect and for timeout to happen
	opStateCacheLock.RLock()
	opStateCacheUpdated, ok := opstateCache[device1.ID]
	opStateCacheLock.RUnlock()
	assert.Assert(t, ok, "Op state cache entry created")
	assert.Equal(t, len(opStateCacheUpdated), 0)

	time.Sleep(1 * time.Second)

	// Device removed from topo
	//device1.Attributes["t1"] = "test"
	device1Update := topo.Object{
		ID:   topo.ID(device1NameStr),
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{Protocols: []*topodevice.ProtocolState{}},
		},
		Attributes: map[string]string{
			topo.Address: "1.2.3.4:11161",
			topo.Target:  "",
			topo.Version: "1.0.0",
			/*Timeout:     &timeout,
			Credentials: topodevice.Credentials{},
			TLS:         topodevice.TlsConfig{},*/
			topo.Type: "TestDevice",
			topo.Role: "spine", // Role is changed - will be ignored
		},
	}
	topoEventUpdated := topo.SubscribeResponse{
		Update: &topo.Update{
			Type:   topo.Update_MODIFY,
			Object: &device1Update,
		},
	}
	topoChan <- &topoEventUpdated

	for resp := range responseChan {
		assert.Error(t, resp.Error(),
			"topo update event ignored update:<type:MODIFY object:<id:\"factoryTd\" type:ENTITY entity:<> attributes:<key:\"address\" value:\"1.2.3.4:11161\" > attributes:<key:\"role\" value:\"spine\" > attributes:<key:\"target\" value:\"\" > attributes:<key:\"type\" value:\"TestDevice\" > attributes:<key:\"version\" value:\"1.0.0\" > > > ")
		break
	}

	// Device removed from topo
	topoEventRemove := topo.SubscribeResponse{
		Update: &topo.Update{
			Type:   topo.Update_DELETE,
			Object: &device1,
		},
	}

	topoChan <- &topoEventRemove

	time.Sleep(1 * time.Second)

	opStateCacheLock.RLock()
	_, ok = opstateCache[device1.ID]
	opStateCacheLock.RUnlock()
	assert.Assert(t, !ok, "Expected Op state cache entry to have been removed")

	close(topoChan)

	wg.Wait()

	/*****************************************************************
	 * Now it should have cleaned up after itself
	 *****************************************************************/
	time.Sleep(time.Millisecond * 100) // Give it a second for the event to take effect
	listeners := dispatcher.GetListeners()
	assert.Equal(t, 0, len(listeners))

	// TODO: Retries recreate the op state in the cache
	//opStateCacheLock.RLock()
	//_, ok = opstateCache[device1.ID]
	//opStateCacheLock.RUnlock()
	//assert.Assert(t, !ok, "Op state cache entry deleted")
}
