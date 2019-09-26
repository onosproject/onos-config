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
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
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
		value := admin.ChangeValue{
			Path:      pathString,
			Value:     []byte(valueString),
			ValueType: admin.ChangeValueType_STRING,
			TypeOpts:  nil,
			Removed:   false,
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
	opstate := getGetOpstateCommand()
	args := make([]string, 1)
	args[0] = "-v"
	err := opstate.RunE(opstate, args)
	assert.NilError(t, err)
	outputString := outputBuffer.String()
	output := strings.Split(strings.TrimSuffix(outputString, "\n"), "\n")
	assert.Equal(t, len(output), 3)
	re0 := regexp.MustCompile(`/root/system/path0\s+\|\(STRING\) value0 +\|`)
	assert.Assert(t, re0.FindString(output[0]) != "", "path0 incorrect")

	re1 := regexp.MustCompile(`/root/system/path1\s+\|\(STRING\) value1 +\|`)
	assert.Assert(t, re1.FindString(output[1]) != "", "path0 incorrect")

	re2 := regexp.MustCompile(`/root/system/path2\s+\|\(STRING\) value2 +\|`)
	assert.Assert(t, re2.FindString(output[2]) != "", "path0 incorrect")
}
