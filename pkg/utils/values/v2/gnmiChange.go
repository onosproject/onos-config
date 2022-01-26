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
	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-config/pkg/utils"
	pathutils "github.com/onosproject/onos-config/pkg/utils/path"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// NewChangeValue decodes a path and value in to a ChangeValue
func NewChangeValue(path string, value configapi.TypedValue, delete bool) (*configapi.ChangeValue, error) {
	cv := configapi.ChangeValue{
		Value:  value,
		Delete: delete,
	}
	if err := pathutils.IsPathValid(path); err != nil {
		return nil, err
	}
	return &cv, nil
}

// NativeChangeToGnmiChange converts a Protobuf defined Change object to gNMI format
func NativeChangeToGnmiChange(c *configapi.Change) (*gnmi.SetRequest, error) {
	var deletePaths = []*gnmi.Path{}
	var replacedPaths = []*gnmi.Update{}
	var updatedPaths = []*gnmi.Update{}

	for path, changeValue := range c.Values {
		elems := utils.SplitPath(path)
		pathElemsRefs, parseError := utils.ParseGNMIElements(elems)

		if parseError != nil {
			return nil, parseError
		}

		if changeValue.Delete {
			deletePaths = append(deletePaths, &gnmi.Path{Elem: pathElemsRefs.Elem})
		} else {
			gnmiValue, err := NativeTypeToGnmiTypedValue(&changeValue.Value)
			if err != nil {
				return nil, errors.NewInvalid("error converting %s: %s", path, err)
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

// PathValuesToGnmiChange converts a Protobuf defined array of values objects to gNMI format
func PathValuesToGnmiChange(values []*configapi.PathValue) (*gnmi.SetRequest, error) {
	var deletePaths = []*gnmi.Path{}
	var replacedPaths = []*gnmi.Update{}
	var updatedPaths = []*gnmi.Update{}

	for _, pathValue := range values {
		elems := utils.SplitPath(pathValue.Path)
		pathElemsRefs, parseError := utils.ParseGNMIElements(elems)

		if parseError != nil {
			return nil, parseError
		}

		if pathValue.Deleted {
			deletePaths = append(deletePaths, &gnmi.Path{Elem: pathElemsRefs.Elem})
		} else {
			gnmiValue, err := NativeTypeToGnmiTypedValue(&pathValue.Value)
			if err != nil {
				return nil, errors.NewInvalid("error converting %s: %s", pathValue.Path, err)
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
