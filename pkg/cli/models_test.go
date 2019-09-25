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

package cli

import (
	"bytes"
	"gotest.tools/assert"
	"strings"
	"testing"
)
func Test_ListPlugins(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)

	setUpMockClients()
	plugins := getGetPluginsCommand()
	args := make([]string, 1)
	args[0] = "-v"
	err := plugins.RunE(plugins, args)
	assert.NilError(t, err)
	output := outputBuffer.String()
	assert.Assert(t, strings.Contains(output, "/root/ropath/path1"))
}