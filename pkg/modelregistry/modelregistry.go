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
	"github.com/onosproject/onos-config-model-go/pkg/model"
	"github.com/onosproject/onos-config-model-go/pkg/registry"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"regexp"
	"sort"
	"strings"
	"sync"

	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/goyang/pkg/yang"
)

var log = logging.GetLogger("modelregistry")

const registryPath = "/root/registry"

// PathMap is an interface that is implemented by ReadOnly- and ReadWrite- PathMaps
type PathMap interface {
	JustPaths() []string
	TypeForPath(path string) (devicechange.ValueType, error)
}

// MatchOnIndex - regexp to find indices in paths names
const MatchOnIndex = `(\[.*?]).*?`

// ReadOnlyAttrib is the known metadata about a Read Only leaf
type ReadOnlyAttrib struct {
	Datatype    devicechange.ValueType
	Description string
	Units       string
	Enum        map[int]string
}

// ReadOnlySubPathMap abstracts the read only subpath
type ReadOnlySubPathMap map[string]ReadOnlyAttrib

// ReadOnlyPathMap abstracts the read only path
type ReadOnlyPathMap map[string]ReadOnlySubPathMap

var rOnIndex = regexp.MustCompile(MatchOnIndex)

// JustPaths extracts keys from a read only path map
func (ro ReadOnlyPathMap) JustPaths() []string {
	keys := make([]string, 0)
	for k, subPaths := range ro {
		for k1 := range subPaths {
			if k1 == "/" {
				keys = append(keys, k)
			} else {
				keys = append(keys, k+k1)
			}
		}
	}
	return keys
}

// TypeForPath finds the type from the model for a particular path
func (ro ReadOnlyPathMap) TypeForPath(path string) (devicechange.ValueType, error) {
	for k, subPaths := range ro {
		for k1, sp := range subPaths {
			if k1 == "/" {
				if k == path {
					return sp.Datatype, nil
				}
			} else {
				if k+k1 == path {
					return sp.Datatype, nil
				}
			}
		}
	}
	return devicechange.ValueType_EMPTY, fmt.Errorf("path %s not found in RO paths of model", path)
}

// ReadWritePathElem holds data about a leaf or container
type ReadWritePathElem struct {
	ValueType   devicechange.ValueType
	Units       string
	Description string
	Mandatory   bool
	Default     string
	Range       []string
	Length      []string
	Enum        map[int]string
}

// ReadWritePathMap is a map of ReadWrite paths a their metadata
type ReadWritePathMap map[string]ReadWritePathElem

// JustPaths extracts keys from a read write path map
// expandSubPaths is not relevant for RW paths
func (rw ReadWritePathMap) JustPaths() []string {
	keys := make([]string, len(rw))
	i := 0
	for k := range rw {
		keys[i] = k
		i++
	}
	return keys
}

// TypeForPath finds the type from the model for a particular path
func (rw ReadWritePathMap) TypeForPath(path string) (devicechange.ValueType, error) {
	for k, elem := range rw {
		if k == path {
			return elem.ValueType, nil
		}
	}
	return devicechange.ValueType_EMPTY, fmt.Errorf("path %s not found in RW paths of model", path)
}

// ModelPlugin is a config model
type ModelPlugin struct {
	Info           model.ConfigModelInfo
	Model          model.ConfigModel
	ReadOnlyPaths  ReadOnlyPathMap
	ReadWritePaths ReadWritePathMap
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		registry: registry.NewRegistry(registry.Config{Path: registryPath}),
		plugins:  make(map[string]*ModelPlugin),
	}
}

// ModelRegistry is a registry of config models
type ModelRegistry struct {
	registry *registry.ConfigModelRegistry
	plugins  map[string]*ModelPlugin
	mu       sync.RWMutex
}

