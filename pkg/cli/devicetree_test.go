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
	"time"
)

var configRecvCount = 0

func diagConfigRecv() (*diags.Configuration, error) {
	if configRecvCount == 0 {
		now := time.Now()
		result := &diags.Configuration{
			Name:       "config1",
			DeviceID:   "device1",
			Version:    "1.0.0",
			DeviceType: "switch",
			Created:    &now,
			Updated:    &now,
			ChangeIDs:  []string{"2Lo0ZC0wqhuwbnSF9px6kDvEEWI="},
		}
		configRecvCount++
		return result, nil
	}
	return nil, io.EOF
}

var changesRecvCount = 0

func diagChangesRecv() (*admin.Change, error) {
	if changesRecvCount < 3 {
		now := time.Now()
		result := &admin.Change{
			Time: &now,
			Id:   "2Lo0ZC0wqhuwbnSF9px6kDvEEWI=",
			Desc: "Change1",
			ChangeValues: []*admin.ChangeValue{
				{Path: "/a/b/c",
					Value:     []byte("VALUE"),
					ValueType: admin.ChangeValueType_STRING},
			},
		}
		changesRecvCount++
		return result, nil
	}
	return nil, io.EOF
}

func Test_DeviceTreeCommand(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)

	configsClient := MockConfigDiagsGetConfigurationsClient{
		recvFn: diagConfigRecv,
	}

	changesClient := MockConfigDiagsGetChangesClient{
		recvFn: diagChangesRecv,
	}

	setUpMockClients(MockClientsConfig{
		configDiagsClientConfigurations: &configsClient,
		configDiagsClientChanges:        &changesClient,
	})
	cmd := getGetDeviceTreeCommand()
	args := []string {"device1"}
	err := cmd.RunE(cmd, args)
	assert.NilError(t, err)
	output := strings.Split(strings.TrimSuffix(outputBuffer.String(), "\n"), "\n")
	fmt.Printf("%s\n", output)

	testCases := []struct {
		description string
		index       int
		regexp      string
	}{
		{description: "Headers", index: 0, regexp: `DEVICE[ \t]+CONFIGURATION[ \t]+TYPE[ \t]+VERSION`},
		{description: "Device Info", index: 1, regexp: `device1[ \t]+device1-1.0.0[ \t]+switch[ \t]+1.0.0`},
		{description: "Change Id", index: 2, regexp: `CHANGE:[ \t]+2Lo0ZC0wqhuwbnSF9px6kDvEEWI=`},
		{description: "Tree", index: 3, regexp: `TREE:`},
		{description: "Change Contents", index: 4, regexp: `{"a":{"b":{"c":"VALUE"}}}`},
	}

	for _, testCase := range testCases {
		re := regexp.MustCompile(testCase.regexp)
		assert.Assert(t, re.MatchString(output[testCase.index]), "%s", testCase.description)
	}
}
