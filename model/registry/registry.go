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
	"encoding/json"
	"fmt"
	"github.com/onosproject/onos-config/model"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/rogpeppe/go-internal/module"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const jsonExt = ".json"

const (
	defaultPath   = "/etc/onos/registry"
	defaultTarget = "github.com/onosproject/onos-config"
)

var log = logging.GetLogger("config-model", "registry")

// Config is a model plugin registry config
type Config struct {
	Path string `yaml:"path" json:"path"`
}

// NewConfigModelRegistry creates a new config model registry
func NewConfigModelRegistry(config Config) *ConfigModelRegistry {
	if config.Path == "" {
		config.Path = defaultPath
	}
	if _, err := os.Stat(config.Path); os.IsNotExist(err) {
		err = os.MkdirAll(config.Path, os.ModePerm)
		if err != nil {
			log.Error(err)
		}
	}
	return &ConfigModelRegistry{
		Config: config,
	}
}

// ConfigModelRegistry is a registry of config models
type ConfigModelRegistry struct {
	Config Config
	mu     sync.RWMutex
}

// GetModel gets a model by name and version
func (r *ConfigModelRegistry) GetModel(name configmodel.Name, version configmodel.Version) (configmodel.ModelInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	path := r.getDescriptorFile(name, version)
	log.Debugf("Loading model definition '%s'", path)
	model, err := loadModel(path)
	if err != nil {
		log.Warnf("Failed loading model definition '%s': %v", path, err)
		return configmodel.ModelInfo{}, err
	}
	log.Infof("Loaded model definition '%s': %s", path, model)
	return model, nil
}

// ListModels lists models in the registry
func (r *ConfigModelRegistry) ListModels() ([]configmodel.ModelInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	log.Debugf("Loading models from '%s'", r.Config.Path)
	var modelFiles []string
	err := filepath.Walk(r.Config.Path, func(file string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(file, jsonExt) {
			modelFiles = append(modelFiles, file)
		}
		return nil
	})
	if err != nil {
		return nil, errors.NewInternal(err.Error())
	}

	var models []configmodel.ModelInfo
	for _, file := range modelFiles {
		log.Debugf("Loading model definition '%s'", file)
		model, err := loadModel(file)
		if err != nil {
			log.Warnf("Failed loading model definition '%s': %v", file, err)
		} else {
			log.Infof("Loaded model definition '%s': %s", file, model)
			models = append(models, model)
		}
	}
	return models, nil
}

// AddModel adds a model to the registry
func (r *ConfigModelRegistry) AddModel(model configmodel.ModelInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	log.Debugf("Adding model '%s/%s' to registry '%s'", model.Name, model.Version, r.Config.Path)
	bytes, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		log.Errorf("Adding model '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	path := r.getDescriptorFile(model.Name, model.Version)
	if err := ioutil.WriteFile(path, bytes, 0666); err != nil {
		log.Errorf("Adding model '%s/%s' failed: %v", model.Name, model.Version, err)
		return err
	}
	log.Infof("Model '%s/%s' added to registry '%s'", model.Name, model.Version, r.Config.Path)
	return nil
}

// RemoveModel removes a model from the registry
func (r *ConfigModelRegistry) RemoveModel(name configmodel.Name, version configmodel.Version) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	log.Debugf("Deleting model '%s/%s' from registry '%s'", name, version, r.Config.Path)
	path := r.getDescriptorFile(name, version)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		if err := os.Remove(path); err != nil {
			log.Errorf("Deleting model '%s/%s' failed: %v", name, version, err)
			return err
		}
	}
	log.Infof("Model '%s/%s' deleted from registry '%s'", name, version, r.Config.Path)
	return nil
}

func (r *ConfigModelRegistry) getDescriptorFile(name configmodel.Name, version configmodel.Version) string {
	return filepath.Join(r.Config.Path, fmt.Sprintf("%s-%s.json", name, version))
}

func loadModel(path string) (configmodel.ModelInfo, error) {
	var model configmodel.ModelInfo
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model, errors.NewNotFound("Model definition '%s' not found", path)
		}
		return model, errors.NewUnknown(err.Error())
	}
	err = json.Unmarshal(bytes, &model)
	if err != nil {
		return model, errors.NewInvalid(err.Error())
	}
	if model.Name == "" || model.Version == "" {
		return model, errors.NewInvalid("'%s' is not a valid model descriptor", path)
	}
	return model, nil
}

// GetPath :
func GetPath(dir, target, replace string) (string, error) {
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", errors.NewInternal(err.Error())
		}
		dir = cwd
	}
	if target == "" {
		target = defaultTarget
	}

	var path string
	var version string
	if replace == "" {
		if i := strings.Index(target, "@"); i >= 0 {
			path = target[:i]
			version = target[i+1:]
		} else {
			path = target
		}
	} else {
		if i := strings.Index(replace, "@"); i >= 0 {
			path = replace[:i]
			version = replace[i+1:]
		} else {
			path = replace
		}
	}

	encPath, err := module.EncodePath(path)
	if err != nil {
		return "", errors.NewInternal(err.Error())
	}

	elems := []string{dir, encPath}
	if version != "" {
		elems = append(elems, fmt.Sprintf("@%s", version))
	}
	return filepath.Join(elems...), nil
}
