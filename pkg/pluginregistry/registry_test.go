/*
 * SPDX-FileCopyrightText: 2022-present Intel Corporation
 *
 * SPDX-License-Identifier: Apache-2.0
 */

package pluginregistry

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	pluginmock "github.com/onosproject/onos-config/internal/northbound/admin"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

const testEndpoint1 = "testmodel1:5152"
const testEndpoint2 = "testmodel2:5153"

func TestLoadPluginInfo(t *testing.T) {
	mctl := gomock.NewController(t)
	pluginClient := pluginmock.NewMockModelPluginServiceClient(mctl)
	pluginClient.EXPECT().ValidateConfig(gomock.Any(), gomock.Any()).AnyTimes().Return(
		&admin.ValidateConfigResponse{
			Valid:   false,
			Message: "just a mocked response",
		},
		nil,
	)
	mockSender := pluginmock.NewMockModelPluginService_ValidateConfigChunkedClient(mctl)
	mockSender.EXPECT().Send(gomock.Any()).AnyTimes().Return(
		nil,
	)
	mockSender.EXPECT().CloseAndRecv().AnyTimes().Return(
		&admin.ValidateConfigResponse{
			Valid:   false,
			Message: "just a mocked response",
		},
		nil,
	)
	pluginClient.EXPECT().ValidateConfigChunked(gomock.Any()).AnyTimes().Return(
		mockSender,
		nil,
	)
	pluginClient.EXPECT().GetValueSelection(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(
		&admin.ValueSelectionResponse{
			Selection: []string{"value1", "value2"},
		},
		nil,
	)

	plugin := &ModelPluginInfo{
		Endpoint: "localhost:5152",
		Client:   pluginClient,
		ID:       "Testmodel:1.0.0",
		Info: admin.ModelInfo{
			Name:    "Testmodel",
			Version: "1.0.0",
			ModelData: []*gnmi.ModelData{
				{
					Name:         "module1",
					Organization: "ONF",
					Version:      "2022-11-25",
				},
				{
					Name:         "module2",
					Organization: "ONF",
					Version:      "2022-11-25",
				},
			},
		},
	}

	ctx := context.Background()
	cap := plugin.Capabilities(ctx)
	assert.NotNil(t, cap)
	assert.Equal(t, 2, len(cap.SupportedModels))

	info := plugin.Validate(ctx, []byte("{}"))
	assert.Equal(t, "configuration is not valid: just a mocked response", info.Error())

	selection, err := plugin.LeafValueSelection(ctx, "/some/path", []byte("{}"))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(selection))
}

func TestGetPlugin(t *testing.T) {
	mctl := gomock.NewController(t)
	pluginClient := pluginmock.NewMockModelPluginServiceClient(mctl)

	modelType := configapi.TargetType("testmodel")
	modelVersion1 := configapi.TargetVersion("1.0.0")
	//testID := fmt.Sprintf("%s-%s", modelType, modelVersion)

	pluginClient.EXPECT().GetModelInfo(gomock.Any(), &admin.ModelInfoRequest{}).AnyTimes().Return(&admin.ModelInfoResponse{
		ModelInfo: &admin.ModelInfo{
			Name:    string(modelType),
			Version: string(modelVersion1),
			ModelData: []*gnmi.ModelData{
				{
					Name:         "module1",
					Organization: "ONF",
					Version:      "2022-11-25",
				},
			},
			GetStateMode: 0,
			ReadOnlyPath: []*admin.ReadOnlyPath{
				{
					Path: "/a/b",
					SubPath: []*admin.ReadOnlySubPath{
						{
							SubPath:     "/c",
							ValueType:   configapi.ValueType_INT,
							TypeOpts:    []uint64{8},
							Description: "Leaf C",
							Units:       "",
							IsAKey:      false,
							AttrName:    "c",
						},
					},
				},
			},
			ReadWritePath: []*admin.ReadWritePath{
				{
					Path:        "/a/b/d",
					ValueType:   configapi.ValueType_INT,
					Units:       "W",
					Description: "leaf d RW",
					Mandatory:   true,
					TypeOpts:    []uint64{8},
					AttrName:    "d",
				},
				{
					Path:        "/a/b/e",
					ValueType:   configapi.ValueType_STRING,
					Description: "leaf e RW",
					TypeOpts:    []uint64{},
					AttrName:    "e",
				},
			},
			SupportedEncodings:  nil,
			NamespaceMappings:   nil,
			SouthboundUsePrefix: false,
		},
	}, nil)

	pluginClient.EXPECT().GetPathValues(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(
		&admin.PathValuesResponse{
			PathValues: []*configapi.PathValue{
				{
					Path: "/a/b/c",
					Value: configapi.TypedValue{
						Bytes:    []byte{0x10},
						Type:     configapi.ValueType_INT,
						TypeOpts: []int32{8},
					},
					Deleted: false,
				},
				{
					Path: "/a/b/d",
					Value: configapi.TypedValue{
						Bytes:    []byte{0x20},
						Type:     configapi.ValueType_INT,
						TypeOpts: []int32{8},
					},
					Deleted: false,
				},
			},
		},
		nil,
	)

	pr := NewPluginRegistry(testEndpoint1, testEndpoint2)
	assert.NotNil(t, pr)
	pr.NewClientFn(func(endpoint string) (admin.ModelPluginServiceClient, error) {
		return pluginClient, nil
	})
	pr.Start()
	defer pr.Stop()

	plugin, found := pr.GetPlugin(modelType, modelVersion1)
	assert.True(t, found, "Plugin not found")
	assert.NotNil(t, plugin, "Plugin not found")

	n, notFound := pr.GetPlugin(configapi.TargetType("non-existing"), configapi.TargetVersion("1.0.0"))
	assert.False(t, notFound, "Plugin should not have been found")
	assert.Nil(t, n)

	// test that the plugin lookup is case insensitive
	plugin, found = pr.GetPlugin(configapi.TargetType("Testmodel"), modelVersion1)
	assert.True(t, found, "Plugin not found")
	assert.NotNil(t, plugin, "Plugin not found")

	pv, err := plugin.GetPathValues(context.Background(), "/a", []byte("{'a':{'b':{'c':10}}}"))
	assert.NoError(t, err)
	assert.NotNil(t, pv)
	assert.Equal(t, 2, len(pv))

	pi, ok := plugin.(*ModelPluginInfo)
	assert.True(t, ok)
	assert.NotNil(t, pi)

	assert.Equal(t, 1, len(pi.ReadOnlyPaths))
	ro1subPaths, okRo := pi.ReadOnlyPaths["/a/b"]
	assert.True(t, okRo)
	assert.NotNil(t, ro1subPaths)
	assert.Equal(t, 1, len(ro1subPaths))
	ro1sub1, okRoSub := ro1subPaths["/c"]
	assert.True(t, okRoSub)
	assert.NotNil(t, ro1sub1)
	assert.Equal(t, "c", ro1sub1.AttrName)
	assert.Equal(t, configapi.ValueType_INT, ro1sub1.ValueType)

	assert.Equal(t, 2, len(pi.ReadWritePaths))
	leafD, okLeafD := pi.ReadWritePaths["/a/b/d"]
	assert.True(t, okLeafD)
	assert.Equal(t, "d", leafD.AttrName)
	assert.Equal(t, configapi.ValueType_INT, leafD.ValueType)

	leafE, okLeafE := pi.ReadWritePaths["/a/b/e"]
	assert.True(t, okLeafE)
	assert.Equal(t, "e", leafE.AttrName)
	assert.Equal(t, configapi.ValueType_STRING, leafE.ValueType)
}
