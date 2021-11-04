// Copyright 2020-present Open Networking Foundation.
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

package modelregistry

import (
	"context"
	configmodelapi "github.com/onosproject/onos-api/go/onos/configmodel"
	"github.com/onosproject/onos-config/model"
	"github.com/onosproject/onos-config/model/plugin/cache"
	"github.com/onosproject/onos-config/model/plugin/compiler"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
	"sync"
)

// NewService :
func NewService(registry *ConfigModelRegistry, cache *plugincache.PluginCache, compiler *plugincompiler.PluginCompiler) northbound.Service {
	return &Service{
		registry: registry,
		cache:    cache,
		compiler: compiler,
	}
}

// Service :
type Service struct {
	registry *ConfigModelRegistry
	cache    *plugincache.PluginCache
	compiler *plugincompiler.PluginCompiler
}

// Register :
func (s *Service) Register(r *grpc.Server) {
	server := &Server{
		registry: s.registry,
		cache:    s.cache,
		compiler: s.compiler,
	}
	configmodelapi.RegisterConfigModelRegistryServiceServer(r, server)
}

var _ northbound.Service = &Service{}

// Server is a registry server
type Server struct {
	registry *ConfigModelRegistry
	cache    *plugincache.PluginCache
	compiler *plugincompiler.PluginCompiler
	mu       sync.RWMutex
}

// GetModel :
func (s *Server) GetModel(ctx context.Context, request *configmodelapi.GetModelRequest) (*configmodelapi.GetModelResponse, error) {
	log.Debugf("Received GetModelRequest %+v", request)
	s.mu.RLock()
	defer s.mu.RUnlock()

	name, version := configmodel.Name(request.Name), configmodel.Version(request.Version)
	modelInfo, err := s.registry.GetModel(name, version)
	if err != nil {
		log.Warnf("GetModelRequest %+v failed: %v", request, err)
		return nil, errors.Status(err).Err()
	}

	var modules []*configmodelapi.ConfigModule
	for _, moduleInfo := range modelInfo.Modules {
		modules = append(modules, &configmodelapi.ConfigModule{
			Name:         string(moduleInfo.Name),
			Organization: moduleInfo.Organization,
			Revision:     string(moduleInfo.Revision),
			File:         moduleInfo.File,
		})
	}
	response := &configmodelapi.GetModelResponse{
		Model: &configmodelapi.ConfigModel{
			Name:    string(modelInfo.Name),
			Version: string(modelInfo.Version),
			Modules: modules,
		},
	}
	log.Debugf("Sending GetModelResponse %+v", response)
	return response, nil
}

// ListModels :
func (s *Server) ListModels(ctx context.Context, request *configmodelapi.ListModelsRequest) (*configmodelapi.ListModelsResponse, error) {
	log.Debugf("Received ListModelsRequest %+v", request)
	s.mu.RLock()
	defer s.mu.RUnlock()

	modelInfos, err := s.registry.ListModels()
	if err != nil {
		log.Warnf("ListModelsRequest %+v failed: %v", request, err)
		return nil, errors.Status(err).Err()
	}

	var models []*configmodelapi.ConfigModel
	for _, modelInfo := range modelInfos {
		var modules []*configmodelapi.ConfigModule
		for _, module := range modelInfo.Modules {
			modules = append(modules, &configmodelapi.ConfigModule{
				Name:         string(module.Name),
				Organization: module.Organization,
				Revision:     string(module.Revision),
				File:         module.File,
			})
		}
		models = append(models, &configmodelapi.ConfigModel{
			Name:    string(modelInfo.Name),
			Version: string(modelInfo.Version),
			Modules: modules,
		})
	}

	response := &configmodelapi.ListModelsResponse{
		Models: models,
	}
	log.Debugf("Sending ListModelsResponse %+v", response)
	return response, nil
}

