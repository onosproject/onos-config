// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"strings"
	"sync"
)

//go:generate mockgen -source registry.go -destination test/mock-registry.go -package test ModelPlugin

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

	chunkSize = 100000 // 100kB
)

// ModelPlugin defines the expected behaviour of a model plugin
type ModelPlugin interface {
	// GetInfo returns the model plugin info
	GetInfo() *ModelPluginInfo

	// Capabilities returns the model plugin gNMI capabilities response
	Capabilities(ctx context.Context) *gnmi.CapabilityResponse

	// Validate validates the specified JSON configuration against the plugin's schema
	Validate(ctx context.Context, jsonData []byte) error

	// GetPathValues extracts typed path values from the specified configuration change JSON
	GetPathValues(ctx context.Context, pathPrefix string, jsonData []byte) ([]*configapi.PathValue, error)

	// LeafValueSelection gets a list of valid options for a leaf by applying selection rules in YANG
	LeafValueSelection(ctx context.Context, selectionPath string, jsonData []byte) ([]string, error)
}

// ModelPluginInfo is a record of information compiled from the configuration model plugin
type ModelPluginInfo struct {
	// The ID is used to identify and index the plugin.
	// Since the final value (comprised of ModelInfoResponse.ModelInfo.Name and ModelInfoResponse.ModelInfo.Version)
	// is discovered only after the plugin has been loaded and
	// a temporary one is generated if the plugin fails to load.
	// The ID is then used to lookup the plugin when a gNMI Set request is received,
	// and it is learned via the gNMI Extension 101 and 102
	ID             string
	Endpoint       string
	Info           api.ModelInfo
	Client         api.ModelPluginServiceClient
	ReadOnlyPaths  path.ReadOnlyPathMap
	ReadWritePaths path.ReadWritePathMap
	NamespaceMap   path.NamespaceMap
	// Status indicates whether a plugin was correctly loaded
	Status modelPluginStatus
	// Error is an optional field populated only if the plugin failed to be correctly discovered
	Error string
}

// PluginRegistry is a set of available configuration model plugins
type PluginRegistry interface {
	// Start the plugin registry
	Start()
	// Stop the plugin registry
	Stop()

	// GetPlugin returns the plugin with the specified ID
	GetPlugin(model configapi.TargetType, version configapi.TargetVersion) (ModelPlugin, bool)

	// GetPlugins returns list of all registered plugins
	GetPlugins() []ModelPlugin

	// NewClientFn -
	NewClientFn(func(endpoint string) (api.ModelPluginServiceClient, error))
}

type pluginRegistry struct {
	endpoints   []string
	plugins     map[string]*ModelPluginInfo
	lock        sync.RWMutex
	newClientFn func(endpoint string) (api.ModelPluginServiceClient, error)
}

// NewPluginRegistry creates a plugin registry that will search the specified gRPC ports to look for model plugins
func NewPluginRegistry(endpoints ...string) PluginRegistry {
	registry := &pluginRegistry{
		endpoints:   endpoints,
		plugins:     make(map[string]*ModelPluginInfo),
		lock:        sync.RWMutex{},
		newClientFn: newClient,
	}
	log.Infof("Created configuration plugin registry with ports: %+v", endpoints)
	return registry
}

// Start the plugin registry
func (r *pluginRegistry) Start() {
	// Discover plugins synchronously on start-up.
	r.discoverPlugins()
}

// Stop the plugin registry
func (r *pluginRegistry) Stop() {
	// TODO: hook for shutdown; nothing required for now
}

func (r *pluginRegistry) discoverPlugins() {
	// TODO: Is it sufficient to do a one-time discovery? For now yes.
	for _, endpoint := range r.endpoints {
		r.discoverPlugin(endpoint)
	}
}

func (r *pluginRegistry) discoverPlugin(endpoint string) {
	log.Infof("Attempting to contact model plugin at: %s", endpoint)

	plugin := &ModelPluginInfo{
		Endpoint: endpoint,
		ID:       endpoint, // we assign the ID as Endpoint as we don't know the Model Name and Version yet.
	}

	client, err := r.newClientFn(plugin.Endpoint)
	if err != nil {
		plugin.Status = loadingError
		plugin.Error = fmt.Sprintf("Unable to create model plugin client: %+v", err)
		log.Errorw(plugin.Error, "pluginId", plugin.ID)

		r.lock.Lock()
		r.plugins[plugin.ID] = plugin
		r.lock.Unlock()
		return
	}
	plugin.Client = client

	r.loadPluginInfo(client, plugin)
}
func (r *pluginRegistry) loadPluginInfo(client api.ModelPluginServiceClient, plugin *ModelPluginInfo) {
	resp, err := client.GetModelInfo(context.Background(), &api.ModelInfoRequest{})
	if err != nil {
		// NOTE we'll never get here only the error has code: Canceled or DeadlineExceeded
		// in all the other cases the RetryingUnaryClientInterceptor will keep retry
		plugin.Status = loadingError
		plugin.Error = fmt.Sprintf("Unable to load model info: %+v", err)
		log.Errorw(plugin.Error, "pluginId", plugin.ID)
		r.lock.Lock()
		r.plugins[plugin.ID] = plugin
		r.lock.Unlock()
		return
	}
	plugin.Status = loaded
	plugin.Info = *resp.ModelInfo
	// we finally have the information we need to populate the ID
	plugin.ID = strings.ToLower(fmt.Sprintf("%s-%s", resp.ModelInfo.Name, resp.ModelInfo.Version))

	// Reconstitute the r/o and r/w path map variables from the model data.
	plugin.ReadOnlyPaths = getRoPathMap(resp)
	plugin.ReadWritePaths = getRWPathMap(resp)
	plugin.NamespaceMap = getNamespaceMap(resp)

	r.lock.Lock()
	r.plugins[plugin.ID] = plugin
	r.lock.Unlock()

	log.Debugf("Got model info for plugin: %+v", plugin)

	log.Infof("Configuration model plugin %s discovered on %s", plugin.ID, plugin.Endpoint)
}

