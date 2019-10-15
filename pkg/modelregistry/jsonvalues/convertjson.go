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
	"fmt"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	types "github.com/onosproject/onos-config/pkg/types/change/device"
	"regexp"
	"sort"
	"strings"
)

const matchOnIndex = `(\[.*?]).*?`
const matchNamespace = `(/.*?:).*?`

type indexEntry struct {
	path string
	key  []string
}

// CorrectJSONPaths takes configuration values extracted from a json, of which we are not sure about the type and,
// through the model plugin assigns them the correct type according to the YANG model provided, returning an
// updated set of configuration values.
func CorrectJSONPaths(jsonBase string, jsonPathValues []*types.PathValue,
	paths modelregistry.PathMap, stripNamespaces bool) ([]*types.PathValue, error) {

	correctedPathValues := make([]*types.PathValue, 0)
	rOnIndex := regexp.MustCompile(matchOnIndex)
	rOnNamespace := regexp.MustCompile(matchNamespace)
	indexMap := make(map[string][]string)

	for _, jsonPathValue := range jsonPathValues {
		jsonPathStr := fmt.Sprintf("%s%s", jsonBase, jsonPathValue.Path)
		if stripNamespaces {
			nsMatches := rOnNamespace.FindAllStringSubmatch(jsonPathStr, -1)
			for _, ns := range nsMatches {
				jsonPathStr = strings.Replace(jsonPathStr, ns[1], "/", 1)
			}
		}
		jsonPathValue.Path = jsonPathStr

		jsonMatches := rOnIndex.FindAllStringSubmatch(jsonPathStr, -1)
		jsonPathWildIndex := jsonPathStr
		jsonPathIdx := jsonPathStr
		for idx, m := range jsonMatches {
			jsonPathWildIndex = strings.Replace(jsonPathWildIndex, m[1], "[*]", 1)
			if idx == len(jsonMatches)-1 { //Last one only
				jsonPathIdx = strings.Replace(jsonPathIdx, m[1]+"/", "[", 1)
			} else {
				jsonPathIdx = strings.Replace(jsonPathIdx, m[1], "[*]", 1)
			}
		}

		for _, modelPath := range paths.JustPaths() {
			modelMatches := rOnIndex.FindAllStringSubmatch(modelPath, -1)
			modelPathWildIndex := modelPath
			modelPathOnlyLastIndex := modelPath
			modelPathOnlySecondIndex := modelPath
			modelLeafAfterIndex := ""
			modelLastIndices := make([]string, 0) // There could be multiple keys
			for idx, m := range modelMatches {
				modelPathWildIndex = strings.Replace(modelPathWildIndex, m[1], "[*]", 1)
				modelLastIndices = extractIndices(m[1])
				if len(modelLastIndices) == 2 {
					modelPathOnlySecondIndex = strings.Replace(modelPathOnlyLastIndex, modelLastIndices[0][1:]+" ", "", 1)
				}
				if idx != len(modelMatches)-1 { // Not the last one
					modelPathOnlyLastIndex = strings.Replace(modelPathOnlyLastIndex, m[1], "[*]", 1)
				} else {
					modelLeafAfterIndex = modelPath[strings.LastIndex(modelPath, "]")+1:]
				}
			}
			// RW paths include an entry for the index - should be ignored
			ignoreLeaf := false
			for _, li := range modelLastIndices {
				if li == modelLeafAfterIndex {
					ignoreLeaf = true // Need to get the continue to outer for
					continue
				}
			}
			if ignoreLeaf {
				continue
			}

			if modelPath == jsonPathStr {
				newTypeValue, err := assignModelType(paths, modelPath, jsonPathValue)
				if err != nil {
					return nil, err
				}
				jsonPathValue.Value = newTypeValue
				err = types.IsPathValid(jsonPathValue.Path)
				if err != nil {
					return nil, fmt.Errorf("invalid value %s", err)
				}
				correctedPathValues = append(correctedPathValues, jsonPathValue)
				break

			} else if hasPrefixMultipleIdx(jsonPathWildIndex, modelPathWildIndex) {
				newTypeValue, err := assignModelType(paths, modelPath, jsonPathValue)
				if err != nil {
					return nil, err
				}
				cv := types.PathValue{
					Path:  jsonPathStr,
					Value: newTypeValue,
				}
				err = types.IsPathValid(cv.Path)
				if err != nil {
					return nil, fmt.Errorf("invalid value %s", err)
				}
				correctedPathValues = append(correctedPathValues, &cv)
				break
			} else if hasPrefixMultipleIdx(modelPathOnlyLastIndex, jsonPathIdx) ||
				hasPrefixMultipleIdx(modelPathOnlySecondIndex, jsonPathIdx) {
				indexName := jsonPathIdx[strings.LastIndex(jsonPathIdx, "[")+1:]
				index := jsonPathValue.GetValue().ValueToString()
				if jsonPathValue.GetValue().GetType() == types.ValueType_FLOAT {
					index = fmt.Sprintf("%.0f", (*types.TypedFloat)(jsonPathValue.GetValue()).Float32())
				}
				jsonRoPath := jsonPathStr[:strings.LastIndex(jsonPathStr, "/")]
				idx, ok := indexMap[jsonRoPath]
				if ok {
					indexMap[jsonRoPath] = append(idx, indexName+"="+index)
				} else {
					arr := make([]string, 0)
					arr = append(arr, indexName+"="+index)
					indexMap[jsonRoPath] = arr
				}
				break
			}
		}
	}

	// Entries need to be compared in alphabetical order - copy in to Array and sort
	indexTable := make([]indexEntry, len(indexMap))
	i := 0
	for path, idxElem := range indexMap {
		sort.Slice(idxElem, func(i, j int) bool {
			return idxElem[i] < idxElem[j]
		})
		indexTable[i] = indexEntry{
			path: path,
			key:  idxElem,
		}
		i++
	}
	sort.Slice(indexTable, func(i, j int) bool {
		return indexTable[i].path > indexTable[j].path
	})

	for _, index := range indexTable {
		for _, cv := range correctedPathValues {
			if strings.HasPrefix(cv.Path, index.path) {
				suffix := cv.Path[len(index.path)+1:]
				newPath := fmt.Sprintf("%s[%s]/%s", index.path[:strings.LastIndex(index.path, "[")], strings.Join(index.key, ","), suffix)
				cv.Path = newPath
			}
		}
	}

	return correctedPathValues, nil
}

