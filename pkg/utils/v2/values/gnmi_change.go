// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package values

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-config/pkg/utils"
	pathutils "github.com/onosproject/onos-config/pkg/utils/path"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// NewChangeValue decodes a path and value in to a ChangeValue
func NewChangeValue(path string, value configapi.TypedValue, delete bool) (*configapi.PathValue, error) {
	cv := configapi.PathValue{
		Path:    path,
		Value:   value,
		Deleted: delete,
	}
	if err := pathutils.IsPathValid(path); err != nil {
		return nil, err
	}
	return &cv, nil
}

// PathValuesToGnmiChange converts a Protobuf defined array of values objects to gNMI format
func PathValuesToGnmiChange(values []*configapi.PathValue, target configapi.TargetID) (*gnmi.SetRequest, error) {
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

	targetPath := &gnmi.Path{Target: string(target)}
	var setRequest = gnmi.SetRequest{
		Delete:  deletePaths,
		Replace: replacedPaths,
		Update:  updatedPaths,
		Prefix:  targetPath,
	}

	return &setRequest, nil
}
