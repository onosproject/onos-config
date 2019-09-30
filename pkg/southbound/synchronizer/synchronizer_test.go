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
	context2 "context"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"testing"
	"time"
)

func synchronizerSetUp() (*store.ChangeStore, *store.ConfigurationStore,
	chan events.TopoEvent, chan<- events.OperationalStateEvent,
	chan events.DeviceResponse, *dispatcher.Dispatcher,
	*modelregistry.ModelRegistry, modelregistry.ReadOnlyPathMap,
	change.TypedValueMap, <-chan events.ConfigEvent,
	error) {

	changeStore, err := store.LoadChangeStore("../../../configs/changeStore-sample.json")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}
	configStore, err := store.LoadConfigStore("../../../configs/configStore-sample.json")
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
	}

	dispatcher := dispatcher.NewDispatcher()
	mr := new(modelregistry.ModelRegistry)
	opStateCache := make(change.TypedValueMap)
	roPathMap := make(modelregistry.ReadOnlyPathMap)
	return &changeStore, &configStore,
		make(chan events.TopoEvent),
		make(chan events.OperationalStateEvent),
		make(chan events.DeviceResponse),
		dispatcher, mr, roPathMap, opStateCache,
		make(chan events.ConfigEvent),
		nil
}

func TestNew(t *testing.T) {
	changeStore, configStore, topoChan, opstateChan, responseChan, dispatcher,
		models, roPathMap, opstateCache, configChan, err := synchronizerSetUp()
	assert.NilError(t, err, "Error in factorySetUp()")
	assert.Assert(t, changeStore != nil)
	assert.Assert(t, configStore != nil)
	assert.Assert(t, topoChan != nil)
	assert.Assert(t, opstateChan != nil)
	assert.Assert(t, responseChan != nil)
	assert.Assert(t, configChan != nil)
	assert.Assert(t, dispatcher != nil)
	assert.Assert(t, models != nil)
	assert.Assert(t, roPathMap != nil)
	assert.Assert(t, opstateCache != nil)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockTarget := NewMockTargetIf(ctrl)
	modelData1 := gnmi.ModelData{
		Name:         "test1",
		Organization: "Open Networking Foundation",
		Version:      "2018-02-20",
	}
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

	mockTarget.EXPECT().ConnectTarget(
		gomock.Any(),
		device1,
	).Return(southbound.DeviceID{DeviceID: device1NameStr}, nil)

	mockTarget.EXPECT().CapabilitiesWithString(
		gomock.Any(),
		"",
	).Return(&gnmi.CapabilityResponse{
		SupportedModels:      []*gnmi.ModelData{&modelData1},
		SupportedEncodings:   []gnmi.Encoding{gnmi.Encoding_PROTO},
		GNMIVersion:          "0.7.0",
		Extension:            nil,
		XXX_NoUnkeyedLiteral: struct{}{},
		XXX_unrecognized:     nil,
		XXX_sizecache:        0,
	}, nil)

	synchronizer, err := New(context2.Background(), changeStore, configStore, &device1, configChan,
		opstateChan, responseChan, opstateCache, roPathMap, mockTarget)
	assert.NilError(t, err, "Creating synchronizer")
	assert.Equal(t, string(synchronizer.ID), device1NameStr)
	assert.Equal(t, string(synchronizer.Device.ID), device1NameStr)

}
