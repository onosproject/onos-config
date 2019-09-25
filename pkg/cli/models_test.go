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
	"github.com/openconfig/gnmi/proto/gnmi"
	"gotest.tools/assert"
	"io"
	"strings"
	"testing"
)

var called = 1 //  hack
func (c MockConfigAdminServiceListRegisteredModelsClient) RecvMock() (*admin.ModelInfo, error) {
	if called <= 2 {
		index := called
		called++
		roPaths := make([]*admin.ReadOnlyPath, 1)
		roPaths[0] = &admin.ReadOnlyPath{
			Path:    fmt.Sprintf("/root/ropath/path%d", index),
			SubPath: nil,
		}
		modelData := make([]*gnmi.ModelData, 1)
		modelData[0] = &gnmi.ModelData{
			Name:         "UT NAME",
			Organization: "UT ORG",
			Version:      "3.3.3",
		}
		return &admin.ModelInfo{
			Name:         fmt.Sprintf("Model-%d", index),
			Version:      "1.0",
			Module:       fmt.Sprintf("Module-%d", index),
			ReadOnlyPath: roPaths,
			ModelData:    modelData,
		}, nil
	}
	return nil, io.EOF
}

func Test_ListPlugins(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)

	modelsClient := MockConfigAdminServiceListRegisteredModelsClient{}
	modelsClient.recvFn = func() (*admin.ModelInfo, error) {
		if called <= 2 {
			index := called
			called++
			roPaths := make([]*admin.ReadOnlyPath, 1)
			roPaths[0] = &admin.ReadOnlyPath{
				Path:    fmt.Sprintf("/root/ropath/path%d", index),
				SubPath: nil,
			}
			modelData := make([]*gnmi.ModelData, 1)
			modelData[0] = &gnmi.ModelData{
				Name:         "UT NAME",
				Organization: "UT ORG",
				Version:      "3.3.3",
			}
			return &admin.ModelInfo{
				Name:         fmt.Sprintf("Model-%d", index),
				Version:      "1.0",
				Module:       fmt.Sprintf("Module-%d", index),
				ReadOnlyPath: roPaths,
				ModelData:    modelData,
			}, nil
		}
		return nil, io.EOF
	}
	setUpMockClients(&modelsClient)
	plugins := getGetPluginsCommand()
	args := make([]string, 1)
	args[0] = "-v"
	err := plugins.RunE(plugins, args)
	assert.NilError(t, err)
	output := outputBuffer.String()
	assert.Assert(t, strings.Contains(output, "/root/ropath/path1"))
}
