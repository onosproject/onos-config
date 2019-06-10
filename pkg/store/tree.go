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
	"github.com/onosproject/onos-config/pkg/store/change"
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
func BuildTree(values []*change.ConfigValue) ([]byte, error) {

	root := make(map[string]interface{})
	rootif := interface{}(root)
	for _, cv := range values {
		err := addPathToTree(cv.Path, cv.Value, &rootif)
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
func addPathToTree(path, value string, nodeif *interface{}) error {
	pathelems := strings.Split(path, slash)

	// Convert to its real type
	nodemap, ok := (*nodeif).(map[string]interface{})
	if !ok {
		return fmt.Errorf("Could not convert nodeif %v for %s", *nodeif, path)
	}

	if len(pathelems) == 2 && value != "" {
		// At the end of a line - this is the leaf
		numval, err := strconv.Atoi(value)
		if err == nil {
			(nodemap)[pathelems[1]] = numval
		} else if strings.ToLower(value) == "true" || strings.ToLower(value) == "false" {
			(nodemap)[pathelems[1]] = strings.ToLower(value)
		} else {
			(nodemap)[pathelems[1]] = value
		}
	} else if strings.Contains(pathelems[1], equals) {
		// To handle list index items
		refinePath := strings.Join(pathelems[2:], slash)
		if refinePath == "" {
			return nil
		}
		refinePath = fmt.Sprintf("/%s", refinePath)
		brktIdx := strings.Index(pathelems[1], bracketsq)
		eqIdx := strings.Index(pathelems[1], equals)
		brktIdx2 := strings.Index(pathelems[1], brktclose)

		listName := pathelems[1][:brktIdx]
		keyName := pathelems[1][brktIdx+1 : eqIdx]
		keyVal := pathelems[1][eqIdx+1 : brktIdx2]

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
			if (lsMap)[keyName] == keyVal {
				listItemMap = lsMap
				foundit = true
				break
			}
		}
		if !foundit {
			listItemMap = make(map[string]interface{})
			listItemMap[keyName] = keyVal
		}
		listItemIf := interface{}(listItemMap)
		err := addPathToTree(refinePath, value, &listItemIf)
		if err != nil {
			return err
		}
		if !foundit {
			listSliceIf = append(listSliceIf, listItemIf)
			(nodemap)[listName] = listSliceIf
		}
	} else {
		refinePath := strings.Join(pathelems[2:], slash)
		if refinePath == "" {
			return nil
		}
		refinePath = fmt.Sprintf("%s%s", slash, refinePath)

		elemMap, ok := (nodemap)[pathelems[1]]
		if !ok {
			elemMap = make(map[string]interface{})
			elemIf := interface{}(elemMap)

			err := addPathToTree(refinePath, value, &elemIf)
			if err != nil {
				return err
			}
			(nodemap)[pathelems[1]] = elemMap
		} else {
			//Reuse existing elemMap
			err := addPathToTree(refinePath, value, &elemMap)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