// GetPlugins gets a list of model plugins
func (r *ModelRegistry) GetPlugins() ([]*ModelPlugin, error) {
	if err := r.loadPlugins(); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	plugins := make([]*ModelPlugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}

// GetPlugin gets a model plugin by name
func (r *ModelRegistry) GetPlugin(name string) (*ModelPlugin, error) {
	plugin, err := r.getPlugin(name)
	if err == nil {
		return plugin, nil
	} else if !errors.IsNotFound(err) {
		return nil, err
	}

	if err := r.loadPlugins(); err != nil {
		return nil, err
	}
	return r.getPlugin(name)
}

// getPlugin gets a model plugin by name
func (r *ModelRegistry) getPlugin(name string) (*ModelPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, ok := r.plugins[name]
	if ok {
		return plugin, nil
	}
	return nil, errors.NewNotFound("Model plugin '%s' not found", name)
}

// loadPlugins loads the available model plugins from the model registry
func (r *ModelRegistry) loadPlugins() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	modelInfos, err := r.registry.ListModels()
	if err != nil {
		return err
	}

	for _, modelInfo := range modelInfos {
		modelName := utils.ToModelName(devicetype.Type(modelInfo.Name), devicetype.Version(modelInfo.Version))
		if _, ok := r.plugins[modelName]; !ok {
			model, err := model.Load(modelInfo.Plugin.File)
			if err != nil {
				return err
			}

			schema, err := model.Schema()
			if err != nil {
				return err
			}

			readOnlyPaths, readWritePaths := ExtractPaths(schema["Device"], yang.TSUnset, "", "")
			plugin := &ModelPlugin{
				Info:           modelInfo,
				Model:          model,
				ReadOnlyPaths:  readOnlyPaths,
				ReadWritePaths: readWritePaths,
			}
			r.plugins[modelName] = plugin
		}
	}
	return nil
}

