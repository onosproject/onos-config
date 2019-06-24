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

package manager

import (
	"fmt"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
	log "k8s.io/klog"
	"plugin"
)

// ModelPlugin is a set of methods that each model plugin should implement
type ModelPlugin interface {
	ModelData() (string, string, []*gnmi.ModelData, string)
	UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error)
	Validate(*ygot.ValidatedGoStruct, ...ygot.ValidationOption) error
}

// RegisterModelPlugin adds an external model plugin to the model registry at startup
// or through the 'admin' gRPC interface. Once plugins are loaded they cannot be unloaded
func (m *Manager) RegisterModelPlugin(moduleName string) (string, string, error) {
	log.Info("Loading module ", moduleName)
	modelPluginModule, err := plugin.Open(moduleName)
	if err != nil {
		log.Warning("Unable to load module ", moduleName)
		return "", "", err
	}
	symbolMP, err := modelPluginModule.Lookup("ModelPlugin")
	if err != nil {
		log.Warning("Unable to find ModelPlugin in module ", moduleName)
		return "", "", err
	}
	modelPlugin, ok := symbolMP.(ModelPlugin)
	if !ok {
		log.Warning("Unable to use ModelPlugin in ", moduleName)
		return "", "", fmt.Errorf("Symbol loaded from module %s is not a ModelPlugin",
			moduleName)
	}
	name, version, _, _ := modelPlugin.ModelData()
	modelName := toModelName(name, version)
	m.ModelRegistry[modelName] = modelPlugin
	return name, version, nil
}

// Capabilities returns an aggregated set of modelData in gNMI capabilities format
// with duplicates removed
func (m *Manager) Capabilities() []*gnmi.ModelData {
	// Make a map - if we get duplicates overwrite them
	modelMap := make(map[string]*gnmi.ModelData)
	for _, model := range m.ModelRegistry {
		_, _, modelItem, _ := model.ModelData()
		for _, mi := range modelItem {
			modelName := toModelName(mi.Name, mi.Version)
			modelMap[modelName] = mi
		}
	}

	outputList := make([]*gnmi.ModelData, len(modelMap))
	i := 0
	for _, modelItem := range modelMap {
		outputList[i] = modelItem
		i++
	}
	return outputList
}

func toModelName(name string, version string) string {
	return fmt.Sprintf("%s-%s", name, version)
}
