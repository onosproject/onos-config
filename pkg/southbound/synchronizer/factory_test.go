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
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	"gotest.tools/assert"
	"testing"
	"time"
)

func factorySetUp() (*store.ChangeStore, *store.ConfigurationStore,
	chan *devicepb.ListResponse, chan<- events.OperationalStateEvent,
	chan events.DeviceResponse, *dispatcher.Dispatcher,
	*modelregistry.ModelRegistry, map[device.ID]change.TypedValueMap, error) {

	changeStore, err := store.LoadChangeStore("../../../configs/changeStore-sample.json")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	configStore, err := store.LoadConfigStore("../../../configs/configStore-sample.json")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	dispatcher := dispatcher.NewDispatcher()
	modelregistry := new(modelregistry.ModelRegistry)
	opStateCache := make(map[device.ID]change.TypedValueMap)
	return &changeStore, &configStore,
		make(chan *devicepb.ListResponse),
		make(chan events.OperationalStateEvent),
		make(chan events.DeviceResponse),
		dispatcher, modelregistry, opStateCache, nil
}

/**
 * Check device is added as a synchronizer correctly, times out on no gRPC device
 * and then un-does everything
 */
func TestFactory_Revert(t *testing.T) {
	changeStore, configStore, topoChan, opstateChan, responseChan, dispatcher,
		models, opstateCache, err := factorySetUp()
	assert.NilError(t, err, "Error in factorySetUp()")
	assert.Assert(t, changeStore != nil)
	assert.Assert(t, configStore != nil)
	assert.Assert(t, topoChan != nil)
	assert.Assert(t, opstateChan != nil)
	assert.Assert(t, responseChan != nil)
	assert.Assert(t, dispatcher != nil)
	assert.Assert(t, models != nil)
	assert.Assert(t, opstateCache != nil)

	go func() {
		Factory(changeStore, configStore, topoChan, opstateChan, responseChan, dispatcher, models, opstateCache)
	}()

	timeout := time.Millisecond * 500
	device1NameStr := "factoryTd"
	device1 := device.Device{
		ID:          device.ID(device1NameStr),
		Revision:    0,
		Address:     "1.2.3.4:11161",
		Target:      "",
		Version:     "1.0.0",
		Timeout:     &timeout,
		Credentials: device.Credentials{},
		TLS:         device.TlsConfig{},
		Type:        "TestDevice",
		Role:        "leaf",
		Attributes:  nil,
	}
	topoEvent := devicepb.ListResponse{
		Type:   devicepb.ListResponse_ADDED,
		Device: &device1,
	}

	topoChan <- &topoEvent
	time.Sleep(time.Millisecond * 100) // Give it a second for the event to take effect

	listeners := dispatcher.GetListeners()
	assert.Equal(t, 2, len(listeners))
	assert.Equal(t, listeners[0], device1NameStr) // One for DeviceListeners
	assert.Equal(t, listeners[1], device1NameStr) // One for OpState

	respListener, ok := dispatcher.GetResponseListener(device1.ID)
	assert.Assert(t, ok)
	assert.Assert(t, respListener != nil)

	configName := store.ConfigName(utils.ToConfigName(device1.ID, device1.Version))
	cfgStoreUpdated, ok := configStore.Store[configName]
	assert.Assert(t, ok, "Checking config created")
	assert.Equal(t, cfgStoreUpdated.Name, configName)

	opStateCacheUpdated, ok := opstateCache[device1.ID]
	assert.Assert(t, ok, "Op state cache entry created")
	assert.Equal(t, len(opStateCacheUpdated), 0)

	// Wait for gRPC connection to timeout
	time.Sleep(time.Millisecond * 600) // Give it a moment for the event to take effect
	for resp := range responseChan {
		assert.Error(t, resp.Error(),
			"could not create a gNMI client: Dialer(1.2.3.4:11161, 500ms): context deadline exceeded", "after gRPC timeout")
		break
	}

	/*****************************************************************
	 * Now it should have cleaned up after itself
	 *****************************************************************/
	time.Sleep(time.Millisecond * 100) // Give it a second for the event to take effect
	listeners = dispatcher.GetListeners()
	assert.Equal(t, 0, len(listeners))

	_, ok = dispatcher.GetResponseListener(device1.ID)
	assert.Assert(t, !ok)

	_, ok = configStore.Store[configName]
	assert.Assert(t, ok, "Checking config not deleted")

	_, ok = opstateCache[device1.ID]
	assert.Assert(t, !ok, "Op state cache entry deleted")
}
