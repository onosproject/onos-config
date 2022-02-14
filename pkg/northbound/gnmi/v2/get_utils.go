// Copyright 2022-present Open Networking Foundation.
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

package gnmi

import (
	"regexp"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/tree"
	valuesv2 "github.com/onosproject/onos-config/pkg/utils/values/v2"
	"github.com/openconfig/gnmi/proto/gnmi"

	"strings"
)

func createUpdate(prefix *gnmi.Path, path *gnmi.Path, configValues []*configapi.PathValue, encoding gnmi.Encoding) ([]*gnmi.Update, error) {
	if len(configValues) == 0 {
		emptyUpdate := gnmi.Update{
			Path: path,
			Val:  nil,
		}
		return []*gnmi.Update{
			&emptyUpdate,
		}, nil
	}

	switch encoding {
	case gnmi.Encoding_JSON, gnmi.Encoding_JSON_IETF:
		json, err := tree.BuildTree(configValues, true)
		if err != nil {
			return nil, err
		}
		update := &gnmi.Update{
			Val: &gnmi.TypedValue{
				Value: &gnmi.TypedValue_JsonVal{
					JsonVal: json,
				},
			},
			Path: path,
		}
		return []*gnmi.Update{
			update,
		}, nil
	case gnmi.Encoding_PROTO:
		updates := make([]*gnmi.Update, 0, len(configValues))
		for _, cv := range configValues {
			gnmiVal, err := valuesv2.NativeTypeToGnmiTypedValue(&cv.Value)
			if err != nil {
				return nil, err
			}
			prefixPath := ""
			if prefix != nil {
				prefixPath = utils.StrPathElem(prefix.Elem)
			}
			pathCv, err := utils.ParseGNMIElements(strings.Split(cv.Path[len(prefixPath)+1:], "/"))
			if err != nil {
				return nil, err
			}
			if path != nil {
				pathCv.Target = path.Target
				pathCv.Origin = path.Origin
			}
			update := &gnmi.Update{
				Path: pathCv,
				Val:  gnmiVal,
			}
			updates = append(updates, update)
		}
		return updates, nil
	default:
		return nil, errors.NewInvalid("unsupported encoding %v", encoding)
	}
}

func filterTargetForURL(target string) string {
	re := regexp.MustCompile(`[.-]`)
	return re.ReplaceAllString(target, "_")
}
