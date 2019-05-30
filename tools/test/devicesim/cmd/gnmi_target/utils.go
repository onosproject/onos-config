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

package main

import (
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

var leafValue string

// findLeaf finds a leaf in the given JSON tree that is stored in a map.
func findLeaf(aMap map[string]interface{}, leaf string) (string, error) {
	for key, val := range aMap {
		switch concreteVal := val.(type) {
		case map[string]interface{}:
			findLeaf(val.(map[string]interface{}), leaf)
		case []interface{}:
			parseArray(val.([]interface{}), leaf)
		default:
			if leaf == key {
				leafValue = concreteVal.(string)
				break
			}

		}
	}
	return leafValue, nil

}

func parseArray(array []interface{}, leaf string) {
	for _, val := range array {
		switch val.(type) {
		case map[string]interface{}:
			findLeaf(val.(map[string]interface{}), leaf)
		case []interface{}:
			parseArray(val.([]interface{}), leaf)

		}
	}
}

// gnmiFullPath builds the full path from the prefix and path.
func gnmiFullPath(prefix, path *pb.Path) *pb.Path {
	fullPath := &pb.Path{Origin: path.Origin}
	if path.GetElement() != nil {
		fullPath.Element = append(prefix.GetElement(), path.GetElement()...)
	}
	if path.GetElem() != nil {
		fullPath.Elem = append(prefix.GetElem(), path.GetElem()...)
	}
	return fullPath
}
