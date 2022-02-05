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
	configurationcontroller "github.com/onosproject/onos-config/pkg/controller/configuration"
	proposalcontroller "github.com/onosproject/onos-config/pkg/controller/proposal"
	transactioncontroller "github.com/onosproject/onos-config/pkg/controller/transaction"
	sb "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/proposal"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"sync"
	"testing"

	"github.com/onosproject/onos-config/pkg/pluginregistry"

	adminapi "github.com/onosproject/onos-api/go/onos/config/admin"

	atomixtest "github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/golang/mock/gomock"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmitest "github.com/onosproject/onos-config/pkg/northbound/gnmi/test"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"github.com/onosproject/onos-config/pkg/store/transaction"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/path"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

type testContext struct {
	mctl                    *gomock.Controller
	atomix                  *atomixtest.Test
	topo                    *gnmitest.MockStore
	registry                *gnmitest.MockPluginRegistry
	conns                   sb.ConnManager
	configuration           configuration.Store
	configurationController *controller.Controller
	proposal                proposal.Store
	proposalController      *controller.Controller
	transaction             transaction.Store
	transactionController   *controller.Controller
	server                  *Server
}

func createServer(t *testing.T) *testContext {
	mctl := gomock.NewController(t)
	registryMock := gnmitest.NewMockPluginRegistry(mctl)
	topoMock := gnmitest.NewMockStore(mctl)
	atomixTest, cfgStore, txStore := testStores(t)

	return &testContext{
		mctl:          mctl,
		atomix:        atomixTest,
		topo:          topoMock,
		registry:      registryMock,
		configuration: cfgStore,
		transaction:   txStore,
		server: &Server{
			mu:             sync.RWMutex{},
			pluginRegistry: registryMock,
			topo:           topoMock,
			transactions:   txStore,
			configurations: cfgStore,
		},
	}
}

func (test *testContext) startControllers(t *testing.T) {
	test.conns = sb.NewConnManager()
	test.configurationController = configurationcontroller.NewController(test.topo, test.conns, test.server.configurations)
	assert.NoError(t, test.configurationController.Start())

	test.proposalController = proposalcontroller.NewController(test.topo, test.conns, test.server.proposals, test.server.configurations, test.registry)
	assert.NoError(t, test.configurationController.Start())

	test.transactionController = transactioncontroller.NewController(test.server.transactions, test.server.proposals)
	assert.NoError(t, test.transactionController.Start())
}

func (test *testContext) stopControllers() {
	test.transactionController.Stop()
	test.configurationController.Stop()
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

func setupTopoAndRegistry(test *testContext, id string, model string, version string, noPlugin bool) {
	plugin := gnmitest.NewMockModelPlugin(test.mctl)
	rwPaths := path.ReadWritePathMap{}
	for _, p := range []string{"/foo", "/bar", "/goo"} {
		rwPaths[p] = path.ReadWritePathElem{
			ReadOnlyAttrib: path.ReadOnlyAttrib{
				ValueType: configapi.ValueType_STRING,
			},
		}
	}
	plugin.EXPECT().GetInfo().AnyTimes().
		Return(&pluginregistry.ModelPluginInfo{Info: adminapi.ModelInfo{Name: model, Version: version}, ReadWritePaths: rwPaths})
	plugin.EXPECT().Validate(gomock.Any(), gomock.Any()).AnyTimes().
		Return(nil)

	pathValues := make([]*configapi.PathValue, 0)
	pathValues = append(pathValues, &configapi.PathValue{
		Path:    "/foo",
		Value:   configapi.TypedValue{Bytes: []byte("Yo!"), Type: configapi.ValueType_STRING, TypeOpts: []int32{}},
		Deleted: false,
	})
	plugin.EXPECT().GetPathValues(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		Return(pathValues, nil)

	test.registry.EXPECT().GetPlugin(configapi.TargetType(model), configapi.TargetVersion(version)).AnyTimes().
		Return(plugin, !noPlugin)
	test.topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(id))).AnyTimes().
		Return(topoEntity(topoapi.ID(id), model, version), nil)
	test.topo.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		Return(nil)
}

func Test_GetNoTarget(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	noTargetPath1 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}
	noTargetPath2 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&noTargetPath1, &noTargetPath2},
	}

	_, err := test.server.Get(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "has no target")
}

func Test_GetUnsupportedEncoding(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	request := gnmi.GetRequest{
		Path:     []*gnmi.Path{targetPath(t, "target", "foo")},
		Encoding: gnmi.Encoding_BYTES,
	}

	_, err := test.server.Get(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid encoding")
}

func Test_BasicGet(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	targetConfigValues := make(map[string]*configapi.PathValue)
	targetConfigValues["/foo"] = &configapi.PathValue{
		Path: "/foo",
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world!"),
			Type:  configapi.ValueType_STRING,
		},
	}

	targetID := configapi.TargetID("target-1")
	targetConfig := &configapi.Configuration{
		ID:       configapi.ConfigurationID(targetID),
		TargetID: targetID,
		Values:   targetConfigValues,
	}

	err := test.server.configurations.Create(context.TODO(), targetConfig)
	assert.NoError(t, err)

	request := gnmi.GetRequest{
		Path:     []*gnmi.Path{targetPath(t, targetID, "foo")},
		Encoding: gnmi.Encoding_JSON,
	}

	result, err := test.server.Get(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Notification, 1)
	assert.Len(t, result.Notification[0].Update, 1)
	assert.Equal(t, "{\n  \"foo\": \"Hello world!\"\n}",
		string(result.Notification[0].Update[0].GetVal().GetJsonVal()))
}

func Test_GetWithPrefixOnly(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Stop()
	defer test.mctl.Finish()

	setupTopoAndRegistry(test, "target-1", "devicesim", "1.0.0", false)

	targetConfigValues := make(map[string]*configapi.PathValue)
	targetConfigValues["/foo"] = &configapi.PathValue{
		Path: "/foo",
		Value: configapi.TypedValue{
			Bytes: []byte("Hello world!"),
			Type:  configapi.ValueType_STRING,
		},
	}

	targetID := configapi.TargetID("target-1")
	targetConfig := &configapi.Configuration{
		ID:       configapi.ConfigurationID(targetID),
		TargetID: targetID,
		Values:   targetConfigValues,
	}

	err := test.server.configurations.Create(context.TODO(), targetConfig)
	assert.NoError(t, err)

	request := gnmi.GetRequest{
		Prefix:   targetPath(t, targetID, "foo"),
		Path:     []*gnmi.Path{},
		Encoding: gnmi.Encoding_JSON,
	}

	result, err := test.server.Get(context.TODO(), &request)
	assert.NoError(t, err)
	assert.Len(t, result.Notification, 1)
	assert.Len(t, result.Notification[0].Update, 1)
	assert.Equal(t, "{\n  \"foo\": \"Hello world!\"\n}",
		string(result.Notification[0].Update[0].GetVal().GetJsonVal()))
}
