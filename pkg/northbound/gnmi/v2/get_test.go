// Copyright 2022-present Open Networking Foundation.
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
	"context"
	atomixtest "github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/golang/mock/gomock"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmitest "github.com/onosproject/onos-config/pkg/northbound/gnmi/test"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"github.com/onosproject/onos-config/pkg/store/transaction"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func createServer(t *testing.T) (*Server, *gomock.Controller, *atomixtest.Test, *gnmitest.MockStore, *gnmitest.MockPluginRegistry) {
	mctl := gomock.NewController(t)
	test, cfgStore, txStore := testStores(t)
	registryMock := gnmitest.NewMockPluginRegistry(mctl)
	topoMock := gnmitest.NewMockStore(mctl)
	return &Server{
		mu:             sync.RWMutex{},
		pluginRegistry: registryMock,
		topo:           topoMock,
		transactions:   txStore,
		configurations: cfgStore,
	}, mctl, test, topoMock, registryMock
}

func testStores(t *testing.T) (*atomixtest.Test, configuration.Store, transaction.Store) {
	test := atomixtest.NewTest(rsm.NewProtocol(), atomixtest.WithReplicas(1), atomixtest.WithPartitions(1))
	assert.NoError(t, test.Start())

	client1, err := test.NewClient("node-1")
	assert.NoError(t, err)

	cfgStore, err := configuration.NewAtomixStore(client1)
	assert.NoError(t, err)

	txStore, err := transaction.NewAtomixStore(client1)
	assert.NoError(t, err)

	return test, cfgStore, txStore
}

func targetPath(t *testing.T, target configapi.TargetID, elms ...string) *gnmi.Path {
	path, err := utils.ParseGNMIElements(elms)
	assert.NoError(t, err)
	path.Target = string(target)
	return path
}

func topoEntity(id topoapi.ID, targetType string, targetVersion string) *topoapi.Object {
	entity := &topoapi.Object{
		ID:   id,
		Type: topoapi.Object_ENTITY,
		Obj: &topoapi.Object_Entity{
			Entity: &topoapi.Entity{},
		},
	}
	_ = entity.SetAspect(&topoapi.Configurable{
		Type:    targetType,
		Address: "",
		Target:  string(id),
		Version: targetVersion,
	})
	return entity
}

func Test_GetNoTarget(t *testing.T) {
	server, mctl, test, _, _ := createServer(t)
	defer test.Stop()
	defer mctl.Finish()

	noTargetPath1 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}
	noTargetPath2 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&noTargetPath1, &noTargetPath2},
	}

	_, err := server.Get(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "has no target")
}

func Test_GetUnsupportedEncoding(t *testing.T) {
	server, mctl, test, _, _ := createServer(t)
	defer test.Stop()
	defer mctl.Finish()

	request := gnmi.GetRequest{
		Path:     []*gnmi.Path{targetPath(t, "target", "foo")},
		Encoding: gnmi.Encoding_BYTES,
	}

	_, err := server.Get(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid encoding")
}

func Test_BasicGet(t *testing.T) {
	server, mctl, test, topo, registry := createServer(t)
	defer test.Stop()
	defer mctl.Finish()

	id := "target-1"
	topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(id))).AnyTimes().
		Return(topoEntity(topoapi.ID(id), "devicesim-1.0.x", "1.0.0"), nil)
	plugin := gnmitest.NewMockModelPlugin(mctl)
	plugin.EXPECT().Validate(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	registry.EXPECT().GetPlugin("devicesim", "1.0.0").AnyTimes().Return(plugin, true)

	targetConfigValues := make(map[string]*configapi.PathValue)
	targetConfigValues["/foo"] = &configapi.PathValue{
		Path: "/foo",
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world!"),
			Type:  configapi.ValueType_STRING,
		},
	}

	target := configapi.TargetID(id)
	targetConfig := &configapi.Configuration{
		ID:            configapi.ConfigurationID(target),
		TargetID:      target,
		TargetVersion: "1.0.0",
		Values:        targetConfigValues,
	}

	err := server.configurations.Create(context.TODO(), targetConfig)
	assert.NoError(t, err)

	request := gnmi.GetRequest{
		Path:     []*gnmi.Path{targetPath(t, target, "foo")},
		Encoding: gnmi.Encoding_JSON,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Notification, 1)
	assert.Len(t, result.Notification[0].Update, 1)
	assert.Equal(t, "{\n  \"foo\": \"Hello world!\"\n}",
		string(result.Notification[0].Update[0].GetVal().GetJsonVal()))
}

func Test_GetWithPrefixOnly(t *testing.T) {
	server, mctl, test, topo, registry := createServer(t)
	defer test.Stop()
	defer mctl.Finish()

	id := "target-1"
	topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(id))).AnyTimes().
		Return(topoEntity(topoapi.ID(id), "devicesim-1.0.0", "1.0.0"), nil)
	plugin := gnmitest.NewMockModelPlugin(mctl)
	plugin.EXPECT().Validate(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	registry.EXPECT().GetPlugin("devicesim", "1.0.0").AnyTimes().Return(plugin, true)

	targetConfigValues := make(map[string]*configapi.PathValue)
	targetConfigValues["/foo"] = &configapi.PathValue{
		Path: "/foo",
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world!"),
			Type:  configapi.ValueType_STRING,
		},
	}

	target := configapi.TargetID(id)
	targetConfig := &configapi.Configuration{
		ID:            configapi.ConfigurationID(target),
		TargetID:      target,
		TargetVersion: "1.0.0",
		Values:        targetConfigValues,
	}

	err := server.configurations.Create(context.TODO(), targetConfig)
	assert.NoError(t, err)

	request := gnmi.GetRequest{
		Prefix:   targetPath(t, target, "foo"),
		Path:     []*gnmi.Path{},
		Encoding: gnmi.Encoding_JSON,
	}

	result, err := server.Get(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Notification, 1)
	assert.Len(t, result.Notification[0].Update, 1)
	assert.Equal(t, "{\n  \"foo\": \"Hello world!\"\n}",
		string(result.Notification[0].Update[0].GetVal().GetJsonVal()))
}
