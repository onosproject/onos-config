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

// ID utils test

package utils

import (
	"gotest.tools/assert"
	"strings"
	"testing"
)

func TestToConfigName(t *testing.T) {
	const (
		deviceID      = "device"
		deviceVersion = "2.2"
	)
	configName := ToConfigName(deviceID, deviceVersion)
	assert.Assert(t, strings.Contains(configName, deviceID))
	assert.Assert(t, strings.Contains(configName, deviceVersion))
}

func TestToModelName(t *testing.T) {
	const (
		deviceName    = "abc"
		deviceVersion = "1.2.3"
	)
	modelName := ToModelName(deviceName, deviceVersion)
	assert.Assert(t, strings.Contains(modelName, deviceName))
	assert.Assert(t, strings.Contains(modelName, deviceVersion))
}
