// Copyright 2019-present Open Networking Foundation
//
// Licensed under the Apache License, Configuration 2.0 (the "License");
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

package store

import (
	"github.com/onosproject/onos-config/pkg/store/change"
	"time"
)

// NetworkConfiguration is a model of a network defined configuration
// change. It is a combination of changes to devices (configurations)
type NetworkConfiguration struct {
	Name                 string
	Created              time.Time
	User                 string
	ConfigurationChanges map[string]change.ID
}
