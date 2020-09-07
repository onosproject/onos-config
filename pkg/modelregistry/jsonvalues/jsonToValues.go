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
	"encoding/binary"
	"encoding/json"
	"fmt"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const matchOnIndex = `(\[.*?]).*?`
const (
	slash     = "/"
	equals    = "="
	bracketsq = "["
	brktclose = "]"
	colon     = ":"
)

var rOnIndex = regexp.MustCompile(matchOnIndex)

type indexValue struct {
	name  string
	value *devicechange.TypedValue
}

// DecomposeJSONWithPaths - handling the decomposition and correction in one go
func DecomposeJSONWithPaths(genericJSON []byte, ropaths modelregistry.ReadOnlyPathMap,
	rwpaths modelregistry.ReadWritePathMap) ([]*devicechange.PathValue, error) {

	var f interface{}
	err := json.Unmarshal(genericJSON, &f)
	if err != nil {
		return nil, err
	}
	values, err := extractValuesWithPaths(f, "", ropaths, rwpaths)
	if err != nil {
		return nil, fmt.Errorf("error decomposing JSON %v", err)
	}
	return values, nil
}

// extractValuesIntermediate recursively walks a JSON tree to create a flat set
// of paths and values.
// Note: it is not possible to find indices of lists and accurate devicechange directly
// from json - for that the RO Paths must be consulted
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
				for _, idxName := range indexNames {
					if removePathIndices(obj.Path) == fmt.Sprintf("%s/%s", removePathIndices(parentPath), idxName) {
						indices = append(indices, indexValue{name: idxName, value: obj.Value})
						isIndex = true
						continue
					}
				}
				if !isIndex {
					nonIndexPaths = append(nonIndexPaths, obj.Path)
				}
			}
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
		changes = append(changes, attr)
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
	return changes, nil
}

func handleAttribute(value interface{}, parentPath string, modelROpaths modelregistry.ReadOnlyPathMap,
	modelRWpaths modelregistry.ReadWritePathMap) (*devicechange.PathValue, error) {

	var modeltype devicechange.ValueType
	var modelPath string
	var ok bool
	var pathElem *modelregistry.ReadWritePathElem
	var subPath *modelregistry.ReadOnlyAttrib
	var err error
	pathElem, modelPath, ok = findModelRwPathNoIndices(modelRWpaths, parentPath)
	if !ok {
		subPath, modelPath, ok = findModelRoPathNoIndices(modelROpaths, parentPath)
		if !ok {
			return nil, fmt.Errorf("unable to locate %s in model", parentPath)
		}
		modeltype = subPath.Datatype
	} else {
		modeltype = pathElem.ValueType
	}
	var typedValue *devicechange.TypedValue
	switch modeltype {
	case devicechange.ValueType_STRING:
		var stringVal string
		switch valueTyped := value.(type) {
		case string:
			stringVal = valueTyped
		case float64:
			stringVal = fmt.Sprintf("%g", value)
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
		typedValue, err = handleAttributeLeafList(modelPath, modeltype, value)
		if err != nil {
			return nil, err
		}
	}
	return &devicechange.PathValue{Path: modelPath, Value: typedValue}, nil
}

// A continuation of handle attribute above
func handleAttributeLeafList(modelPath string, modeltype devicechange.ValueType,
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

	searchpathNoIndices := removePathIndices(searchpath)
	for path, value := range modelRWpaths {
		pathNoIndices := removePathIndices(path)
		if pathNoIndices == searchpathNoIndices {
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

	searchpathNoIndices := removePathIndices(searchpath)
	for path, value := range modelROpaths {
		for subpath, subpathValue := range value {
			var fullpath string
			if subpath == "/" {
				fullpath = path
			} else {
				fullpath = fmt.Sprintf("%s%s", path, subpath)
			}
			pathNoIndices := removePathIndices(fullpath)
			if pathNoIndices == searchpathNoIndices {
				return &subpathValue, fullpath, true
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

	searchpathNoIndices := removePathIndices(searchpath)
	for path := range modelRWpaths {
		pathNoIndices := removePathIndices(path)
		// Find a short path
		if pathNoIndices[:strings.LastIndex(pathNoIndices, slash)] == searchpathNoIndices {
			return extractIndexNames(path)
		}
	}

	for path, value := range modelROpaths {
		for subpath := range value {
			var fullpath string
			if subpath == "/" {
				fullpath = path
			} else {
				fullpath = fmt.Sprintf("%s%s", path, subpath)
			}
			pathNoIndices := removePathIndices(fullpath)
			// Find a short path
			if pathNoIndices[:strings.LastIndex(pathNoIndices, slash)] == searchpathNoIndices {
				return extractIndexNames(fullpath)
			}
		}
	}

	return []string{}
}

func removePathIndices(path string) string {
	jsonMatches := rOnIndex.FindAllStringSubmatch(path, -1)
	for _, m := range jsonMatches {
		path = strings.ReplaceAll(path, m[1], "")
	}
	return path
}

func extractIndexNames(path string) []string {
	indexNames := make([]string, 0)
	jsonMatches := rOnIndex.FindAllStringSubmatch(path, -1)
	for _, m := range jsonMatches {
		idxName := m[1][1:strings.LastIndex(m[1], "=")]
		indexNames = append(indexNames, idxName)
	}
	return indexNames
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
				continue
				//return "", fmt.Errorf("unexpected index name %s", index.name)
			}
			switch index.value.Type {
			case devicechange.ValueType_STRING:
				actualValue = string(index.value.Bytes)
			case devicechange.ValueType_UINT, devicechange.ValueType_INT:
				actualValue = fmt.Sprintf("%d", binary.LittleEndian.Uint64(index.value.Bytes))
			}
			pathParts[i] = fmt.Sprintf("%s=%s%s", idxName, actualValue, pathPart[closeIdx:])
		}
	}

	return fmt.Sprintf("%s%s", strings.Join(pathParts, bracketsq), ignored), nil
}
