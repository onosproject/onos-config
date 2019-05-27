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

package gnmi

const (
	// GnmiExtensionNetwkChangeID is the extension number used in SetRequest and SetResponse
	GnmiExtensionNetwkChangeID = 100

	// GnmiExtensionVersion is used in Set, Get and Subscribe
	GnmiExtensionVersion = 101

	// GnmiExtensionDeviceType is used in Set only when creating a device the first time
	// It can be used as a discriminator on Get when wildcard target is given
	GnmiExtensionDeviceType = 102
)
