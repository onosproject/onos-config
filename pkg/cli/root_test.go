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

func Test_Root(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)

	setUpMockClients(MockClientsConfig{})

	testCases := []struct {
		description string
		expected    string
	}{
		{description: "Config command", expected: `Read and update the config configuration`},
		{description: "Roll back command", expected: `Rolls-back a network configuration change`},
		{description: "Usage header", expected: `Usage:`},
		{description: "Usage config command", expected: `config [command]`},
	}

	cmd := GetCommand()
	assert.Assert(t, cmd != nil)
	cmd.SetOut(outputBuffer)

	usageErr := cmd.Usage()
	assert.NilError(t, usageErr)

	output := outputBuffer.String()

	for _, testCase := range testCases {
		assert.Assert(t, strings.Contains(output, testCase.expected), `Expected output "%s"" for %s not found`,
			testCase.expected, testCase.description)
	}
}
