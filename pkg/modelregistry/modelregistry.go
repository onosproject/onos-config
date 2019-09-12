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
	"sort"
	"strings"
)

// PathMap is an interface that is implemented by ReadOnly- and ReadWrite- PathMaps
type PathMap interface {
	JustPaths() []string
}

// ReadOnlySubPathMap abstracts the read only subpath
type ReadOnlySubPathMap map[string]change.ValueType

// ReadOnlyPathMap abstracts the read only path
type ReadOnlyPathMap map[string]ReadOnlySubPathMap

// JustPaths extracts keys from a read only path map
func (ro ReadOnlyPathMap) JustPaths() []string {
	keys := make([]string, len(ro))
	i := 0
	for k := range ro {
		keys[i] = k
		i++
	}
	return keys
}

// ReadWritePathElem holds data about a leaf or container
type ReadWritePathElem struct {
	ValueType   change.ValueType
	Units       string
	Description string
	Mandatory   bool
	Default     string
	Range       []string
	Length      []string
}

// ReadWritePathMap is a map of ReadWrite paths a their metadata
type ReadWritePathMap map[string]ReadWritePathElem

// JustPaths extracts keys from a read write path map
func (rw ReadWritePathMap) JustPaths() []string {
	keys := make([]string, len(rw))
	i := 0
	for k := range rw {
		keys[i] = k
		i++
	}
	return keys
}

// ModelRegistry is the object for the saving information about device models
type ModelRegistry struct {
	ModelPlugins        map[string]ModelPlugin
	ModelReadOnlyPaths  map[string]ReadOnlyPathMap
	ModelReadWritePaths map[string]ReadWritePathMap
	LocationStore       map[string]string
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
	readOnlyPaths, readWritePaths := ExtractPaths(modelschema["Device"], yang.TSUnset, "", "")

	/////////////////////////////////////////////////////////////////////
	// Stratum - special case
	// It has 139 Read Only paths in its YANG but as of Aug'19 only 1 is
	// supported by the actual device - /interfaces/interface[name=*]/state
	// Either the YANG should be adjusted or the device should implement
	// the paths. As a workaround just add the working path here
	// In addition Stratum does not fully support wildcards, and so calling this
	// path will only retrieve the ifindex and name under this branch - other paths
	// will have to be called explicitly by their interface name without wildcard
	/////////////////////////////////////////////////////////////////////
	if name == "Stratum" && version == "1.0.0" {
		stratumIfRwPaths := make(ReadWritePathMap)
		const StratumIfRwPaths = "/interfaces/interface[name=*]/config"
		stratumIfRwPaths[StratumIfRwPaths+"/loopback-mode"] = readWritePaths[StratumIfRwPaths+"/loopback-mode"]
		stratumIfRwPaths[StratumIfRwPaths+"/name"] = readWritePaths[StratumIfRwPaths+"/name"]
		stratumIfRwPaths[StratumIfRwPaths+"/id"] = readWritePaths[StratumIfRwPaths+"/id"]
		stratumIfRwPaths[StratumIfRwPaths+"/health-indicator"] = readWritePaths[StratumIfRwPaths+"/health-indicator"]
		stratumIfRwPaths[StratumIfRwPaths+"/mtu"] = readWritePaths[StratumIfRwPaths+"/mtu"]
		stratumIfRwPaths[StratumIfRwPaths+"/description"] = readWritePaths[StratumIfRwPaths+"/description"]
		stratumIfRwPaths[StratumIfRwPaths+"/type"] = readWritePaths[StratumIfRwPaths+"/type"]
		stratumIfRwPaths[StratumIfRwPaths+"/tpid"] = readWritePaths[StratumIfRwPaths+"/tpid"]
		stratumIfRwPaths[StratumIfRwPaths+"/enabled"] = readWritePaths[StratumIfRwPaths+"/enabled"]
		registry.ModelReadWritePaths[modelName] = stratumIfRwPaths

		stratumIfPath := make(ReadOnlyPathMap)
		const StratumIfPath = "/interfaces/interface[name=*]/state"
		stratumIfPath[StratumIfPath] = readOnlyPaths[StratumIfPath]
		registry.ModelReadOnlyPaths[modelName] = stratumIfPath
		log.Infof("Model %s %s loaded. HARDCODED to 1 readonly path."+
			"%d read only paths. %d read write paths", name, version,
			len(registry.ModelReadOnlyPaths[modelName]), len(registry.ModelReadWritePaths[modelName]))
		return name, version, nil
	}

	registry.ModelReadOnlyPaths[modelName] = readOnlyPaths
	registry.ModelReadWritePaths[modelName] = readWritePaths
	log.Infof("Model %s %s loaded. %d read only paths. %d read write paths", name, version,
		len(registry.ModelReadOnlyPaths[modelName]), len(registry.ModelReadWritePaths[modelName]))
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
			if err != nil {
				log.Errorf(err.Error())
			}
			if parentState == yang.TSFalse {
				leafMap, ok := readOnlyPaths[parentPath]
				if !ok {
					leafMap = make(ReadOnlySubPathMap)
					readOnlyPaths[parentPath] = leafMap
					leafMap[strings.Replace(itemPath, parentPath, "", 1)] = t
				} else {
					leafMap[strings.Replace(itemPath, parentPath, "", 1)] = t
				}
			} else if dirEntry.Config == yang.TSFalse {
				leafMap := make(ReadOnlySubPathMap)
				leafMap["/"] = t
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
			}
		} else {
			log.Errorf("Unexpected type of leaf for %s", itemPath)
		}
	}
	return readOnlyPaths, readWritePaths
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
		//have to ensure index order is consistent where there's more than one
		keyParts := strings.Split(dirEntry.Key, " ")
		sort.Slice(keyParts, func(i, j int) bool {
			return keyParts[i] < keyParts[j]
		})
		name = fmt.Sprintf("%s/%s[%s=*]", parentAndSubPath, dirEntry.Name, strings.Join(keyParts, " "))
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

func toValueType(entry *yang.YangType, isLeafList bool) (change.ValueType, error) {
	//TODO evaluate better types and error return
	switch entry.Name {
	case "int8", "int16", "int32", "int64":
		if isLeafList {
			return change.ValueTypeLeafListINT, nil
		}
		return change.ValueTypeINT, nil
	case "uint8", "uint16", "uint32", "uint64", "counter64":
		if isLeafList {
			return change.ValueTypeLeafListUINT, nil
		}
		return change.ValueTypeUINT, nil
	case "decimal64":
		if isLeafList {
			return change.ValueTypeLeafListDECIMAL, nil
		}
		return change.ValueTypeDECIMAL, nil
	case "string", "enumeration", "leafref", "identityref", "union", "instance-identifier":
		if isLeafList {
			return change.ValueTypeLeafListSTRING, nil
		}
		return change.ValueTypeSTRING, nil
	case "boolean":
		if isLeafList {
			return change.ValueTypeLeafListBOOL, nil
		}
		return change.ValueTypeBOOL, nil
	case "bits", "binary":
		if isLeafList {
			return change.ValueTypeLeafListBYTES, nil
		}
		return change.ValueTypeBYTES, nil
	case "empty":
		return change.ValueTypeEMPTY, nil
	default:
		return change.ValueTypeSTRING, nil
	}
}