// Capabilities returns an aggregated set of modelData in gNMI capabilities format
// with duplicates removed
func (r *ModelRegistry) Capabilities() ([]*gnmi.ModelData, error) {
	if err := r.loadPlugins(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Make a map - if we get duplicates overwrite them
	modelMap := make(map[string]*gnmi.ModelData)
	for _, plugin := range r.plugins {
		for _, modelData := range plugin.Model.Data() {
			modelName := utils.ToModelName(devicetype.Type(modelData.Name), devicetype.Version(modelData.Version))
			modelMap[modelName] = modelData
		}
	}

	models := make([]*gnmi.ModelData, 0, len(modelMap))
	for _, modelData := range modelMap {
		models = append(models, modelData)
	}
	return models, nil
}

// ExtractPaths is a recursive function to extract a list of read only paths from a YGOT schema
func ExtractPaths(deviceEntry *yang.Entry, parentState yang.TriState, parentPath string,
	subpathPrefix string) (ReadOnlyPathMap, ReadWritePathMap) {
	readOnlyPaths := make(ReadOnlyPathMap)
	readWritePaths := make(ReadWritePathMap)
	for _, dirEntry := range deviceEntry.Dir {
		itemPath := formatName(dirEntry, false, parentPath, subpathPrefix)
		if dirEntry.IsLeaf() || dirEntry.IsLeafList() {
			// No need to recurse
			t, err := toValueType(dirEntry.Type, dirEntry.IsLeafList())
			tObj := ReadOnlyAttrib{Datatype: t, Description: dirEntry.Description, Units: dirEntry.Units}
			if err != nil {
				log.Errorf(err.Error())
			}
			var enum map[int]string
			if dirEntry.Type.Kind == yang.Yidentityref {
				enum = handleIdentity(dirEntry.Type)
			}
			tObj.Enum = enum
			if parentState == yang.TSFalse {
				leafMap, ok := readOnlyPaths[parentPath]
				if !ok {
					leafMap = make(ReadOnlySubPathMap)
					readOnlyPaths[parentPath] = leafMap
				}
				leafMap[strings.Replace(itemPath, parentPath, "", 1)] = tObj
			} else if dirEntry.Config == yang.TSFalse {
				leafMap := make(ReadOnlySubPathMap)
				leafMap["/"] = tObj
				readOnlyPaths[itemPath] = leafMap
			} else {
				ranges := make([]string, 0)
				for _, r := range dirEntry.Type.Range {
					ranges = append(ranges, fmt.Sprintf("%v", r))
				}
				lengths := make([]string, 0)
				for _, l := range dirEntry.Type.Length {
					lengths = append(lengths, fmt.Sprintf("%v", l))
				}
				rwElem := ReadWritePathElem{
					ValueType:   t,
					Description: dirEntry.Description,
					Mandatory:   dirEntry.Mandatory == yang.TSTrue,
					Units:       dirEntry.Units,
					Default:     dirEntry.Default,
					Range:       ranges,
					Length:      lengths,
					Enum:        enum,
				}
				readWritePaths[itemPath] = rwElem
			}
		} else if dirEntry.IsContainer() {
			if dirEntry.Config == yang.TSFalse || parentState == yang.TSFalse {
				subpathPfx := subpathPrefix
				if parentState == yang.TSFalse {
					subpathPfx = itemPath[len(parentPath):]
				}
				subPaths, _ := ExtractPaths(dirEntry, yang.TSFalse, itemPath, subpathPfx)
				subPathsMap := make(ReadOnlySubPathMap)
				for _, v := range subPaths {
					for k, u := range v {
						subPathsMap[k] = u
					}
				}
				readOnlyPaths[itemPath] = subPathsMap
				continue
			}
			readOnlyPathsTemp, readWritePathTemp := ExtractPaths(dirEntry, dirEntry.Config, itemPath, "")
			for k, v := range readOnlyPathsTemp {
				readOnlyPaths[k] = v
			}
			for k, v := range readWritePathTemp {
				readWritePaths[k] = v
			}
		} else if dirEntry.IsList() {
			itemPath = formatName(dirEntry, true, parentPath, subpathPrefix)
			if dirEntry.Config == yang.TSFalse || parentState == yang.TSFalse {
				subpathPfx := subpathPrefix
				if parentState == yang.TSFalse {
					subpathPfx = itemPath[len(parentPath):]
				}
				subPaths, _ := ExtractPaths(dirEntry, yang.TSFalse, parentPath, subpathPfx)
				subPathsMap := make(ReadOnlySubPathMap)
				for _, v := range subPaths {
					for k, u := range v {
						subPathsMap[k] = u
					}
				}
				readOnlyPaths[itemPath] = subPathsMap
				continue
			}
			readOnlyPathsTemp, readWritePathsTemp := ExtractPaths(dirEntry, dirEntry.Config, itemPath, "")
			for k, v := range readOnlyPathsTemp {
				readOnlyPaths[k] = v
			}
			for k, v := range readWritePathsTemp {
				readWritePaths[k] = v
				// Need to copy the index of the list across to the RO list too
				roIdxName := k[:strings.LastIndex(k, "/")]
				roIdxSubPath := k[strings.LastIndex(k, "/"):]
				indices := ExtractIndexNames(itemPath[strings.LastIndex(itemPath, "/"):])
				kIsIdxAttr := false
				for _, idx := range indices {
					if roIdxSubPath == fmt.Sprintf("/%s", idx) {
						kIsIdxAttr = true
					}
				}
				if roIdxName == itemPath && kIsIdxAttr {
					roIdx := ReadOnlyAttrib{
						Datatype:    v.ValueType,
						Description: v.Description,
						Units:       v.Units,
					}
					readOnlyPaths[roIdxName] = make(map[string]ReadOnlyAttrib)
					readOnlyPaths[roIdxName][roIdxSubPath] = roIdx
				}
			}

		} else if dirEntry.IsChoice() || dirEntry.IsCase() {
			// Recurse down through Choice and Case
			readOnlyPathsTemp, readWritePathsTemp := ExtractPaths(dirEntry, dirEntry.Config, parentPath, "")
			for k, v := range readOnlyPathsTemp {
				readOnlyPaths[k] = v
			}
			for k, v := range readWritePathsTemp {
				readWritePaths[k] = v
			}
		} else {
			log.Errorf("Unexpected type of leaf for %s %v", itemPath, dirEntry)
		}
	}
	return readOnlyPaths, readWritePaths
}

// RemovePathIndices removes the index value from a path to allow it to be compared to a model path
func RemovePathIndices(path string) string {
	indices := rOnIndex.FindAllStringSubmatch(path, -1)
	for _, i := range indices {
		path = strings.Replace(path, i[0], "", 1)
	}
	return path
}

// ExtractIndexNames - get an ordered array of index names
func ExtractIndexNames(path string) []string {
	indexNames := make([]string, 0)
	jsonMatches := rOnIndex.FindAllStringSubmatch(path, -1)
	for _, m := range jsonMatches {
		idxName := m[1][1:strings.LastIndex(m[1], "=")]
		indexNames = append(indexNames, idxName)
	}
	return indexNames
}

func formatName(dirEntry *yang.Entry, isList bool, parentPath string, subpathPrefix string) string {
	parentAndSubPath := parentPath
	if subpathPrefix != "/" {
		parentAndSubPath = fmt.Sprintf("%s%s", parentPath, subpathPrefix)
	}

	var name string
	if isList {
		//have to ensure index order is consistent where there's more than one
		keyParts := strings.Split(dirEntry.Key, " ")
		sort.Slice(keyParts, func(i, j int) bool {
			return keyParts[i] < keyParts[j]
		})
		name = fmt.Sprintf("%s/%s", parentAndSubPath, dirEntry.Name)
		for _, k := range keyParts {
			name += fmt.Sprintf("[%s=*]", k)
		}
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

//PathsRW extract the read write path
func PathsRW(rwPathMap ReadWritePathMap) []string {
	keys := make([]string, 0, len(rwPathMap))
	for k := range rwPathMap {
		keys = append(keys, k)
	}
	return keys
}

func toValueType(entry *yang.YangType, isLeafList bool) (devicechange.ValueType, error) {
	//TODO evaluate better devicechange and error return
	switch entry.Name {
	case "int8", "int16", "int32", "int64":
		if isLeafList {
			return devicechange.ValueType_LEAFLIST_INT, nil
		}
		return devicechange.ValueType_INT, nil
	case "uint8", "uint16", "uint32", "uint64", "counter64":
		if isLeafList {
			return devicechange.ValueType_LEAFLIST_UINT, nil
		}
		return devicechange.ValueType_UINT, nil
	case "decimal64":
		if isLeafList {
			return devicechange.ValueType_LEAFLIST_DECIMAL, nil
		}
		return devicechange.ValueType_DECIMAL, nil
	case "string", "enumeration", "leafref", "identityref", "union", "instance-identifier":
		if isLeafList {
			return devicechange.ValueType_LEAFLIST_STRING, nil
		}
		return devicechange.ValueType_STRING, nil
	case "boolean":
		if isLeafList {
			return devicechange.ValueType_LEAFLIST_BOOL, nil
		}
		return devicechange.ValueType_BOOL, nil
	case "bits", "binary":
		if isLeafList {
			return devicechange.ValueType_LEAFLIST_BYTES, nil
		}
		return devicechange.ValueType_BYTES, nil
	case "empty":
		return devicechange.ValueType_EMPTY, nil
	default:
		return devicechange.ValueType_STRING, nil
	}
}

func handleIdentity(yangType *yang.YangType) map[int]string {
	identityMap := make(map[int]string)
	identityMap[0] = "UNSET"
	for i, val := range yangType.IdentityBase.Values {
		identityMap[i+1] = val.Name
	}
	return identityMap
}
