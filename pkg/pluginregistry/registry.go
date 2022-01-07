// Copyright 2021-present Open Networking Foundation.
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

package pluginregistry

import (
	"context"
	"crypto/tls"
	"fmt"
	api "github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"sync"
)

var log = logging.GetLogger("registry")

// ModelPlugin is a record of information compiled from the configuration model plugin
type ModelPlugin struct {
	ID     string
	Port   uint
	Info   api.ModelInfo
	Client api.ModelPluginServiceClient
}

// PluginRegistry is a set of available configuration model plugins
type PluginRegistry struct {
	ports   []uint
	plugins map[string]*ModelPlugin
	lock    sync.RWMutex
}

// NewPluginRegistry creates a plugin registry that will search the specified gRPC ports to look for model plugins
func NewPluginRegistry(ports ...uint) *PluginRegistry {
	registry := &PluginRegistry{
		ports:   ports,
		plugins: make(map[string]*ModelPlugin),
	}
	log.Infof("Created configuration plugin registry with ports: %+v", ports)
	go registry.discoverPlugins()
	return registry
}

func (r *PluginRegistry) discoverPlugins() {
	// TODO: Is it sufficient to do a one-time discovery? For now yes.
	for _, port := range r.ports {
		r.discoverPlugin(port)
	}
}

func (r *PluginRegistry) discoverPlugin(port uint) {
	log.Infof("Attempting to contact model plugin on port %d", port)
	client, err := newClient(port)
	if err != nil {
		log.Error("Unable to create model plugin client: %+v", err)
		return
	}

	resp, err := client.GetModelInfo(context.Background(), &api.ModelInfoRequest{})
	if err != nil {
		log.Error("Unable to create model plugin client: %+v", err)
		return
	}

	plugin := &ModelPlugin{
		ID:     fmt.Sprintf("%s-%s", resp.ModelInfo.Name, resp.ModelInfo.Version),
		Port:   port,
		Info:   *resp.ModelInfo,
		Client: client,
	}
	log.Debugf("Got model info for plugin: %+v", plugin)

	r.lock.Lock()
	defer r.lock.Unlock()
	r.plugins[plugin.ID] = plugin
	log.Infof("Configuration model plugin %s discovered on port %d", plugin.ID, port)
}

const localhost = "localhost"

func newClient(port uint) (api.ModelPluginServiceClient, error) {
	clientCreds, _ := getClientCredentials()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(clientCreds)),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor()),
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor()),
	}
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", localhost, port), opts...)
	if err != nil {
		return nil, err
	}
	return api.NewModelPluginServiceClient(conn), nil
}

// GetPlugin returns the plugin with the specified ID
func (r *PluginRegistry) GetPlugin(id string) (*ModelPlugin, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	p, ok := r.plugins[id]
	return p, ok
}

// GetPlugins returns list of all registered plugins
func (r *PluginRegistry) GetPlugins() []*ModelPlugin {
	plugins := make([]*ModelPlugin, 0, len(r.plugins))
	r.lock.RLock()
	defer r.lock.RUnlock()
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// Validate validates the specified JSON configuration against the plugin's schema
func (p *ModelPlugin) Validate(ctx context.Context, jsonData []byte) error {
	resp, err := p.Client.ValidateConfig(ctx, &api.ValidateConfigRequest{Json: jsonData})
	if err != nil {
		return err
	}
	if !resp.Valid {
		return errors.NewInvalid("configuration is not valid")
	}
	return nil
}

// GetClientCredentials :
func getClientCredentials() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}, nil
}
