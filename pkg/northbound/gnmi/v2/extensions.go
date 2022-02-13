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
	"github.com/gogo/protobuf/proto"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

var log = logging.GetLogger("northbound", "gnmi")

// extractExtension extract the value of an extension from a list given an extension ID
// if extType is passed we assume the content of the extension is a proto that needs to be Unmarshalled,
// if extType is nil we return the value as is
func extractExtension(ext []*gnmi_ext.Extension, extID configapi.ExtensionID, extType proto.Message) (interface{}, error) {
	for _, ex := range ext {
		if regExt, ok := ex.Ext.(*gnmi_ext.Extension_RegisteredExt); ok &&
			regExt.RegisteredExt.Id == extID {
			// if the extensions is proto message, then unmarshal it
			if extType != nil {
				if err := proto.Unmarshal(regExt.RegisteredExt.Msg, extType); err != nil {
					return nil, errors.NewInvalid(err.Error())
				}
				return extType, nil
			}
			// otherwise just return the values
			return regExt.RegisteredExt.Msg, nil
		}
	}
	return extType, nil
}

func getTransactionStrategy(request interface{}) (configapi.TransactionStrategy, error) {
	var err error
	var s interface{}
	switch req := request.(type) {
	case *gnmi.SetRequest:
		s, err = extractExtension(req.GetExtension(), configapi.TransactionStrategyExtensionID, &configapi.TransactionStrategy{})
	case *gnmi.GetRequest:
		s, err = extractExtension(req.GetExtension(), configapi.TransactionStrategyExtensionID, &configapi.TransactionStrategy{})
	default:
		return configapi.TransactionStrategy{}, errors.NewInvalid("invalid-request-type")
	}
	if err != nil {
		return configapi.TransactionStrategy{}, err
	}
	strategy, ok := s.(*configapi.TransactionStrategy)
	if !ok {
		return configapi.TransactionStrategy{}, errors.NewInternal("extracted-the-wrong-extensions")
	}
	return *strategy, nil
}
