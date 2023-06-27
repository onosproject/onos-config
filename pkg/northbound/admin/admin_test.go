/*
 * SPDX-FileCopyrightText: 2022-present Intel Corporation
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package admin

import (
	"context"
	"github.com/atomix/go-sdk/pkg/test"
	"github.com/golang/mock/gomock"
	adminapi "github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	pluginmock "github.com/onosproject/onos-config/internal/pluginregistry"
	topomock "github.com/onosproject/onos-config/internal/store/topo"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/onosproject/onos-config/pkg/store/v2/configuration"
	"github.com/onosproject/onos-config/pkg/store/v2/transaction"
	"github.com/onosproject/onos-config/pkg/utils/path"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testContext struct {
	mctl   *gomock.Controller
	atomix *test.Client
	topo   *topomock.MockStore
	server *Server
}

func createServer(t *testing.T) *testContext {
	mctl := gomock.NewController(t)
	registryMock := pluginmock.NewMockPluginRegistry(mctl)
	topoMock := topomock.NewMockStore(mctl)

	atomixTest, cfgStore, txStore := testStores(t)

	return &testContext{
		mctl:   mctl,
		atomix: atomixTest,
		topo:   topoMock,
		server: &Server{
			txStore,
			cfgStore,
			registryMock,
		},
	}
}

func testStores(t *testing.T) (*test.Client, configuration.Store, transaction.Store) {
	cluster := test.NewClient()

	cfgStore, err := configuration.NewAtomixStore(cluster)
	assert.NoError(t, err)

	txStore, err := transaction.NewAtomixStore(cluster)
	assert.NoError(t, err)

	return cluster, cfgStore, txStore
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

func setupTopoAndRegistry(t *testing.T, test *testContext, id string, model string, version string, noPlugin bool) {
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
	plugin.EXPECT().LeafValueSelection(gomock.Any(), "/foo", gomock.Any()).AnyTimes().
		Return([]string{id, model, version}, nil)
	plugin.EXPECT().LeafValueSelection(gomock.Any(), "/bar", gomock.Any()).AnyTimes().
		Return([]string{"bar1", "bar2", "bar3"}, nil)
	plugin.EXPECT().LeafValueSelection(gomock.Any(), "/goo", gomock.Any()).AnyTimes().
		Return([]string{}, nil)
	plugin.EXPECT().LeafValueSelection(gomock.Any(), "", gomock.Any()).AnyTimes().
		Return(nil, errors.NewInvalid("navigatedValue path cannot be empty"))

	mockRegistry, ok := test.server.pluginRegistry.(*pluginmock.MockPluginRegistry)
	assert.True(t, ok)
	mockRegistry.EXPECT().GetPlugin(configapi.TargetType(model), configapi.TargetVersion(version)).AnyTimes().
		Return(plugin, !noPlugin)

	test.topo.EXPECT().Get(gomock.Any(), gomock.Eq(topoapi.ID(id))).AnyTimes().
		Return(topoEntity(topoapi.ID(id), model, version), nil)
	test.topo.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		Return(nil)
}

func TestServer_LeafSelectionQuery_NoChange(t *testing.T) {
	type testCase struct {
		name      string
		req       adminapi.LeafSelectionQueryRequest
		expected  []string
		expectErr string
	}

	test := createServer(t)
	defer test.atomix.Close()
	defer test.mctl.Finish()

	// Create a plugin and a topo entry for target-1 with 1.0.0 model
	setupTopoAndRegistry(t, test, "target-1", "devicesim", "1.0.0", false)
	// Create another for the 2.0.0 model
	setupTopoAndRegistry(t, test, "target-1", "devicesim", "2.0.0", false)

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

	err := test.server.configurationsStore.Create(context.TODO(), targetConfig)
	assert.NoError(t, err)

	targetConfigV2 := &configapi.Configuration{
		ID:       configuration.NewID(targetID, "devicesim", "2.0.0"),
		TargetID: targetID,
		Values:   targetConfigValues,
	}

	err = test.server.configurationsStore.Create(context.TODO(), targetConfigV2)
	assert.NoError(t, err)

	testCases := []testCase{
		{
			name: "valid path",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "devicesim",
				Version:       "1.0.0",
				SelectionPath: "/foo",
			},
			expected: []string{"target-1", "devicesim", "1.0.0"},
		},
		{
			name: "alternate version",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "devicesim",
				Version:       "2.0.0",
				SelectionPath: "/foo",
			},
			expected: []string{"target-1", "devicesim", "2.0.0"},
		},
		{
			name: "invalid target",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-2",
				Type:          "devicesim",
				Version:       "1.0.0",
				SelectionPath: "/foo",
			},
			expectErr: "rpc error: code = NotFound desc = key target-2-devicesim-1.0.0 not found",
		},
		{
			name: "invalid type",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "device-wrong",
				Version:       "1.0.0",
				SelectionPath: "/foo",
			},
			expectErr: "rpc error: code = NotFound desc = key target-1-device-wrong-1.0.0 not found",
		},
		{
			name: "selection path empty",
			req: adminapi.LeafSelectionQueryRequest{
				Target:  "target-1",
				Type:    "devicesim",
				Version: "1.0.0",
			},
			expectErr: "rpc error: code = InvalidArgument desc = error getting leaf selection for ''. navigatedValue path cannot be empty",
		},
		{
			name: "selection path no values - not an error",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "devicesim",
				Version:       "1.0.0",
				SelectionPath: "/goo",
			},
		},
	}

	for _, tc := range testCases {
		resp, err1 := test.server.LeafSelectionQuery(context.TODO(), &tc.req)
		if tc.expectErr != "" {
			if err1 != nil {
				assert.Equal(t, tc.expectErr, err1.Error(), tc.name)
				continue
			}
			t.Logf("%s Expected error %s", tc.name, tc.expectErr)
			t.FailNow()
		}
		assert.NoError(t, err1, tc.name)
		assert.NotNil(t, resp, tc.name)
		assert.Equal(t, len(tc.expected), len(resp.Selection), tc.name)
		if len(tc.expected) == 2 {
			assert.Equal(t, tc.req.Version, resp.Selection[2])
		}
	}
}

func TestServer_LeafSelectionQuery_WithChange(t *testing.T) {
	type testCase struct {
		name      string
		req       adminapi.LeafSelectionQueryRequest
		expected  []string
		expectErr string
	}

	test := createServer(t)
	defer test.atomix.Close()
	defer test.mctl.Finish()

	// Create a plugin and a topo entry for target-1 with 1.0.0 model
	setupTopoAndRegistry(t, test, "target-1", "devicesim", "1.0.0", false)
	// Create another for the 2.0.0 model
	setupTopoAndRegistry(t, test, "target-1", "devicesim", "2.0.0", false)

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

	err := test.server.configurationsStore.Create(context.TODO(), targetConfig)
	assert.NoError(t, err)

	targetConfigV2 := &configapi.Configuration{
		ID:       configuration.NewID(targetID, "devicesim", "2.0.0"),
		TargetID: targetID,
		Values:   targetConfigValues,
	}

	err = test.server.configurationsStore.Create(context.TODO(), targetConfigV2)
	assert.NoError(t, err)

	// change context
	gnmiSetrequest := &gnmi.SetRequest{
		Update: []*gnmi.Update{
			{
				Path: &gnmi.Path{
					Target: string(targetID),
					Elem: []*gnmi.PathElem{
						{
							Name: "bar",
						},
					},
				},
				Val: &gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: "Hello all!"}},
			},
		},
	}

	testCases := []testCase{
		{
			name: "valid path",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "devicesim",
				Version:       "1.0.0",
				SelectionPath: "/foo",
			},
			expected: []string{"target-1", "devicesim", "1.0.0"},
		},
		{
			name: "alternate version",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "devicesim",
				Version:       "2.0.0",
				SelectionPath: "/foo",
				ChangeContext: gnmiSetrequest,
			},
			expected: []string{"target-1", "devicesim", "2.0.0"},
		},
		{
			name: "invalid target",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-2",
				Type:          "devicesim",
				Version:       "1.0.0",
				SelectionPath: "/foo",
				ChangeContext: gnmiSetrequest,
			},
			expectErr: "rpc error: code = NotFound desc = key target-2-devicesim-1.0.0 not found",
		},
		{
			name: "invalid type",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "device-wrong",
				Version:       "1.0.0",
				SelectionPath: "/foo",
				ChangeContext: gnmiSetrequest,
			},
			expectErr: "rpc error: code = NotFound desc = key target-1-device-wrong-1.0.0 not found",
		},
		{
			name: "selection path empty",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "devicesim",
				Version:       "1.0.0",
				ChangeContext: gnmiSetrequest,
			},
			expectErr: "rpc error: code = InvalidArgument desc = error getting leaf selection for ''. navigatedValue path cannot be empty",
		},
		{
			name: "selection path - change context not in the config store",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "devicesim",
				Version:       "1.0.0",
				SelectionPath: "/bar",
				ChangeContext: gnmiSetrequest,
			},
			expected: []string{"bar1", "bar2", "bar3"},
		},
		{
			name: "selection path no values - not an error",
			req: adminapi.LeafSelectionQueryRequest{
				Target:        "target-1",
				Type:          "devicesim",
				Version:       "1.0.0",
				SelectionPath: "/goo",
				ChangeContext: gnmiSetrequest,
			},
		},
	}

	for _, tc := range testCases {
		resp, err1 := test.server.LeafSelectionQuery(context.TODO(), &tc.req)
		if tc.expectErr != "" {
			if err1 != nil {
				assert.Equal(t, tc.expectErr, err1.Error(), tc.name)
				continue
			}
			t.Logf("%s Expected error %s", tc.name, tc.expectErr)
			t.FailNow()
		}
		assert.NoError(t, err1, tc.name)
		assert.NotNil(t, resp, tc.name)
		assert.Equal(t, len(tc.expected), len(resp.Selection), tc.name)
		if len(tc.expected) == 2 {
			assert.Equal(t, tc.req.Version, resp.Selection[2])
		}
	}
}
