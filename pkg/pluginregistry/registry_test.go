// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package pluginregistry

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"strings"
	"testing"
)

const testEndpoint1 = "testmodel1:5152"
const testEndpoint2 = "testmodel2:5153"

var pr = NewPluginRegistry(testEndpoint1, testEndpoint2).(*pluginRegistry)

type MockModelPluginServiceClient struct {
	getModelInfoResponse *admin.ModelInfoResponse
}

func (m MockModelPluginServiceClient) GetModelInfo(ctx context.Context, in *admin.ModelInfoRequest, opts ...grpc.CallOption) (*admin.ModelInfoResponse, error) {
	return m.getModelInfoResponse, nil
}

func (MockModelPluginServiceClient) ValidateConfig(ctx context.Context, in *admin.ValidateConfigRequest, opts ...grpc.CallOption) (*admin.ValidateConfigResponse, error) {
	return nil, nil
}

func (MockModelPluginServiceClient) GetPathValues(ctx context.Context, in *admin.PathValuesRequest, opts ...grpc.CallOption) (*admin.PathValuesResponse, error) {
	return nil, nil
}

func TestLoadPluginInfo(t *testing.T) {
	assert.Equal(t, 0, len(pr.plugins), "Plugins list is not empty at the beginning of the test")

	plugin := &ModelPluginInfo{
		Endpoint: "localhost:5152",
	}

	modelInfo := &admin.ModelInfo{Name: "Testmodel", Version: "1.0.0"}
	mockClient := MockModelPluginServiceClient{
		getModelInfoResponse: &admin.ModelInfoResponse{
			ModelInfo: modelInfo,
		},
	}

	pr.loadPluginInfo(mockClient, plugin)

	assert.Equal(t, strings.ToLower(fmt.Sprintf("%s-%s", modelInfo.Name, modelInfo.Version)), plugin.ID, "Plugin ID is wrong")
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
