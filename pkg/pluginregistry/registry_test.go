package pluginregistry

import (
	"context"
	"fmt"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"strings"
	"testing"
)

const testEndpoint1 = "testmodel1:5152"
const testEndpoint2 = "testmodel2:5153"
var pr = NewPluginRegistry(testEndpoint1, testEndpoint2)

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

	plugin := &ModelPlugin{
		Name: "testmodel1",
		Port: 5152,
	}

	modelInfo :=  &admin.ModelInfo{Name: "Testmodel", Version: "1.0.0"}
	mockClient := MockModelPluginServiceClient{
		getModelInfoResponse: &admin.ModelInfoResponse{
			ModelInfo: modelInfo,
		},
	}

	pr.loadPluginInfo(mockClient, plugin)

	assert.Equal(t, strings.ToLower(fmt.Sprintf("%s-%s", modelInfo.Name, modelInfo.Version)), plugin.ID, "Plugin ID is wrong")
}

func TestGetPlugin(t *testing.T) {

	testId := "testmodel-1.0.0"
	p1 := &ModelPlugin{
		Name: "testmodel1",
		Port: 5152,
		ID: testId,
	}
	p2 := &ModelPlugin{
		Name: "testmodel2",
		Port: 5153,
		ID: "someID",
	}

	pr.plugins[p1.Name] = p1
	pr.plugins[p2.Name] = p2

	plugin, found := pr.GetPlugin(testId)
	assert.True(t, found, "Plugin not found")
	assert.NotNil(t, plugin, "Plugin not found")
}
