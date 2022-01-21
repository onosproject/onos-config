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
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/utils/path"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"strconv"
	"strings"
	"sync"
)

var log = logging.GetLogger("registry")

type modelPluginStatus int

func (m modelPluginStatus) String() string {
	names := [...]string{
		"Loaded",
		"Error",
	}
	return names[m]
}

const (
	loaded modelPluginStatus = iota
	loadingError
)

// ModelPlugin is a record of information compiled from the configuration model plugin
type ModelPlugin struct {
	ID             string
	Port           uint
	Info           api.ModelInfo
	Client         api.ModelPluginServiceClient
	ReadOnlyPaths  path.ReadOnlyPathMap
	ReadWritePaths path.ReadWritePathMap
	Status         modelPluginStatus
	Error          string
}

// PluginRegistry is a set of available configuration model plugins
type PluginRegistry struct {
	endpoints []string
	plugins   map[string]*ModelPlugin
	lock      sync.RWMutex
}

// NewPluginRegistry creates a plugin registry that will search the specified gRPC ports to look for model plugins
func NewPluginRegistry(endpoints ...string) *PluginRegistry {
	registry := &PluginRegistry{
		endpoints: endpoints,
		plugins:   make(map[string]*ModelPlugin),
		lock:      sync.RWMutex{},
	}
	log.Infof("Created configuration plugin registry with ports: %+v", endpoints)
	return registry
}

// Start the plugin registry
func (r *PluginRegistry) Start() {
	go r.discoverPlugins()
}

// Stop the plugin registry
func (r *PluginRegistry) Stop() {
	// TODO: hook for shutdown; nothing required for now
}

func (r *PluginRegistry) discoverPlugins() {
	// TODO: Is it sufficient to do a one-time discovery? For now yes.
	for _, endpoint := range r.endpoints {
		r.discoverPlugin(endpoint)
	}
}

func (r *PluginRegistry) discoverPlugin(endpoint string) {

	log.Infof("Attempting to contact model plugin at: %s", endpoint)
	pieces := strings.Split(endpoint, ":")
	name := pieces[0]

	plugin := &ModelPlugin{
		ID: name,
	}
	r.lock.Lock()
	r.plugins[plugin.ID] = plugin
	r.lock.Unlock()

	port, e := strconv.ParseUint(pieces[1], 10, 32)
	if e != nil {
		plugin.Status = loadingError
		plugin.Error = fmt.Sprintf("Cannot parse port %s for plugin %s", pieces[1], name)
		log.Errorw(plugin.Error, "pluginId", plugin.ID)
		return
	}
	plugin.Port = uint(port)

	client, err := newClient(plugin.Port)
	if err != nil {
		plugin.Status = loadingError
		plugin.Error = fmt.Sprintf("Unable to create model plugin client: %+v", err)
		log.Errorw(plugin.Error, "pluginId", plugin.ID)
		return
	}
	plugin.Client = client

	resp, err := client.GetModelInfo(context.Background(), &api.ModelInfoRequest{})
	if err != nil {
		// NOTE we'll never get here only the error has code: Canceled or DeadlineExceeded
		// in all the other cases the RetryingUnaryClientInterceptor will keep retry
		plugin.Status = loadingError
		plugin.Error = fmt.Sprintf("Unable to load model info: %+v", err)
		log.Errorw(plugin.Error, "pluginId", plugin.ID)
		return
	}
	plugin.Status = loaded
	plugin.Info = *resp.ModelInfo

	// Reconstitute the r/o and r/w path map variables from the model data.
	plugin.ReadOnlyPaths = getRoPathMap(resp)
	plugin.ReadWritePaths = getRWPathMap(resp)

	log.Debugf("Got model info for plugin: %+v", plugin)

	log.Infof("Configuration model plugin %s discovered on port %d", plugin.ID, port)
}

func getRoPathMap(resp *api.ModelInfoResponse) path.ReadOnlyPathMap {
	pm := make(map[string]path.ReadOnlySubPathMap)
	for _, pe := range resp.ModelInfo.ReadOnlyPath {
		// TODO: Implement conversion
		pm[pe.Path] = path.ReadOnlySubPathMap{}
	}
	return pm
}

func getRWPathMap(resp *api.ModelInfoResponse) path.ReadWritePathMap {
	pm := make(map[string]path.ReadWritePathElem)
	for _, pe := range resp.ModelInfo.ReadWritePath {
		pm[pe.Path] = path.ReadWritePathElem{
			ReadOnlyAttrib: path.ReadOnlyAttrib{
				ValueType:   pe.ValueType,
				TypeOpts:    getTypeOpts(pe.TypeOpts),
				Description: pe.Description,
				Units:       pe.Units,
				IsAKey:      pe.IsAKey,
				AttrName:    pe.AttrName,
			},
			Mandatory: pe.Mandatory,
			Default:   pe.Default,
			Range:     pe.Range,
			Length:    pe.Length,
		}
	}
	return pm
}

func getTypeOpts(typeOpts []uint64) []uint8 {
	tos := make([]uint8, 0, len(typeOpts))
	for _, to := range typeOpts {
		tos = append(tos, uint8(to))
	}
	return tos
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

// Capabilities returns the model plugin gNMI capabilities response
func (p *ModelPlugin) Capabilities(ctx context.Context, jsonData []byte) *gnmi.CapabilityResponse {
	return &gnmi.CapabilityResponse{
		SupportedModels:    p.Info.ModelData,
		SupportedEncodings: p.Info.SupportedEncodings,
		GNMIVersion:        "0.7.0",
		Extension:          nil,
	}
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

// GetPathValues extracts typed path values from the specified configuration change JSON
func (p *ModelPlugin) GetPathValues(ctx context.Context, pathPrefix string, jsonData []byte) ([]*configapi.PathValue, error) {
	resp, err := p.Client.GetPathValues(ctx, &api.PathValuesRequest{PathPrefix: pathPrefix, Json: jsonData})
	if err != nil {
		return nil, err
	}
	return resp.PathValues, nil
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
