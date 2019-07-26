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

package modelregistry

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
	"github.com/openconfig/ygot/ygot"
	log "k8s.io/klog"
	"plugin"
	"regexp"
	"strings"
)

// ReadOnlySubPathMap abstracts the read only subpath
type ReadOnlySubPathMap map[string]change.ValueType

// ReadOnlyPathMap abstracts the read only path
type ReadOnlyPathMap map[string]ReadOnlySubPathMap

// GlobalReadOnlyPaths abstracts the map for the read only paths
type GlobalReadOnlyPaths map[string]ReadOnlyPathMap

// ModelRegistry is the object for the saving information about device models
type ModelRegistry struct {
	ModelPlugins       map[string]ModelPlugin
	ModelReadOnlyPaths map[string]ReadOnlyPathMap
	LocationStore      map[string]string
}

// ModelPlugin is a set of methods that each model plugin should implement
type ModelPlugin interface {
	ModelData() (string, string, []*gnmi.ModelData, string)
	UnmarshalConfigValues(jsonTree []byte) (*ygot.ValidatedGoStruct, error)
	Validate(*ygot.ValidatedGoStruct, ...ygot.ValidationOption) error
	Schema() (map[string]*yang.Entry, error)
}

// RegisterModelPlugin adds an external model plugin to the model registry at startup
// or through the 'admin' gRPC interface. Once plugins are loaded they cannot be unloaded
func (registry *ModelRegistry) RegisterModelPlugin(moduleName string) (string, string, error) {
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
		return "", "", fmt.Errorf("symbol loaded from module %s is not a ModelPlugin",
			moduleName)
	}
	name, version, _, _ := modelPlugin.ModelData()
	modelName := utils.ToModelName(name, version)
	registry.ModelPlugins[modelName] = modelPlugin
	//Saving the model plugin name and library name in a distributed list for other instances to access it.
	registry.LocationStore[modelName] = moduleName
	modelschema, err := modelPlugin.Schema()
	if err != nil {
		log.Warning("Error loading schema from model plugin", modelName, err)
		return "", "", err
	}
	readOnlyPaths := ExtractReadOnlyPaths(modelschema["Device"], yang.TSUnset, "", "")
	registry.ModelReadOnlyPaths[modelName] = readOnlyPaths
	log.Info(registry.ModelReadOnlyPaths[modelName])
	log.Infof("Model %s %s loaded. %d read only paths", name, version,
		len(registry.ModelReadOnlyPaths[modelName]))
	return name, version, nil
}

