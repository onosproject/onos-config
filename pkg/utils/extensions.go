// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

// TargetVersionOverrideExtension returns a target type/version override extension
func TargetVersionOverrideExtension(id configapi.TargetID, targetType configapi.TargetType, targetVersion configapi.TargetVersion) (*gnmi_ext.Extension, error) {
	ext := configapi.TargetVersionOverrides{
		Overrides: map[string]*configapi.TargetTypeVersion{string(id): {TargetType: targetType, TargetVersion: targetVersion}},
	}
	b, err := ext.Marshal()
	if err != nil {
		return nil, err
	}
	return &gnmi_ext.Extension{
		Ext: &gnmi_ext.Extension_RegisteredExt{
			RegisteredExt: &gnmi_ext.RegisteredExtension{
				Id:  configapi.TargetVersionOverridesID,
				Msg: b,
			},
		},
	}, nil
}
