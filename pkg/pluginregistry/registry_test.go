// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package pluginregistry

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	test_plugin "github.com/onosproject/onos-config/pkg/northbound/gnmi/test-plugin"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
)

const testEndpoint1 = "testmodel1:5152"
const testEndpoint2 = "testmodel2:5153"

var pr = NewPluginRegistry(testEndpoint1, testEndpoint2).(*pluginRegistry)

func TestLoadPluginInfo(t *testing.T) {
	mctl := gomock.NewController(t)
	pluginClient := test_plugin.NewMockModelPluginServiceClient(mctl)
	pluginClient.EXPECT().ValidateConfig(gomock.Any(), gomock.Any()).AnyTimes().Return(
		&admin.ValidateConfigResponse{
			Valid:   false,
			Message: "just a mocked response",
		},
		nil,
	)
	pluginClient.EXPECT().GetValueSelection(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(
		&admin.ValueSelectionResponse{
			Selection: []string{"value1", "value2"},
		},
		nil,
	)

	assert.Equal(t, 0, len(pr.plugins), "Plugins list is not empty at the beginning of the test")

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
	modelType := configapi.TargetType("testmodel")
	modelVersion := configapi.TargetVersion("1.0.0")
	testID := fmt.Sprintf("%s-%s", modelType, modelVersion)
	p1 := &ModelPluginInfo{
		Endpoint: "localhost:5152",
		ID:       testID,
	}
	p2 := &ModelPluginInfo{
		Endpoint: "localhost:5153",
		ID:       "someID",
	}

	pr.plugins[p1.ID] = p1
	pr.plugins[p2.ID] = p2

	plugin, found := pr.GetPlugin(modelType, modelVersion)
	assert.True(t, found, "Plugin not found")
	assert.NotNil(t, plugin, "Plugin not found")

	n, notFound := pr.GetPlugin(configapi.TargetType("non-existing"), configapi.TargetVersion("1.0.0"))
	assert.False(t, notFound, "Plugin should not have been found")
	assert.Nil(t, n)

	// test that the plugin lookup is case insensitive
	plugin, found = pr.GetPlugin(configapi.TargetType("Testmodel"), modelVersion)
	assert.True(t, found, "Plugin not found")
	assert.NotNil(t, plugin, "Plugin not found")
}
