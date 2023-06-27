// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"github.com/atomix/go-sdk/pkg/test"
	configurationcontroller "github.com/onosproject/onos-config/pkg/controller/v2/configuration"
	proposalcontroller "github.com/onosproject/onos-config/pkg/controller/v2/proposal"
	transactioncontroller "github.com/onosproject/onos-config/pkg/controller/v2/transaction"
	sb "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	proposal "github.com/onosproject/onos-config/pkg/store/v2/proposal"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/onosproject/onos-config/pkg/pluginregistry"

	adminapi "github.com/onosproject/onos-api/go/onos/config/admin"

	"github.com/golang/mock/gomock"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	pluginmock "github.com/onosproject/onos-config/internal/pluginregistry"
	topomock "github.com/onosproject/onos-config/internal/store/topo"
	configuration "github.com/onosproject/onos-config/pkg/store/v2/configuration"
	transaction "github.com/onosproject/onos-config/pkg/store/v2/transaction"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/path"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

type testContext struct {
	mctl                    *gomock.Controller
	atomix                  *test.Client
	topo                    *topomock.MockStore
	registry                *pluginmock.MockPluginRegistry
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
	registryMock := pluginmock.NewMockPluginRegistry(mctl)
	topoMock := topomock.NewMockStore(mctl)
	atomixTest, cfgStore, propStore, txStore := testStores(t)

	return &testContext{
		mctl:          mctl,
		atomix:        atomixTest,
		topo:          topoMock,
		registry:      registryMock,
		configuration: cfgStore,
		proposal:      propStore,
		transaction:   txStore,
		server: &Server{
			mu:             sync.RWMutex{},
			pluginRegistry: registryMock,
			topo:           topoMock,
			transactions:   txStore,
			proposals:      propStore,
			configurations: cfgStore,
		},
	}
}

func (test *testContext) startControllers(t *testing.T) {
	test.conns = sb.NewConnManager()
	test.configurationController = configurationcontroller.NewController(test.topo, test.conns, test.server.configurations)
	assert.NoError(t, test.configurationController.Start())

	test.proposalController = proposalcontroller.NewController(test.topo, test.conns, test.server.proposals, test.server.configurations, test.registry)
	assert.NoError(t, test.proposalController.Start())

	test.transactionController = transactioncontroller.NewController(test.server.transactions, test.server.proposals)
	assert.NoError(t, test.transactionController.Start())
}

func (test *testContext) stopControllers() {
	test.transactionController.Stop()
	test.configurationController.Stop()
}