// PushModel :
func (s *Server) PushModel(ctx context.Context, request *configmodelapi.PushModelRequest) (*configmodelapi.PushModelResponse, error) {
	log.Debugf("Received PushModelRequest %+v", request)
	s.mu.Lock()
	defer s.mu.Unlock()

	name, version := configmodel.Name(request.Model.Name), configmodel.Version(request.Model.Version)

	// First check the registry for the model
	_, err := s.registry.GetModel(name, version)
	if err == nil {
		err = errors.NewAlreadyExists("model '%s@%s' already exists", request.Model.Name, request.Model.Version)
	}
	if err != nil && !errors.IsNotFound(err) {
		log.Warnf("PushModelRequest '%s@%s' failed: %s", request.Model.Name, request.Model.Version, err)
		return nil, errors.Status(err).Err()
	}

	// Add the model if it's not already present in the registry
	fileInfos := make([]configmodel.FileInfo, 0, len(request.Model.Files))
	for path, data := range request.Model.Files {
		fileInfos = append(fileInfos, configmodel.FileInfo{
			Path: path,
			Data: []byte(data),
		})
	}

	moduleInfos := make([]configmodel.ModuleInfo, len(request.Model.Modules))
	for i, module := range request.Model.Modules {
		moduleInfos[i] = configmodel.ModuleInfo{
			Name:         configmodel.Name(module.Name),
			File:         module.File,
			Organization: module.Organization,
			Revision:     configmodel.Revision(module.Revision),
		}
	}

	var getStateMode configmodel.GetStateMode
	switch request.Model.GetStateMode {
	case configmodelapi.GetStateMode_NONE:
		getStateMode = configmodel.GetStateNone
	case configmodelapi.GetStateMode_OP_STATE:
		getStateMode = configmodel.GetStateOpState
	case configmodelapi.GetStateMode_EXPLICIT_RO_PATHS:
		getStateMode = configmodel.GetStateExplicitRoPaths
	case configmodelapi.GetStateMode_EXPLICIT_RO_PATHS_EXPAND_WILDCARDS:
		getStateMode = configmodel.GetStateExplicitRoPathsExpandWildcards
	}

	modelInfo := configmodel.ModelInfo{
		Name:         configmodel.Name(request.Model.Name),
		Version:      configmodel.Version(request.Model.Version),
		GetStateMode: getStateMode,
		Files:        fileInfos,
		Modules:      moduleInfos,
		Plugin: configmodel.PluginInfo{
			Name:    configmodel.Name(request.Model.Name),
			Version: configmodel.Version(request.Model.Version),
		},
	}

	// Acquire a lock on the cache before adding it to the registry to ensure subsequent
	// requests to load the same plugin will be blocked until compilation is complete.
	entry := s.cache.Entry(name, version)
	if err := entry.Lock(ctx); err != nil {
		log.Errorf("Failed to acquire cache lock: %s", err)
		return nil, errors.Status(err).Err()
	}

	defer func() {
		if err := recover(); err != nil {
			_ = entry.Unlock(context.Background())
		}
	}()

	// Add the model to the registry
	err = s.registry.AddModel(modelInfo)
	if err != nil {
		log.Warnf("PushModelRequest '%s@%s' failed: %s", request.Model.Name, request.Model.Version, err)
		return nil, errors.Status(err).Err()
	}

	// Look for the plugin in the cache
	cached, err := entry.Cached()
	if err != nil {
		log.Errorf("Failed to compile plugin for model '%s@%s': %s", request.Model.Name, request.Model.Version, err)
		return nil, errors.Status(err).Err()
	}

	// If the plugin is not present in the cache, compile it
	if !cached {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					_ = entry.Unlock(context.Background())
				}
			}()

			defer func() {
				if err := entry.Unlock(context.Background()); err != nil {
					log.Errorf("Failed to release cache lock: %s", err)
				}
			}()

			err = s.compiler.CompilePlugin(modelInfo, entry.Path)
			if err != nil {
				log.Errorf("Failed to compile plugin for model '%s@%s': %s", request.Model.Name, request.Model.Version, err)
			}
		}()
	}

	response := &configmodelapi.PushModelResponse{}
	log.Debugf("Sending PushModelResponse %+v", response)
	return response, nil
}

// DeleteModel :
func (s *Server) DeleteModel(ctx context.Context, request *configmodelapi.DeleteModelRequest) (*configmodelapi.DeleteModelResponse, error) {
	log.Debugf("Received DeleteModelRequest %+v", request)
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.registry.RemoveModel(configmodel.Name(request.Name), configmodel.Version(request.Version))
	if err != nil {
		log.Warnf("DeleteModelRequest %+v failed: %v", request, err)
		return nil, errors.Status(err).Err()
	}

	response := &configmodelapi.DeleteModelResponse{}
	log.Debugf("Sending DeleteModelResponse %+v", response)
	return response, nil
}

var _ configmodelapi.ConfigModelRegistryServiceServer = &Server{}
