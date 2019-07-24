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

type indexEntry struct {
	path string
	key  string
}

//CorrectJSONPaths takes configuration values extracted from a json, of which we are not sure about the type and,
// through the model plugin assigns them the correct type according to the YANG model provided, returning an
// updated set of configuration values.
func CorrectJSONPaths(jsonPathValues []*change.ConfigValue, roPaths modelregistry.ReadOnlyPathMap) ([]*change.ConfigValue, error) {
	correctedPathValues := make([]*change.ConfigValue, 0)
	rOnIndex := regexp.MustCompile(matchOnIndex)
	indexTable := make([]indexEntry, 0)

	for _, jsonPathValue := range jsonPathValues {
		jsonMatches := rOnIndex.FindAllStringSubmatch(jsonPathValue.Path, -1)
		jsonPathWildIndex := jsonPathValue.Path
		jsonPathIdx := jsonPathValue.Path
		for idx, m := range jsonMatches {
			jsonPathWildIndex = strings.Replace(jsonPathWildIndex, m[1], "[*]", -1)
			if idx == len(jsonMatches)-1 { //Last one only
				jsonPathIdx = strings.Replace(jsonPathIdx, m[1]+"/", "[", -1)
			} else {
				jsonPathIdx = strings.Replace(jsonPathIdx, m[1], "[*]", -1)
			}
		}

		for ropath, subpaths := range roPaths {
			roMatches := rOnIndex.FindAllStringSubmatch(ropath, -1)
			roPathWildIndex := ropath
			roPathOnlyLastIndex := ropath
			for idx, m := range roMatches {
				roPathWildIndex = strings.Replace(roPathWildIndex, m[1], "[*]", -1)
				if idx != len(roMatches)-1 { // Not the last one
					roPathOnlyLastIndex = strings.Replace(roPathOnlyLastIndex, m[1], "[*]", -1)
				}
			}

			if ropath == jsonPathValue.Path {
				// Simple read-only path is leaf
				if len(subpaths) != 1 {
					return nil, fmt.Errorf("expected RO path %s to have only 1 subpath. Found %d", ropath, len(subpaths))
				}
				// FIXME - there is no support at present for setting the type when ReadOnly leaf is "config false"
				correctedPathValues = append(correctedPathValues, jsonPathValue)
				break
			} else if hasPrefixMultipleIdx(jsonPathWildIndex, roPathWildIndex) {
				jsonSubPath := jsonPathWildIndex[len(roPathWildIndex):]
				for subPath, spType := range subpaths {
					if jsonSubPath == subPath {
						var newTypeValue *change.TypedValue
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
							default:
								newTypeValue = &jsonPathValue.TypedValue
							}

						default:
							newTypeValue, _ = change.CreateTypedValue(jsonPathValue.Value, spType, []int{})
						}

						jsonPathValue.TypedValue = *newTypeValue
						break
					}

				}

				correctedPathValues = append(correctedPathValues, jsonPathValue)
				break
			} else if hasPrefixMultipleIdx(roPathOnlyLastIndex, jsonPathIdx) {
				indexName := jsonPathIdx[strings.LastIndex(jsonPathIdx, "[")+1:]
				index := jsonPathValue.TypedValue.String()
				jsonRoPath := jsonPathValue.Path[:strings.LastIndex(jsonPathValue.Path, "/")]
				indexTable = append(indexTable, indexEntry{path: jsonRoPath, key: indexName + "=" + index})
				break
			}

		}
	}

	sort.Slice(indexTable, func(i, j int) bool {
		return indexTable[i].path > indexTable[j].path
	})

	for _, index := range indexTable {
		for _, cv := range correctedPathValues {
			if strings.HasPrefix(cv.Path, index.path) {
				suffix := cv.Path[len(index.path)+1:]
				newPath := index.path[:strings.LastIndex(index.path, "[")] + "[" + index.key + "]/" + suffix
				cv.Path = newPath
			}
		}
	}

	return correctedPathValues, nil
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
