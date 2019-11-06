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
	"fmt"
	"github.com/onosproject/onos-config/api/diags"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	"gotest.tools/assert"
	"io"
	"regexp"
	"strings"
	"testing"
)

var opstateInfo []diags.OpStateResponse

func generateOpstate(count int) {
	opstateInfo = make([]diags.OpStateResponse, count)
	for opstateIndex := range opstateInfo {
		valueString := fmt.Sprintf("value%d", opstateIndex)
		pathString := fmt.Sprintf("/root/system/path%d", opstateIndex)
		value := devicechange.PathValue{
			Path:  pathString,
			Value: devicechange.NewTypedValueString(valueString),
		}
		opstateInfo[opstateIndex] = diags.OpStateResponse{
			Type:      0,
			Pathvalue: &value,
		}
	}
}

var nextOpstateIndex = 0

func recvOpstateMock() (*diags.OpStateResponse, error) {
	if nextOpstateIndex < len(opstateInfo) {
		opstate := opstateInfo[nextOpstateIndex]
		nextOpstateIndex++

		return &opstate, nil
	}
	return nil, io.EOF
}

func Test_Opstate(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)

	opStateClient := MockOpStateDiagsGetOpStateClient{
		recvFn: recvOpstateMock,
	}

	setUpMockClients(MockClientsConfig{opstateClient: &opStateClient})
	generateOpstate(3)
	opstateCmd := getGetOpstateCommand()
	args := make([]string, 1)
	args[0] = "My Device"
	err := opstateCmd.RunE(opstateCmd, args)
	assert.NilError(t, err, "Error fetching opstate command")
	outputString := outputBuffer.String()
	output := strings.Split(strings.TrimSuffix(outputString, "\n"), "\n")

	testCases := []struct {
		description string
		index       int
		regexp      string
	}{
		{description: "Path 0", index: 2, regexp: `/root/system/path0\s+\|\(STRING\) value0 +\|`},
		{description: "Path 1", index: 3, regexp: `/root/system/path1\s+\|\(STRING\) value1 +\|`},
		{description: "Path 2", index: 4, regexp: `/root/system/path2\s+\|\(STRING\) value2 +\|`},
		{description: "Header", index: 0, regexp: `OPSTATE CACHE: My Device`},
		{description: "Column Headers", index: 1, regexp: `PATH +\|VALUE`},
	}

	for _, testCase := range testCases {
		re := regexp.MustCompile(testCase.regexp)
		assert.Assert(t, re.MatchString(output[testCase.index]),
			testCase.description, fmt.Sprintf(". '%s' does not match '%s'", re.String(), output[testCase.index]))
	}
}
