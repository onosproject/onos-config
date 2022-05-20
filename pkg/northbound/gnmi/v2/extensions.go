// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"github.com/gogo/protobuf/proto"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

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

func getTargetVersionOverrides(request interface{}) (*configapi.TargetVersionOverrides, error) {
	var err error
	var s interface{}
	switch req := request.(type) {
	case *gnmi.SetRequest:
		s, err = extractExtension(req.GetExtension(), configapi.TargetVersionOverridesID, &configapi.TargetVersionOverrides{})
	case *gnmi.GetRequest:
		s, err = extractExtension(req.GetExtension(), configapi.TargetVersionOverridesID, &configapi.TargetVersionOverrides{})
	default:
		return nil, errors.NewInvalid("invalid-request-type")
	}
	if err != nil {
		return nil, err
	}
	overrides, ok := s.(*configapi.TargetVersionOverrides)
	if !ok {
		return nil, errors.NewInternal("extracted-the-wrong-extensions")
	}
	return overrides, nil
}
