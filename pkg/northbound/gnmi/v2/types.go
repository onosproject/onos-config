// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/openconfig/gnmi/proto/gnmi"
)

type targetInfo struct {
	targetID      configapi.TargetID
	targetVersion configapi.TargetVersion
	targetType    configapi.TargetType
	plugin        pluginregistry.ModelPlugin
	persistent    bool
	updates       configapi.TypedValueMap
	removes       []string
	configuration *configapi.Configuration
}

type pathInfo struct {
	targetID     configapi.TargetID
	path         *gnmi.Path
	pathAsString string
}
