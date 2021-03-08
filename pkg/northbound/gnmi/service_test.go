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

package gnmi

import (
	"context"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"testing"
)

func TestService_getGNMIServiceVersion(t *testing.T) {
	version, err := getGNMIServiceVersion()
	assert.NoError(t, err)
	assert.Equal(t, *version, "0.7.0")
}

func TestService_Capabilities(t *testing.T) {
	server := Server{}
	request := gnmi.CapabilityRequest{}
	config := modelregistry.Config{
		ModPath:      "test/data/TestService_Capabilities/mod",
		RegistryPath: "test/data/TestService_Capabilities/registry",
		PluginPath:   "test/data/TestService_Capabilities/plugins",
		ModTarget:    "github.com/onosproject/onos-config@master",
	}
	modelRegistry, err := modelregistry.NewModelRegistry(config)
	assert.NoError(t, err)
	manager.GetManager().ModelRegistry = modelRegistry
	response, err := server.Capabilities(context.Background(), &request)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, response.GNMIVersion, "0.7.0")
	assert.Equal(t, len(response.SupportedEncodings), 3)
	assert.Equal(t, response.SupportedEncodings[0], gnmi.Encoding_JSON)
	assert.Equal(t, response.SupportedEncodings[1], gnmi.Encoding_JSON_IETF)
	assert.Equal(t, response.SupportedEncodings[2], gnmi.Encoding_PROTO)
}

func TestService_Register(t *testing.T) {
	service := Service{}
	server := grpc.NewServer()
	m := manager.GetManager()
	m.Dispatcher = dispatcher.NewDispatcher()
	service.Register(server)
	// If the registration does not crash with a fatal error it was successful
}
