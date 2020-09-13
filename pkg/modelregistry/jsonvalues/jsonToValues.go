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

package jsonvalues

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"math"
	"sort"
	"strconv"
	"strings"
)

const (
	slash     = "/"
	equals    = "="
	bracketsq = "["
	brktclose = "]"
	colon     = ":"
)

type indexValue struct {
	name  string
	value *devicechange.TypedValue
	order int
}

// DecomposeJSONWithPaths - handling the decomposition and correction in one go
func DecomposeJSONWithPaths(prefixPath string, genericJSON []byte, ropaths modelregistry.ReadOnlyPathMap,
	rwpaths modelregistry.ReadWritePathMap) ([]*devicechange.PathValue, error) {

	var f interface{}
	err := json.Unmarshal(genericJSON, &f)
	if err != nil {
		return nil, err
	}
	values, err := extractValuesWithPaths(f, removeIndexNames(prefixPath), ropaths, rwpaths)
	if err != nil {
		return nil, fmt.Errorf("error decomposing JSON %v", err)
	}
	return values, nil
}

// extractValuesIntermediate recursively walks a JSON tree to create a flat set
// of paths and values.
func extractValuesWithPaths(f interface{}, parentPath string,
	modelROpaths modelregistry.ReadOnlyPathMap,
	modelRWpaths modelregistry.ReadWritePathMap) ([]*devicechange.PathValue, error) {

	changes := make([]*devicechange.PathValue, 0)

	switch value := f.(type) {
	case map[string]interface{}:
		mapChanges, err := handleMap(value, parentPath, modelROpaths, modelRWpaths)
		if err != nil {
			return nil, err
		}
		changes = append(changes, mapChanges...)

	case []interface{}:
		indexNames := indicesOfPath(modelROpaths, modelRWpaths, parentPath)
		// Iterate through to look for indexes first
		for idx, v := range value {
			indices := make([]indexValue, 0)
			nonIndexPaths := make([]string, 0)
			objs, err := extractValuesWithPaths(v, fmt.Sprintf("%s[%d]", parentPath, idx),
				modelROpaths, modelRWpaths)
			if err != nil {
				return nil, err
			}
			for _, obj := range objs {
				isIndex := false
				for i, idxName := range indexNames {
					if modelregistry.RemovePathIndices(obj.Path) == fmt.Sprintf("%s/%s", modelregistry.RemovePathIndices(parentPath), idxName) {
						indices = append(indices, indexValue{name: idxName, value: obj.Value, order: i})
						isIndex = true
						break
					}
				}
				if !isIndex {
					nonIndexPaths = append(nonIndexPaths, obj.Path)
				}
			}
			sort.Slice(indices, func(i, j int) bool {
				return indices[i].order < indices[j].order
			})
			// Now we have indices, need to go through again
			for _, obj := range objs {
				for _, nonIdxPath := range nonIndexPaths {
					if obj.Path == nonIdxPath {
						suffixLen := prefixLength(obj.Path, parentPath)
						obj.Path, err = replaceIndices(obj.Path, suffixLen, indices)
						if err != nil {
							return nil, fmt.Errorf("error replacing indices in %s %v", obj.Path, err)
						}
						changes = append(changes, obj)
					}
				}
			}
		}
	default:
		attr, err := handleAttribute(value, parentPath, modelROpaths, modelRWpaths)
		if err != nil {
			return nil, fmt.Errorf("error handling json attribute value %v", err)
		}
		if attr != nil {
			changes = append(changes, attr)
		}
	}

	return changes, nil
}

