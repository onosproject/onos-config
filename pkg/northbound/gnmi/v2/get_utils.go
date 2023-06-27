// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"regexp"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/v2/tree"
	valuesv2 "github.com/onosproject/onos-config/pkg/utils/v2/values"
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
			if len(prefixPath) > len(cv.Path) {
				//  If prefix is longer than the path, it can't possibly match
				continue
			}
			pathCv, err := utils.ParseGNMIElements(strings.Split(strings.Trim(cv.Path, "/"), "/"))
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
