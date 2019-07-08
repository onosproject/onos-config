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

package values

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// NativeChangeToGnmiChange converts a Change object to gNMI format
func NativeChangeToGnmiChange(c *change.Change) (*gnmi.SetRequest, error) {
	var deletePaths = []*gnmi.Path{}
	var replacedPaths = []*gnmi.Update{}
	var updatedPaths = []*gnmi.Update{}

	for _, changeValue := range c.Config {
		elems := utils.SplitPath(changeValue.Path)
		pathElemsRefs, parseError := utils.ParseGNMIElements(elems)

		if parseError != nil {
			return nil, parseError
		}

		if changeValue.Remove {
			deletePaths = append(deletePaths, &gnmi.Path{Elem: pathElemsRefs.Elem})
		} else {
			gnmiValue, err := NativeTypeToGnmiTypedValue(&changeValue.TypedValue)
			if err != nil {
				return nil, fmt.Errorf("error converting %s: %s", changeValue.Path, err)
			}
			updatePath := gnmi.Path{Elem: pathElemsRefs.Elem}
			updatedPaths = append(updatedPaths, &gnmi.Update{Path: &updatePath, Val: gnmiValue})
		}
	}

	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
	}

	return &setRequest, nil
}