// Capabilities returns an aggregated set of modelData in gNMI capabilities format
// with duplicates removed
func (registry *ModelRegistry) Capabilities() []*gnmi.ModelData {
	// Make a map - if we get duplicates overwrite them
	modelMap := make(map[string]*gnmi.ModelData)
	for _, model := range registry.ModelPlugins {
		_, _, modelItem, _ := model.ModelData()
		for _, mi := range modelItem {
			modelName := utils.ToModelName(mi.Name, mi.Version)
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

// ExtractReadOnlyPaths is a recursive function to extract a list of read only paths from a YGOT schema
func ExtractReadOnlyPaths(deviceEntry *yang.Entry, parentState yang.TriState, parentPath string,
	subpathPrefix string) ReadOnlyPathMap {
	readOnlyPaths := make(ReadOnlyPathMap)
	for _, dirEntry := range deviceEntry.Dir {
		itemPath := formatName(dirEntry, false, parentPath, subpathPrefix)
		if dirEntry.IsLeaf() {
			// No need to recurse
			if parentState == yang.TSFalse {
				leafMap, ok := readOnlyPaths[parentPath]
				if !ok {
					leafMap = make(ReadOnlySubPathMap)
					readOnlyPaths[parentPath] = leafMap
					t, err := toValueType(dirEntry.Type)
					if err != nil {
						log.Errorf(err.Error())
					}
					leafMap[strings.Replace(itemPath, parentPath, "", 1)] = t
				} else {
					t, err := toValueType(dirEntry.Type)
					if err != nil {
						log.Errorf(err.Error())
					}
					leafMap[strings.Replace(itemPath, parentPath, "", 1)] = t
				}
			} else if dirEntry.Config == yang.TSFalse {
				leafMap := make(ReadOnlySubPathMap)
				t, err := toValueType(dirEntry.Type)
				if err != nil {
					log.Errorf(err.Error())
				}
				leafMap["/"] = t
				readOnlyPaths[itemPath] = leafMap
			}
		} else if dirEntry.IsContainer() {
			if dirEntry.Config == yang.TSFalse || parentState == yang.TSFalse {
				subpathPfx := subpathPrefix
				if parentState == yang.TSFalse {
					subpathPfx = itemPath[len(parentPath):]
				}
				subPaths := ExtractReadOnlyPaths(dirEntry, yang.TSFalse, itemPath, subpathPfx)
				subPathsMap := make(ReadOnlySubPathMap)
				for _, v := range subPaths {
					for k, u := range v {
						subPathsMap[k] = u
					}
				}
				readOnlyPaths[itemPath] = subPathsMap
				continue
			}
			readOnlyPathsTemp := ExtractReadOnlyPaths(dirEntry, dirEntry.Config, itemPath, "")
			for k, v := range readOnlyPathsTemp {
				readOnlyPaths[k] = v
			}
		} else if dirEntry.IsList() {
			itemPath = formatName(dirEntry, true, parentPath, subpathPrefix)
			if dirEntry.Config == yang.TSFalse || parentState == yang.TSFalse {
				subpathPfx := subpathPrefix
				if parentState == yang.TSFalse {
					subpathPfx = itemPath[len(parentPath):]
				}
				subPaths := ExtractReadOnlyPaths(dirEntry, yang.TSFalse, parentPath, subpathPfx)
				subPathsMap := make(ReadOnlySubPathMap)
				for _, v := range subPaths {
					for k, u := range v {
						subPathsMap[k] = u
					}
				}
				readOnlyPaths[itemPath] = subPathsMap
				continue
			}
			readOnlyPathsTemp := ExtractReadOnlyPaths(dirEntry, dirEntry.Config, itemPath, "")
			for k, v := range readOnlyPathsTemp {
				readOnlyPaths[k] = v
			}
		}
	}
	return readOnlyPaths
}

// RemovePathIndices removes the index value from a path to allow it to be compared to a model path
func RemovePathIndices(path string) string {
	const indexPattern = `=.*?]`
	rname := regexp.MustCompile(indexPattern)
	indices := rname.FindAllStringSubmatch(path, -1)
	for _, i := range indices {
		path = strings.Replace(path, i[0], "=*]", 1)
	}
	return path
}

func formatName(dirEntry *yang.Entry, isList bool, parentPath string, subpathPrefix string) string {
	parentAndSubPath := parentPath
	if subpathPrefix != "/" {
		parentAndSubPath = fmt.Sprintf("%s%s", parentPath, subpathPrefix)
	}

	var name string
	if isList {
		name = fmt.Sprintf("%s/%s[%s=*]", parentAndSubPath, dirEntry.Name, dirEntry.Key)
	} else {
		name = fmt.Sprintf("%s/%s", parentAndSubPath, dirEntry.Name)
	}

	return name
}

//Paths extract the read only path up to the first read only container
func Paths(readOnly ReadOnlyPathMap) []string {
	keys := make([]string, 0, len(readOnly))
	for k := range readOnly {
		keys = append(keys, k)
	}
	return keys
}

func toValueType(entry *yang.YangType) (change.ValueType, error) {
	//TODO evaluate better types and error return
	switch entry.Name {
	case "int8":
		return change.ValueTypeINT, nil
	case "int16":
		return change.ValueTypeINT, nil
	case "int32":
		return change.ValueTypeINT, nil
	case "int64":
		return change.ValueTypeINT, nil
	case "uint8":
		return change.ValueTypeUINT, nil
	case "uint16":
		return change.ValueTypeUINT, nil
	case "uint32":
		return change.ValueTypeUINT, nil
	case "uint64":
		return change.ValueTypeUINT, nil
	case "counter64":
		return change.ValueTypeUINT, nil
	case "decimal64":
		return change.ValueTypeDECIMAL, nil
	case "string":
		return change.ValueTypeSTRING, nil
	case "boolean":
		return change.ValueTypeBOOL, nil
	case "bits":
		return change.ValueTypeBYTES, nil
	case "binary":
		return change.ValueTypeBYTES, nil
	case "empty":
		return change.ValueTypeEMPTY, nil
	case "enumeration":
		return change.ValueTypeSTRING, nil
	case "leafref":
		return change.ValueTypeSTRING, nil
	case "identityref":
		return change.ValueTypeSTRING, nil
	case "union":
		return change.ValueTypeSTRING, nil
	case "instance-identifier":
		return change.ValueTypeSTRING, nil
	default:
		return change.ValueTypeSTRING, nil
	}
}