func extractIndices(indexStr string) []string {
	indices := make([]string, 0)
	indicesTrimmed := indexStr[1:strings.LastIndex(indexStr, "=")]
	for _, i := range strings.Split(indicesTrimmed, " ") {
		indices = append(indices, fmt.Sprintf("/%s", i))
	}
	return indices
}

func assignModelType(paths modelregistry.PathMap, modelPath string,
	jsonPathValue *types.PathValue) (*types.TypedValue, error) {

	modelType, err := paths.TypeForPath(modelPath)
	if err != nil {
		return nil, err
	}

	newTypeValue, err := pathType(jsonPathValue, modelType)
	if err != nil {
		return nil, err
	}

	return newTypeValue, nil
}

func pathType(jsonPathValue *types.PathValue, spType types.ValueType) (*types.TypedValue, error) {
	var newTypeValue *types.TypedValue
	var err error
	switch jsonPathValue.Value.Type {
	case types.ValueType_FLOAT:
		// Could be int, uint, or float from json - convert to numeric
		floatVal := (*types.TypedFloat)(jsonPathValue.Value).Float32()

		switch spType {
		case types.ValueType_FLOAT:
			newTypeValue = jsonPathValue.GetValue()
		case types.ValueType_INT:
			newTypeValue = types.NewTypedValueInt64(int(floatVal))
		case types.ValueType_UINT:
			newTypeValue = types.NewTypedValueUint64(uint(floatVal))
			//case types.ValueTypeDECIMAL:
			// TODO add a conversion from float to D64 will also need number of decimal places from Read Only SubPath
			//	newTypeValue = types.NewTypedValueDecimal64()
		case types.ValueType_STRING:
			newTypeValue = types.NewTypedValueString(fmt.Sprintf("%.0f", float64(floatVal)))
		default:
			newTypeValue = jsonPathValue.GetValue()
		}

	default:
		newTypeValue, err = types.NewTypedValue(jsonPathValue.GetValue().GetBytes(), spType, []int32{})
		if err != nil {
			return nil, err
		}
	}
	return newTypeValue, nil
}

func hasPrefixMultipleIdx(a string, b string) bool {
	aParts := strings.Split(a, "[")
	bParts := strings.Split(b, "[")
	if len(aParts) != len(bParts) {
		return false
	}

	for idx, p := range aParts {
		if idx < len(aParts)-1 {
			if p != bParts[idx] {
				return false
			}
		}
		// For the last part
		if !strings.HasPrefix(p, bParts[idx]) {
			return false
		}
	}
	return true
}
