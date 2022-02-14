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
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/openconfig/gnmi/proto/gnmi"
)

type targetInfo struct {
	targetID      configapi.TargetID
	targetVersion configapi.TargetVersion
	targetType    configapi.TargetType
	plugin        pluginregistry.ModelPlugin
	updates       configapi.TypedValueMap
	removes       []string
	configuration *configapi.Configuration
}

type pathInfo struct {
	targetID     configapi.TargetID
	path         *gnmi.Path
	pathAsString string
}