func handleMap(value map[string]interface{}, parentPath string,
	modelROpaths modelregistry.ReadOnlyPathMap,
	modelRWpaths modelregistry.ReadWritePathMap) ([]*devicechange.PathValue, error) {

	changes := make([]*devicechange.PathValue, 0)

	for key, v := range value {
		objs, err := extractValuesWithPaths(v, fmt.Sprintf("%s/%s", parentPath, stripNamespace(key)),
			modelROpaths, modelRWpaths)
		if err != nil {
			return nil, err
		}
		if len(objs) > 0 {
			firstType := (objs[0].Value).Type
			matching := true
			for _, o := range objs {
				// In a leaf list all value types have to match
				if o.Value.Type != firstType {
					// Not a leaf list
					matching = false
					break
				}
			}
			if !matching {
				changes = append(changes, objs...)
			} else {
				switch (objs[0].Value).Type {
				case devicechange.ValueType_LEAFLIST_INT:
					llVals := make([]int, 0)
					for _, obj := range objs {
						llI := (*devicechange.TypedLeafListInt64)(obj.Value)
						llVals = append(llVals, llI.List()...)
					}
					newCv := devicechange.PathValue{Path: objs[0].Path, Value: devicechange.NewLeafListInt64Tv(llVals)}
					changes = append(changes, &newCv)
				case devicechange.ValueType_LEAFLIST_STRING:
					llVals := make([]string, 0)
					for _, obj := range objs {
						llI := (*devicechange.TypedLeafListString)(obj.Value)
						llVals = append(llVals, llI.List()...)
					}
					newCv := devicechange.PathValue{Path: objs[0].Path, Value: devicechange.NewLeafListStringTv(llVals)}
					changes = append(changes, &newCv)
				case devicechange.ValueType_LEAFLIST_UINT:
					llVals := make([]uint, 0)
					for _, obj := range objs {
						llI := (*devicechange.TypedLeafListUint)(obj.Value)
						llVals = append(llVals, llI.List()...)
					}
					newCv := devicechange.PathValue{Path: objs[0].Path, Value: devicechange.NewLeafListUint64Tv(llVals)}
					changes = append(changes, &newCv)
				case devicechange.ValueType_LEAFLIST_BOOL:
					llVals := make([]bool, 0)
					for _, obj := range objs {
						llI := (*devicechange.TypedLeafListBool)(obj.Value)
						llVals = append(llVals, llI.List()...)
					}
					newCv := devicechange.PathValue{Path: objs[0].Path, Value: devicechange.NewLeafListBoolTv(llVals)}
					changes = append(changes, &newCv)
				case devicechange.ValueType_LEAFLIST_BYTES:
					llVals := make([][]byte, 0)
					for _, obj := range objs {
						llI := (*devicechange.TypedLeafListBytes)(obj.Value)
						llVals = append(llVals, llI.List()...)
					}
					newCv := devicechange.PathValue{Path: objs[0].Path, Value: devicechange.NewLeafListBytesTv(llVals)}
					changes = append(changes, &newCv)
				case devicechange.ValueType_LEAFLIST_DECIMAL:
					llDigits := make([]int64, 0)
					var llPrecision uint32
					for _, obj := range objs {
						llD := (*devicechange.TypedLeafListDecimal)(obj.Value)
						digitsList, precision := llD.List()
						llPrecision = precision
						llDigits = append(llDigits, digitsList...)
					}
					newCv := devicechange.PathValue{Path: objs[0].Path, Value: devicechange.NewLeafListDecimal64Tv(llDigits, llPrecision)}
					changes = append(changes, &newCv)
				case devicechange.ValueType_LEAFLIST_FLOAT:
					llVals := make([]float32, 0)
					for _, obj := range objs {
						llI := (*devicechange.TypedLeafListFloat)(obj.Value)
						llVals = append(llVals, llI.List()...)
					}
					newCv := devicechange.PathValue{Path: objs[0].Path, Value: devicechange.NewLeafListFloat32Tv(llVals)}
					changes = append(changes, &newCv)
				default:
					// Not a leaf list
					changes = append(changes, objs...)
				}
			}
		}
	}
	return changes, nil
}

