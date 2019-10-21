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

package store

import (
	"encoding/json"
	"fmt"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	"github.com/onosproject/onos-config/pkg/utils"
	"strconv"
	"strings"
)

const (
	slash     = "/"
	equals    = "="
	bracketsq = "["
	brktclose = "]"
)

// BuildTree is a function that takes an ordered array of ConfigValues and
// produces a structured formatted JSON tree
func BuildTree(values []*devicechangetypes.PathValue, floatAsStr bool) ([]byte, error) {

	root := make(map[string]interface{})
	rootif := interface{}(root)
	for _, cv := range values {
		err := addPathToTree(cv.Path, cv.GetValue(), &rootif, floatAsStr)
		if err != nil {
			return nil, err
		}
	}

	buf, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// addPathToTree is a recursive function that builds up a map
// suitable for using with json.Marshal, which in turn can be used to feed in to
// ygot.Unmarshall
// This follows the approach in https://blog.golang.org/json-and-go "Generic JSON with interface{}"
func addPathToTree(path string, value *devicechangetypes.TypedValue, nodeif *interface{}, floatAsStr bool) error {
	pathelems := utils.SplitPath(path)

	// Convert to its real type
	nodemap, ok := (*nodeif).(map[string]interface{})
	if !ok {
		return fmt.Errorf("could not convert nodeif %v for %s", *nodeif, path)
	}

	if len(pathelems) == 1 && len(value.Bytes) > 0 {
		// At the end of a line - this is the leaf
		handleLeafValue(nodemap, value, pathelems, floatAsStr)

	} else if strings.Contains(pathelems[0], equals) {
		// To handle list index items
		refinePath := strings.Join(pathelems[1:], slash)
		if refinePath == "" {
			return nil
		}
		refinePath = fmt.Sprintf("/%s", refinePath)
		brktIdx := strings.Index(pathelems[0], bracketsq)
		eqIdx := strings.Index(pathelems[0], equals)
		brktIdx2 := strings.Index(pathelems[0], brktclose)

		listName := pathelems[0][:brktIdx]
		keyName := pathelems[0][brktIdx+1 : eqIdx]
		keyVal := pathelems[0][eqIdx+1 : brktIdx2]
		keyValNum, keyValNumErr := strconv.Atoi(keyVal)

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
		var foundit bool
		for idx, ls := range listSliceIf {
			lsMap, ok := ls.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Failed to convert list slice %d", idx)
			}
			if (lsMap)[keyName] == keyVal || (keyValNumErr == nil && (lsMap)[keyName] == keyValNum) {
				listItemMap = lsMap
				foundit = true
				break
			}
		}
		if !foundit {
			listItemMap = make(map[string]interface{})
			if keyValNumErr == nil {
				listItemMap[keyName] = keyValNum
			} else {
				listItemMap[keyName] = keyVal
			}
		}
		listItemIf := interface{}(listItemMap)
		err := addPathToTree(refinePath, value, &listItemIf, floatAsStr)
		if err != nil {
			return err
		}
		if !foundit {
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
			elemIf := interface{}(elemMap)

			err := addPathToTree(refinePath, value, &elemIf, floatAsStr)
			if err != nil {
				return err
			}
			(nodemap)[pathelems[0]] = elemMap
		} else {
			//Reuse existing elemMap
			err := addPathToTree(refinePath, value, &elemMap, floatAsStr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func handleLeafValue(nodemap map[string]interface{}, value *devicechangetypes.TypedValue, pathelems []string, floatAsStr bool) {
	switch value.Type {
	case devicechangetypes.ValueType_EMPTY:
		// NOOP
	case devicechangetypes.ValueType_STRING:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedString)(value).String()
	case devicechangetypes.ValueType_INT:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedInt64)(value).Int()
	case devicechangetypes.ValueType_UINT:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedUint64)(value).Uint()
	case devicechangetypes.ValueType_DECIMAL:
		if floatAsStr {
			(nodemap)[pathelems[0]] = (*devicechangetypes.TypedDecimal64)(value).String()
		} else {
			(nodemap)[pathelems[0]] = (*devicechangetypes.TypedDecimal64)(value).Float()
		}
	case devicechangetypes.ValueType_FLOAT:
		if floatAsStr {
			(nodemap)[pathelems[0]] = (*devicechangetypes.TypedFloat)(value).String()
		} else {
			(nodemap)[pathelems[0]] = (*devicechangetypes.TypedFloat)(value).Float32()
		}
	case devicechangetypes.ValueType_BOOL:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedBool)(value).Bool()
	case devicechangetypes.ValueType_BYTES:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedBytes)(value).ByteArray()
	case devicechangetypes.ValueType_LEAFLIST_STRING:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedLeafListString)(value).List()
	case devicechangetypes.ValueType_LEAFLIST_INT:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedLeafListInt64)(value).List()
	case devicechangetypes.ValueType_LEAFLIST_UINT:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedLeafListUint)(value).List()
	case devicechangetypes.ValueType_LEAFLIST_BOOL:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedLeafListBool)(value).List()
	case devicechangetypes.ValueType_LEAFLIST_DECIMAL:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedLeafListDecimal)(value).ListFloat()
	case devicechangetypes.ValueType_LEAFLIST_FLOAT:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedLeafListFloat)(value).List()
	case devicechangetypes.ValueType_LEAFLIST_BYTES:
		(nodemap)[pathelems[0]] = (*devicechangetypes.TypedLeafListBytes)(value).List()
	default:
		(nodemap)[pathelems[0]] = fmt.Sprintf("unexpected %d", value.Type)
	}

}
