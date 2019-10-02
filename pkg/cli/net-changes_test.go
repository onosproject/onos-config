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
	"gotest.tools/assert"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"
)

// test data and generator for it
var netChanges []*admin.NetChange

func generateNetChanges(count int) {
	netChanges = make([]*admin.NetChange, count)
	now := time.Now()
	for netChangesIndex := range netChanges {
		configChange := admin.ConfigChange{
			Id:   fmt.Sprintf("Change%d", netChangesIndex),
			Hash: fmt.Sprintf("Hash%d", netChangesIndex),
		}
		configChanges := make([]*admin.ConfigChange, 1)
		configChanges[0] = &configChange
		netChanges[netChangesIndex] = &admin.NetChange{
			Time:    &now,
			Name:    fmt.Sprintf("Change%d", netChangesIndex),
			User:    "User1",
			Changes: configChanges,
		}
	}
}

//  mock Recv() function - returns the prepared test data to the calling CLI command
var currentRecvIndex = 0

func mockChangesRecv() (*admin.NetChange, error) {
	if currentRecvIndex < len(netChanges) {
		retval := netChanges[currentRecvIndex]
		currentRecvIndex++
		return retval, nil
	}
	return nil, io.EOF
}

// Test_NetChanges tests the CLI get net-changes command
func Test_NetChanges(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)

	changesClient := MockConfigAdminServiceGetNetworkChangesClient{
		recvFn: mockChangesRecv,
	}
	setUpMockClients(MockClientsConfig{netChangesClient: &changesClient})
	generateNetChanges(4)

	cmd := getGetNetChangesCommand()
	args := make([]string, 0)
	err := cmd.RunE(cmd, args)
	assert.NilError(t, err)
	output := strings.Split(strings.TrimSuffix(outputBuffer.String(), "\n"), "\n")
	assert.Assert(t, len(output) == 2*len(netChanges))

	testCases := []struct {
		description string
		index       int
		regexp      string
	}{
		{description: "Change 0", index: 0, regexp: `[0-9]{4}-[0-9]{2}-[0-9]{2} .* Change0 \(User1\)`},
		{description: "Hash 0", index: 1, regexp: `.+Change0: Hash0`},
		{description: "Change 1", index: 2, regexp: `[0-9]{4}-[0-9]{2}-[0-9]{2} .* Change1 \(User1\)`},
		{description: "Hash 1", index: 3, regexp: `.+Change1: Hash1`},
		{description: "Change 2", index: 4, regexp: `[0-9]{4}-[0-9]{2}-[0-9]{2} .* Change2 \(User1\)`},
		{description: "Hash 2", index: 5, regexp: `.+Change2: Hash2`},
		{description: "Change 3", index: 6, regexp: `[0-9]{4}-[0-9]{2}-[0-9]{2} .* Change3 \(User1\)`},
		{description: "Hash 3", index: 7, regexp: `.+Change3: Hash3`},
	}

	for _, testCase := range testCases {
		re := regexp.MustCompile(testCase.regexp)
		assert.Assert(t, re.MatchString(output[testCase.index]), "%s", testCase.description)
	}
}