func handleAttribute(value interface{}, parentPath string, modelROpaths modelregistry.ReadOnlyPathMap,
	modelRWpaths modelregistry.ReadWritePathMap) (*devicechange.PathValue, error) {

	var modeltype devicechange.ValueType
	var modelPath string
	var ok bool
	var pathElem *modelregistry.ReadWritePathElem
	var subPath *modelregistry.ReadOnlyAttrib
	var enum map[int]string
	var err error
	pathElem, modelPath, ok = findModelRwPathNoIndices(modelRWpaths, parentPath)
	if !ok {
		subPath, modelPath, ok = findModelRoPathNoIndices(modelROpaths, parentPath)
		if !ok {
			if modelROpaths == nil || modelRWpaths == nil {
				// If RO paths was not given - then we assume this missing path was a RO path
				return nil, nil
			}
			return nil, fmt.Errorf("unable to locate %s in model", parentPath)
		}
		modeltype = subPath.Datatype
		enum = subPath.Enum
	} else {
		modeltype = pathElem.ValueType
		enum = pathElem.Enum
	}
	var typedValue *devicechange.TypedValue
	switch modeltype {
	case devicechange.ValueType_STRING:
		var stringVal string
		switch valueTyped := value.(type) {
		case string:
			if len(enum) > 0 {
				stringVal, err = convertEnumIdx(valueTyped, enum, parentPath)
				if err != nil {
					return nil, err
				}
			} else {
				stringVal = valueTyped
			}
		case float64:
			if len(enum) > 0 {
				stringVal, err = convertEnumIdx(fmt.Sprintf("%g", valueTyped), enum, parentPath)
				if err != nil {
					return nil, err
				}
			} else {
				stringVal = fmt.Sprintf("%g", valueTyped)
			}
		case bool:
			stringVal = fmt.Sprintf("%v", value)
		}
		typedValue = devicechange.NewTypedValueString(stringVal)
	case devicechange.ValueType_BOOL:
		typedValue = devicechange.NewTypedValueBool(value.(bool))
	case devicechange.ValueType_UINT:
		var uintVal uint
		switch valueTyped := value.(type) {
		case string:
			intVal, err := strconv.ParseInt(valueTyped, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("error converting to %v %s", modeltype, valueTyped)
			}
			uintVal = uint(intVal)
		case float64:
			uintVal = uint(valueTyped)
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewTypedValueUint64(uintVal)
	case devicechange.ValueType_DECIMAL:
		var digits int64
		var precision uint32 = 6 // TODO should get this from the model (when it is populated in it)
		switch valueTyped := value.(type) {
		case float64:
			digits = int64(valueTyped * math.Pow(10, float64(precision)))
		case string:
			floatVal, err := strconv.ParseFloat(valueTyped, 64)
			if err != nil {
				return nil, fmt.Errorf("error converting string to float %v", err)
			}
			digits = int64(floatVal * math.Pow(10, float64(precision)))
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewTypedValueDecimal64(digits, precision)
	case devicechange.ValueType_BYTES:
		var dstBytes []byte
		switch valueTyped := value.(type) {
		case string:
			// Values should be base64
			dstBytes, err = base64.StdEncoding.DecodeString(valueTyped)
			if err != nil {
				return nil, fmt.Errorf("expected binary value as base64. error decoding %s as base64 %v", valueTyped, err)
			}
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewTypedValueBytes(dstBytes)
	default:
		typedValue, err = handleAttributeLeafList(modeltype, value)
		if err != nil {
			return nil, err
		}
	}
	return &devicechange.PathValue{Path: modelPath, Value: typedValue}, nil
}

// A continuation of handle attribute above
func handleAttributeLeafList(modeltype devicechange.ValueType,
	value interface{}) (*devicechange.TypedValue, error) {

	var typedValue *devicechange.TypedValue

	switch modeltype {
	case devicechange.ValueType_LEAFLIST_INT:
		var leafvalue int
		switch valueTyped := value.(type) {
		case float64:
			leafvalue = int(valueTyped)
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewLeafListInt64Tv([]int{leafvalue})
	case devicechange.ValueType_LEAFLIST_UINT:
		var leafvalue uint
		switch valueTyped := value.(type) {
		case float64:
			leafvalue = uint(valueTyped)
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewLeafListUint64Tv([]uint{leafvalue})
	case devicechange.ValueType_LEAFLIST_FLOAT:
		var leafvalue float32
		switch valueTyped := value.(type) {
		case float64:
			leafvalue = float32(valueTyped)
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewLeafListFloat32Tv([]float32{leafvalue})
	case devicechange.ValueType_LEAFLIST_STRING:
		var leafvalue string
		switch valueTyped := value.(type) {
		case string:
			leafvalue = valueTyped
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewLeafListStringTv([]string{leafvalue})
	case devicechange.ValueType_LEAFLIST_BOOL:
		var leafvalue bool
		switch valueTyped := value.(type) {
		case bool:
			leafvalue = valueTyped
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewLeafListBoolTv([]bool{leafvalue})
	case devicechange.ValueType_LEAFLIST_BYTES:
		var leafvalue []byte
		var err error
		switch valueTyped := value.(type) {
		case string:
			// Values should be base64
			leafvalue, err = base64.StdEncoding.DecodeString(valueTyped)
			if err != nil {
				return nil, fmt.Errorf("expected binary value as base64. error decoding %s as base64 %v", valueTyped, err)
			}
		default:
			return nil, fmt.Errorf("unhandled conversion to %v %s", modeltype, valueTyped)
		}
		typedValue = devicechange.NewLeafListBytesTv([][]byte{leafvalue})
	default:
		return nil, fmt.Errorf("unhandled conversion to %v", modeltype)
	}
	return typedValue, nil
}

func findModelRwPathNoIndices(modelRWpaths modelregistry.ReadWritePathMap,
	searchpath string) (*modelregistry.ReadWritePathElem, string, bool) {

	searchpathNoIndices := modelregistry.RemovePathIndices(searchpath)
	for path, value := range modelRWpaths {
		if modelregistry.RemovePathIndices(path) == searchpathNoIndices {
			pathWithNumericalIdx, err := insertNumericalIndices(path, searchpath)
			if err != nil {
				return nil, fmt.Sprintf("could not replace wildcards in model path with numerical ids %v", err), false
			}
			return &value, pathWithNumericalIdx, true
		}
	}
	return nil, "", false
}

func findModelRoPathNoIndices(modelROpaths modelregistry.ReadOnlyPathMap,
	searchpath string) (*modelregistry.ReadOnlyAttrib, string, bool) {

	searchpathNoIndices := modelregistry.RemovePathIndices(searchpath)
	for path, value := range modelROpaths {
		for subpath, subpathValue := range value {
			var fullpath string
			if subpath == "/" {
				fullpath = path
			} else {
				fullpath = fmt.Sprintf("%s%s", path, subpath)
			}
			if modelregistry.RemovePathIndices(fullpath) == searchpathNoIndices {
				pathWithNumericalIdx, err := insertNumericalIndices(fullpath, searchpath)
				if err != nil {
					return nil, fmt.Sprintf("could not replace wildcards in model path with numerical ids %v", err), false
				}
				return &subpathValue, pathWithNumericalIdx, true
			}
		}
	}
	return nil, "", false
}

// YGOT does not handle namespaces, so there is no point in us maintaining them
// They may come from the southbound or northbound in a JSON payload though, so
// we have to be able to deal with them
func stripNamespace(path string) string {
	pathParts := strings.Split(path, "/")
	for idx, pathPart := range pathParts {
		colonPos := strings.Index(pathPart, colon)
		if colonPos > 0 {
			pathParts[idx] = pathPart[colonPos+1:]
		}
	}
	return strings.Join(pathParts, "/")
}

// For RW paths
func indicesOfPath(modelROpaths modelregistry.ReadOnlyPathMap,
	modelRWpaths modelregistry.ReadWritePathMap, searchpath string) []string {

	searchpathNoIndices := modelregistry.RemovePathIndices(searchpath)
	// First search through the RW paths
	for path := range modelRWpaths {
		pathNoIndices := modelregistry.RemovePathIndices(path)
		// Find a short path
		if pathNoIndices[:strings.LastIndex(pathNoIndices, slash)] == searchpathNoIndices {
			return modelregistry.ExtractIndexNames(path)
		}
	}

	// If not found then search through the RO paths
	for path, value := range modelROpaths {
		for subpath := range value {
			var fullpath string
			if subpath == "/" {
				fullpath = path
			} else {
				fullpath = fmt.Sprintf("%s%s", path, subpath)
			}
			pathNoIndices := modelregistry.RemovePathIndices(fullpath)
			// Find a short path
			if pathNoIndices[:strings.LastIndex(pathNoIndices, slash)] == searchpathNoIndices {
				return modelregistry.ExtractIndexNames(fullpath)
			}
		}
	}

	return []string{}
}

func insertNumericalIndices(modelPath string, jsonPath string) (string, error) {
	jsonParts := strings.Split(jsonPath, slash)
	modelParts := strings.Split(modelPath, slash)
	if len(modelParts) != len(jsonParts) {
		return "", fmt.Errorf("strings must have the same number of / characters %d!=%d", len(modelParts), len(jsonParts))
	}
	for idx, jsonPart := range jsonParts {
		brktIdx := strings.LastIndex(jsonPart, bracketsq)
		if brktIdx > 0 {
			modelParts[idx] = strings.ReplaceAll(modelParts[idx], "*", jsonPart[brktIdx+1:len(jsonPart)-1])
		}
	}

	return strings.Join(modelParts, "/"), nil
}

func prefixLength(objPath string, parentPath string) int {
	objPathParts := strings.Split(objPath, "/")
	parentPathParts := strings.Split(parentPath, "/")
	return len(strings.Join(objPathParts[:len(parentPathParts)], "/"))
}

// There might not be an index for everything
func replaceIndices(path string, ignoreAfter int, indices []indexValue) (string, error) {
	ignored := path[ignoreAfter:]
	pathParts := strings.Split(path[:ignoreAfter], bracketsq)
	idxOffset := len(pathParts) - len(indices) - 1

	// Range in reverse
	for i := len(pathParts) - 1; i > 0; i-- {
		pathPart := pathParts[i]
		eqIdx := strings.LastIndex(pathPart, equals)
		if eqIdx > 0 {
			closeIdx := strings.LastIndex(pathPart, brktclose)
			idxName := pathPart[:eqIdx]
			var actualValue string
			if i-idxOffset-1 < 0 {
				continue
			}
			index := indices[i-idxOffset-1]
			if index.name != idxName {
				//continue
				return "", fmt.Errorf("unexpected index name %s", index.name)
			}
			switch index.value.Type {
			case devicechange.ValueType_STRING:
				actualValue = string(index.value.Bytes)
			case devicechange.ValueType_UINT, devicechange.ValueType_INT:
				actualValue = fmt.Sprintf("%d", (*devicechange.TypedUint64)(index.value).Uint())
			}
			pathParts[i] = fmt.Sprintf("%s=%s%s", idxName, actualValue, pathPart[closeIdx:])
		}
	}

	return fmt.Sprintf("%s%s", strings.Join(pathParts, bracketsq), ignored), nil
}

func convertEnumIdx(valueTyped string, enum map[int]string,
	parentPath string) (string, error) {
	var stringVal string
	for k, v := range enum {
		if v == valueTyped {
			stringVal = valueTyped
			break
		} else if fmt.Sprintf("%d", k) == valueTyped {
			stringVal = v
			break
		}
	}
	if stringVal == "" {
		enumOpts := make([]string, len(enum)*2)
		i := 0
		for k, v := range enum {
			enumOpts[i*2] = fmt.Sprintf("%d", k)
			enumOpts[i*2+1] = v
			i++
		}
		return "", fmt.Errorf("value %s for %s does not match any enumerated value %s",
			valueTyped, parentPath, strings.Join(enumOpts, ";"))
	}
	return stringVal, nil
}

// for a path like
// "/interfaces/interface[name=eth1]/subinterfaces/subinterface[index=120]/config/description",
// Remove the "name=" and "index="
func removeIndexNames(prefixPath string) string {
	splitPath := strings.Split(prefixPath, equals)
	for i, pathPart := range splitPath {
		if i < len(splitPath)-1 {
			lastBrktPos := strings.LastIndex(pathPart, bracketsq)
			splitPath[i] = pathPart[:lastBrktPos] + "["
		}
	}

	return strings.Join(splitPath, "")
}