func getRoPathMap(resp *api.ModelInfoResponse) path.ReadOnlyPathMap {
	pm := make(map[string]path.ReadOnlySubPathMap)
	for _, pe := range resp.ModelInfo.ReadOnlyPath {
		roSubPathMap := path.ReadOnlySubPathMap{}
		for _, subPath := range pe.SubPath {
			roSubPathMap[subPath.SubPath] = *subPath
		}
		pm[pe.Path] = roSubPathMap
	}
	return pm
}

func getRWPathMap(resp *api.ModelInfoResponse) path.ReadWritePathMap {
	pm := make(map[string]api.ReadWritePath)
	for _, pe := range resp.ModelInfo.ReadWritePath {
		pm[pe.Path] = *pe
	}
	return pm
}

func getNamespaceMap(resp *api.ModelInfoResponse) path.NamespaceMap {
	ns := make(map[string]string)
	for _, n := range resp.ModelInfo.NamespaceMappings {
		ns[n.Prefix] = n.Module
	}

	return ns
}

func getTypeOpts(typeOpts []uint64) []uint8 {
	tos := make([]uint8, 0, len(typeOpts))
	for _, to := range typeOpts {
		tos = append(tos, uint8(to))
	}
	return tos
}

func (r *pluginRegistry) NewClientFn(newFunc func(endpoint string) (api.ModelPluginServiceClient, error)) {
	r.newClientFn = newFunc
}

func newClient(endpoint string) (api.ModelPluginServiceClient, error) {
	clientCreds, _ := getClientCredentials()
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(clientCreds)),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable))),
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable))),
	}
	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		return nil, err
	}
	return api.NewModelPluginServiceClient(conn), nil
}

// GetPlugin returns the plugin with the specified ID
// NOTE this method might get slow if we have a lot of plugins loaded. Do we see that as a possibility?
// If so we might need to re-work the registry and index the plugins using the ID while using an autogenerated ID
// for the ones that fail to load (eg: unknown-fee754)
func (r *pluginRegistry) GetPlugin(modelType configapi.TargetType, version configapi.TargetVersion) (ModelPlugin, bool) {
	id := strings.ToLower(fmt.Sprintf("%s-%s", modelType, version))
	r.lock.RLock()
	defer r.lock.RUnlock()
	p, ok := r.plugins[id]
	if !ok {
		log.Warnw("Cannot find plugin", "id", id)
		return nil, false
	}
	return p, true
}

// GetPlugins returns list of all registered plugins
func (r *pluginRegistry) GetPlugins() []ModelPlugin {
	plugins := make([]ModelPlugin, 0, len(r.plugins))
	r.lock.RLock()
	defer r.lock.RUnlock()
	for _, p := range r.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// GetInfo returns the model plugin info
func (p *ModelPluginInfo) GetInfo() *ModelPluginInfo {
	return p
}

// Capabilities returns the model plugin gNMI capabilities response
func (p *ModelPluginInfo) Capabilities(ctx context.Context) *gnmi.CapabilityResponse {
	return &gnmi.CapabilityResponse{
		SupportedModels:    p.Info.ModelData,
		SupportedEncodings: p.Info.SupportedEncodings,
		GNMIVersion:        "0.7.0",
		Extension:          nil,
	}
}

// Validate validates the specified JSON configuration against the plugin's schema
func (p *ModelPluginInfo) Validate(ctx context.Context, jsonData []byte) error {
	sender, err := p.Client.ValidateConfigChunked(ctx)
	if err != nil {
		return err
	}

	jsonLen := len(jsonData)
	position := 0
	for position < jsonLen {
		var chunk []byte
		if position+chunkSize < jsonLen {
			chunk = jsonData[position : position+chunkSize]
			position += chunkSize
		} else {
			chunk = jsonData[position:]
			position = jsonLen
		}
		sendErr := sender.Send(&api.ValidateConfigRequestChunk{
			Json: chunk,
		})
		if sendErr != nil {
			return fmt.Errorf("error sending chunk to model plugin. %v", sendErr)
		}
	}
	resp, err := sender.CloseAndRecv()

	if err != nil {
		return err
	}
	if !resp.Valid {
		return errors.NewInvalid("configuration is not valid: %s", resp.Message)
	}
	return nil
}

// GetPathValues extracts typed path values from the specified configuration change JSON
func (p *ModelPluginInfo) GetPathValues(ctx context.Context, pathPrefix string, jsonData []byte) ([]*configapi.PathValue, error) {
	resp, err := p.Client.GetPathValues(ctx, &api.PathValuesRequest{PathPrefix: pathPrefix, Json: jsonData})
	if err != nil {
		return nil, err
	}
	return resp.PathValues, nil
}

// LeafValueSelection gets a list of valid options for a leaf by applying selection rules in YANG
func (p *ModelPluginInfo) LeafValueSelection(ctx context.Context, selectionPath string, jsonData []byte) ([]string, error) {
	resp, err := p.Client.GetValueSelection(ctx, &api.ValueSelectionRequest{
		SelectionPath: selectionPath,
		ConfigJson:    jsonData,
	})
	if err != nil {
		return nil, err
	}
	return resp.Selection, nil
}

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
