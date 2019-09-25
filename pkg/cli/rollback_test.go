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

// Unit tests for rollback CLI
package cli

import (
	"bytes"
	"gotest.tools/assert"
	"strings"
	"testing"
)

func Test_rollback(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)

	setUpMockClients(nil)
	rollback := getRollbackCommand()
	args := make([]string, 1)
	args[0] = "ABCD1234"
	err := rollback.RunE(rollback, args)
	assert.NilError(t, err)
	assert.Equal(t, LastCreatedClient.rollBackID, "ABCD1234")
	output := outputBuffer.String()
	assert.Assert(t, strings.Contains(output, "Rollback was successful"))
}