func testStores(t *testing.T) (*test.Client, configuration.Store, proposal.Store, transaction.Store) {
	cluster := test.NewClient()

	cfgStore, err := configuration.NewAtomixStore(cluster)
	assert.NoError(t, err)

	propStore, err := proposal.NewAtomixStore(cluster)
	assert.NoError(t, err)

	txStore, err := transaction.NewAtomixStore(cluster)
	assert.NoError(t, err)

	return cluster, cfgStore, propStore, txStore
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
	plugin := pluginmock.NewMockModelPlugin(test.mctl)
	rwPaths := path.ReadWritePathMap{}
	for _, p := range []string{"/foo", "/bar", "/goo", "/some/nested/path"} {
		rwPaths[p] = adminapi.ReadWritePath{
			ValueType: configapi.ValueType_STRING,
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
	defer test.atomix.Close()
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
	defer test.atomix.Close()
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
	defer test.atomix.Close()
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
		ID:       configuration.NewID(targetID, "devicesim", "1.0.0"),
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

func Test_BasicGetUpdateWithOverride(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Close()
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

	tvoext, err := utils.TargetVersionOverrideExtension(targetID, "devicesim", "1.0.0")
	assert.NoError(t, err)

	targetConfig := &configapi.Configuration{
		ID:       configuration.NewID(targetID, "devicesim", "1.0.0"),
		TargetID: targetID,
		Values:   targetConfigValues,
	}

	err = test.server.configurations.Create(context.TODO(), targetConfig)
	assert.NoError(t, err)

	request := gnmi.GetRequest{
		Path:      []*gnmi.Path{targetPath(t, targetID, "foo")},
		Encoding:  gnmi.Encoding_JSON,
		Extension: []*gnmi_ext.Extension{tvoext},
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
	defer test.atomix.Close()
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
		ID:       configuration.NewID(targetID, "devicesim", "1.0.0"),
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

func Test_ReportAllTargetsNoSecurityAndJson(t *testing.T) {
	test := createServer(t)
	defer test.atomix.Close()
	defer test.mctl.Finish()

	topo1 := topoEntity("target-1", "devicesim", "1.0.0")
	topo2 := topoEntity("target-2", "devicesim", "1.0.0")
	topo3 := topoEntity("target-3", "devicesim", "1.0.0")

	test.topo.EXPECT().List(gomock.Any(), gomock.Any()).Return(
		[]topoapi.Object{
			*topo1,
			*topo2,
			*topo3,
		}, nil,
	).AnyTimes()

	testCtx := context.Background()
	response, err := test.server.reportAllTargets(testCtx, gnmi.Encoding_JSON_IETF, []string{
		"target-1", "target-2", "non-target-group",
	})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, len(response.Notification))
	not0 := response.Notification[0]
	assert.Equal(t, 1, len(not0.Update))
	upd00 := not0.Update[0]
	assert.Equal(t, "elem:{name:\"all-targets\"} target:\"*\"",
		strings.ReplaceAll(upd00.Path.String(), "  ", " "))

	// Expect all 3 because no security is applied
	expectedJSON := `json_val:"{\"targets\": [\"target-1\",\"target-2\",\"target-3\"]}"`
	assert.Equal(t, expectedJSON, upd00.GetVal().String())
}

func Test_ReportAllTargetsWithSecurityAndProto(t *testing.T) {
	test := createServer(t)
	oldValue := os.Getenv(OIDCServerURL)
	err := os.Setenv(OIDCServerURL, "test-value")
	assert.NoError(t, err)
	defer test.atomix.Close()
	defer test.mctl.Finish()
	defer func(key, value string) {
		err := os.Setenv(key, value)
		if err != nil {
			t.Logf("Error unsetting env var: %s, %v", OIDCServerURL, err)
		}
	}(OIDCServerURL, oldValue)

	topo1 := topoEntity("target-1", "devicesim", "1.0.0")
	topo2 := topoEntity("target-2", "devicesim", "1.0.0")
	topo3 := topoEntity("target-3", "devicesim", "1.0.0")

	test.topo.EXPECT().List(gomock.Any(), gomock.Any()).Return(
		[]topoapi.Object{
			*topo1,
			*topo2,
			*topo3,
		}, nil,
	).AnyTimes()

	testCtx := context.Background()
	response, err := test.server.reportAllTargets(testCtx, gnmi.Encoding_PROTO, []string{
		"target-1", "target-2", "non-target-group",
	})
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 1, len(response.Notification))
	not0 := response.Notification[0]
	assert.Equal(t, 1, len(not0.Update))
	upd00 := not0.Update[0]
	assert.Equal(t, "elem:{name:\"all-targets\"} target:\"*\"",
		strings.ReplaceAll(upd00.Path.String(), "  ", " "))

	expectedProto := `leaflist_val:{element:{string_val:"target-1"} element:{string_val:"target-2"}}`
	assert.Equal(t, expectedProto, strings.ReplaceAll(upd00.GetVal().String(), "  ", " "))

	// Now try it with an "AetherRocAdmin" - should allow all
	response2, err := test.server.reportAllTargets(testCtx, gnmi.Encoding_PROTO, []string{
		"target-1", "target-2", "non-target-group", aetherROCAdmin,
	})
	assert.NoError(t, err)
	assert.NotNil(t, response2)
	assert.Equal(t, 1, len(response2.Notification))
	not20 := response2.Notification[0]
	assert.Equal(t, 1, len(not20.Update))
	upd200 := not20.Update[0]
	assert.Equal(t, "elem:{name:\"all-targets\"} target:\"*\"",
		strings.ReplaceAll(upd200.Path.String(), "  ", " "))

	expectedProto2 := `leaflist_val:{element:{string_val:"target-1"} element:{string_val:"target-2"} element:{string_val:"target-3"}}`
	assert.Equal(t, expectedProto2, strings.ReplaceAll(upd200.GetVal().String(), "  ", " "))
}
