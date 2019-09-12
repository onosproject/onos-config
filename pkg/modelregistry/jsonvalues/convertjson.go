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
	"github.com/onosproject/onos-config/pkg/store/change"
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
func CorrectJSONPaths(jsonBase string, jsonPathValues []*change.ConfigValue,
	paths modelregistry.PathMap, stripNamespaces bool) ([]*change.ConfigValue, error) {

	correctedPathValues := make([]*change.ConfigValue, 0)
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
				newTypeValue, err := assignModelType(paths, modelPath, "", jsonPathValue)
				if err != nil {
					return nil, err
				}
				jsonPathValue.TypedValue = *newTypeValue
				err = jsonPathValue.IsPathValid()
				if err != nil {
					return nil, fmt.Errorf("invalid value %s", err)
				}
				correctedPathValues = append(correctedPathValues, jsonPathValue)
				break

			} else if hasPrefixMultipleIdx(jsonPathWildIndex, modelPathWildIndex) {
				jsonSubPath := jsonPathWildIndex[len(modelPathWildIndex):]

				newTypeValue, err := assignModelType(paths, modelPath, jsonSubPath, jsonPathValue)
				if err != nil {
					return nil, err
				}
				cv := change.ConfigValue{
					Path:       jsonPathStr,
					TypedValue: *newTypeValue,
				}
				err = cv.IsPathValid()
				if err != nil {
					return nil, fmt.Errorf("invalid value %s", err)
				}
				correctedPathValues = append(correctedPathValues, &cv)
				break
			} else if hasPrefixMultipleIdx(modelPathOnlyLastIndex, jsonPathIdx) ||
				hasPrefixMultipleIdx(modelPathOnlySecondIndex, jsonPathIdx) {
				indexName := jsonPathIdx[strings.LastIndex(jsonPathIdx, "[")+1:]
				index := jsonPathValue.TypedValue.String()
				if jsonPathValue.Type == change.ValueTypeFLOAT {
					index = fmt.Sprintf("%.0f", (*change.TypedFloat)(&jsonPathValue.TypedValue).Float32())
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

func assignModelType(paths modelregistry.PathMap, modelPath string, jsonSubPath string, jsonPathValue *change.ConfigValue) (*change.TypedValue, error) {
	// If it's a RO path then have to go in to the subpaths to find
	// the right type
	ro, ok := paths.(modelregistry.ReadOnlyPathMap)
	if ok {
		subPaths, pathok := ro[modelPath]
		if pathok {
			for subPath, spType := range subPaths {
				if jsonSubPath == subPath {
					newTypeValue, err := pathType(jsonPathValue, spType)
					if err != nil {
						return nil, err
					}
					return newTypeValue, nil
				}
			}
		}
	}

	// If it's a RW path then extract the object and find its type
	rw, ok := paths.(modelregistry.ReadWritePathMap)
	if ok {
		rwObj, pathok := rw[modelPath]
		if pathok {
			newTypeValue, err := pathType(jsonPathValue, rwObj.ValueType)
			if err != nil {
				return nil, err
			}
			return newTypeValue, nil
		}
	}

	return &jsonPathValue.TypedValue, nil
}

func pathType(jsonPathValue *change.ConfigValue, spType change.ValueType) (*change.TypedValue, error) {
	var newTypeValue *change.TypedValue
	var err error
	switch jsonPathValue.Type {
	case change.ValueTypeFLOAT:
		// Could be int, uint, or float from json - convert to numeric
		floatVal := (*change.TypedFloat)(&jsonPathValue.TypedValue).Float32()

		switch spType {
		case change.ValueTypeFLOAT:
			newTypeValue = &jsonPathValue.TypedValue
		case change.ValueTypeINT:
			newTypeValue = change.CreateTypedValueInt64(int(floatVal))
		case change.ValueTypeUINT:
			newTypeValue = change.CreateTypedValueUint64(uint(floatVal))
			//case change.ValueTypeDECIMAL:
			// TODO add a conversion from float to D64 will also need number of decimal places from Read Only SubPath
			//	newTypeValue = change.CreateTypedValueDecimal64()
		case change.ValueTypeSTRING:
			newTypeValue = change.CreateTypedValueString(fmt.Sprintf("%.0f", float64(floatVal)))
		default:
			newTypeValue = &jsonPathValue.TypedValue
		}

	default:
		newTypeValue, err = change.CreateTypedValue(jsonPathValue.Value, spType, []int{})
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
