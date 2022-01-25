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

package tree

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-config/pkg/utils"
)

const (
	slash     = "/"
	equals    = "="
	bracketsq = "["
	brktclose = "]"
)

// BuildTree is a function that takes an ordered array of ConfigValues and
// produces a structured formatted JSON tree
// For YANG the only type of float value is decimal, which is represented as a
// string - therefore all float value must be string in JSON
// Same with int64 and uin64 as per RFC 7951
func BuildTree(values []*configapi.PathValue, jsonRFC7951 bool) ([]byte, error) {

	root := make(map[string]interface{})
	rootif := interface{}(root)
	for _, cv := range values {
		err := addPathToTree(cv.Path, &cv.Value, &rootif, jsonRFC7951)
		if err != nil {
			return nil, err
		}
	}

	buf, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// addPathToTree is a recursive function that builds up a map
// suitable for using with json.Marshal, which in turn can be used to feed in to
// ygot.Unmarshall
// This follows the approach in https://blog.golang.org/json-and-go "Generic JSON with interface{}"
func addPathToTree(path string, value *configapi.TypedValue, nodeif *interface{}, jsonRFC7951 bool) error {
	pathelems := utils.SplitPath(path)

	// Convert to its real type
	nodemap, ok := (*nodeif).(map[string]interface{})
	if !ok {
		return fmt.Errorf("could not convert nodeif %v for %s", *nodeif, path)
	}

	if len(pathelems) == 1 {
		// At the end of a line - this is the leaf
		handleLeafValue(nodemap, value, pathelems, jsonRFC7951)

	} else if strings.Contains(pathelems[0], equals) {
		// To handle list index items
		refinePath := strings.Join(pathelems[1:], slash)
		if refinePath == "" {
			return nil
		}
		refinePath = fmt.Sprintf("/%s", refinePath)

		brktIdx := strings.Index(pathelems[0], bracketsq)
		listName := pathelems[0][:brktIdx]

		// Build up a map of keyName to keyVal
		keyMap := make(map[string]interface{})
		keyString := pathelems[0][brktIdx:]
		for strings.Contains(keyString, equals) {
			brktIdx := strings.Index(keyString, bracketsq)
			eqIdx := strings.Index(keyString, equals)
			brktIdx2 := strings.Index(keyString, brktclose)

			keyName := keyString[brktIdx+1 : eqIdx]
			keyVal := keyString[eqIdx+1 : brktIdx2]
			keyMap[keyName] = keyVal

			// position to look at next potential key string
			keyString = keyString[brktIdx2+1:]
		}

		listSlice, ok := nodemap[listName]
		if !ok {
			listSlice = make([]interface{}, 0)
			(nodemap)[listName] = listSlice
		}
		listSliceIf, ok := listSlice.([]interface{})
		if !ok {
			return fmt.Errorf("Failed to convert list slice %s", listName)
		}
		//Reuse existing listSlice
		var listItemMap map[string]interface{}
		var foundkeys int

		for idx, ls := range listSliceIf {
			lsMap, ok := ls.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Failed to convert list slice %d", idx)
			}
			for k, v := range keyMap {
				if l, ok := lsMap[k]; ok {
					// compare as strings
					lStr := convertBasicType(l)
					vStr := convertBasicType(v)
					if lStr == vStr {
						foundkeys++
						listItemMap = lsMap
					}
				}
			}
		}
		if foundkeys < len(keyMap) {
			listItemMap = keyMap
		}
		listItemIf := interface{}(listItemMap)
		err := addPathToTree(refinePath, value, &listItemIf, jsonRFC7951)
		if err != nil {
			return err
		}
		if foundkeys < len(keyMap) {
			listSliceIf = append(listSliceIf, listItemIf)
			(nodemap)[listName] = listSliceIf
		}
	} else {
		refinePath := strings.Join(pathelems[1:], slash)
		if refinePath == "" {
			return nil
		}
		refinePath = fmt.Sprintf("%s%s", slash, refinePath)

		elemMap, ok := (nodemap)[pathelems[0]]
		if !ok {
			elemMap = make(map[string]interface{})
			elemIf := elemMap

			err := addPathToTree(refinePath, value, &elemIf, jsonRFC7951)
			if err != nil {
				return err
			}
			(nodemap)[pathelems[0]] = elemMap
		} else {
			//Reuse existing elemMap
			err := addPathToTree(refinePath, value, &elemMap, jsonRFC7951)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func convertBasicType(v interface{}) string {
	vv := reflect.ValueOf(v)
	switch vv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", vv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", vv.Uint())
	case reflect.Bool:
		if vv.Bool() {
			return "true"
		}
		return "false"
	}
	return vv.String()
}

func handleLeafValue(nodemap map[string]interface{}, value *configapi.TypedValue, pathelems []string, jsonRFC7951 bool) {
	switch value.Type {
	case configapi.ValueType_EMPTY:
		// NOOP
	case configapi.ValueType_STRING:
		(nodemap)[pathelems[0]] = (*configapi.TypedString)(value).String()
	case configapi.ValueType_INT:
		if jsonRFC7951 && len(value.TypeOpts) > 0 && value.TypeOpts[0] > int32(configapi.WidthThirtyTwo) {
			(nodemap)[pathelems[0]] = (*configapi.TypedInt)(value).String()
		} else {
			(nodemap)[pathelems[0]] = (*configapi.TypedInt)(value).Int()
		}
	case configapi.ValueType_UINT:
		if jsonRFC7951 && len(value.TypeOpts) > 0 && value.TypeOpts[0] > int32(configapi.WidthThirtyTwo) {
			(nodemap)[pathelems[0]] = (*configapi.TypedUint)(value).String()
		} else {
			(nodemap)[pathelems[0]] = (*configapi.TypedUint)(value).Uint()
		}
	case configapi.ValueType_DECIMAL:
		if jsonRFC7951 {
			(nodemap)[pathelems[0]] = (*configapi.TypedDecimal)(value).String()
		} else {
			(nodemap)[pathelems[0]] = (*configapi.TypedDecimal)(value).Float()
		}
	case configapi.ValueType_FLOAT:
		if jsonRFC7951 {
			(nodemap)[pathelems[0]] = (*configapi.TypedFloat)(value).String()
		} else {
			(nodemap)[pathelems[0]] = (*configapi.TypedFloat)(value).Float32()
		}
	case configapi.ValueType_BOOL:
		(nodemap)[pathelems[0]] = (*configapi.TypedBool)(value).Bool()
	case configapi.ValueType_BYTES:
		(nodemap)[pathelems[0]] = (*configapi.TypedBytes)(value).ByteArray()
	case configapi.ValueType_LEAFLIST_STRING:
		(nodemap)[pathelems[0]] = (*configapi.TypedLeafListString)(value).List()
	case configapi.ValueType_LEAFLIST_INT:
		leafList, width := (*configapi.TypedLeafListInt)(value).List()
		if jsonRFC7951 && width > configapi.WidthThirtyTwo {
			asStrList := make([]string, 0)
			for _, l := range leafList {
				asStrList = append(asStrList, fmt.Sprintf("%d", l))
			}
			(nodemap)[pathelems[0]] = asStrList
		} else {
			(nodemap)[pathelems[0]] = leafList
		}
	case configapi.ValueType_LEAFLIST_UINT:
		leafList, width := (*configapi.TypedLeafListUint)(value).List()
		if jsonRFC7951 && width > configapi.WidthThirtyTwo {
			asStrList := make([]string, 0)
			for _, l := range leafList {
				asStrList = append(asStrList, fmt.Sprintf("%d", l))
			}
			(nodemap)[pathelems[0]] = asStrList
		} else {
			(nodemap)[pathelems[0]] = leafList
		}
	case configapi.ValueType_LEAFLIST_BOOL:
		(nodemap)[pathelems[0]] = (*configapi.TypedLeafListBool)(value).List()
	case configapi.ValueType_LEAFLIST_DECIMAL:
		(nodemap)[pathelems[0]] = (*configapi.TypedLeafListDecimal)(value).ListFloat()
	case configapi.ValueType_LEAFLIST_FLOAT:
		(nodemap)[pathelems[0]] = (*configapi.TypedLeafListFloat)(value).List()
	case configapi.ValueType_LEAFLIST_BYTES:
		(nodemap)[pathelems[0]] = (*configapi.TypedLeafListBytes)(value).List()
	default:
		(nodemap)[pathelems[0]] = fmt.Sprintf("unexpected %d", value.Type)
	}

}