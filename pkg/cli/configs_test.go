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
	"fmt"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/store"
	"gotest.tools/assert"
	"io"
	"strings"
	"testing"
	"time"
)

var configurations []diags.Configuration
var changes []admin.Change

const version = "1.0.0"
const deviceType = "TestDevice"

func generateConfigurationData(count int) {
	configurations = make([]diags.Configuration, count)
	now := time.Now()
	for cfgIdx := range configurations {
		deviceID := fmt.Sprintf("device-%d", cfgIdx)

		changeIds := make([]string, count)
		for i := 0; i < count; i++ {
			changeIds[i] = store.B64([]byte(fmt.Sprintf("change-%d", i)))
		}

		configurations[cfgIdx] = diags.Configuration{
			Name:       fmt.Sprintf("%s-%s", deviceID, version),
			DeviceID:   deviceID,
			Version:    version,
			DeviceType: deviceType,
			Created:    &now,
			Updated:    &now,
			ChangeIDs:  changeIds,
		}
	}
}

func generateChangeData(count int) {
	changes = make([]admin.Change, count)
	now := time.Now()
	for chIdx := range changes {
		changeValues := make([]*admin.ChangeValue, count)
		for cvIdx := range changeValues {
			cv := admin.ChangeValue{
				Path:      fmt.Sprintf("/root/leaf-%d", cvIdx),
				Value:     []byte("Test"),
				ValueType: admin.ChangeValueType_STRING,
			}
			changeValues[cvIdx] = &cv
		}

		changes[chIdx] = admin.Change{
			Time:         &now,
			Id:           store.B64([]byte(fmt.Sprintf("change-%d", chIdx))),
			Desc:         fmt.Sprintf("Test change-%d", chIdx),
			ChangeValues: changeValues,
		}
	}
}

var nextConfigIndex = 0
var nextChangeIndex = 0

func recvCfgsMock() (*diags.Configuration, error) {
	if nextConfigIndex < len(configurations) {
		cfg := configurations[nextConfigIndex]
		nextConfigIndex++

		return &cfg, nil
	}
	return nil, io.EOF
}

func recvChangessMock() (*admin.Change, error) {
	if nextChangeIndex < len(changes) {
		change := changes[nextChangeIndex]
		nextChangeIndex++

		return &change, nil
	}
	return nil, io.EOF
}

func Test_GetConfiguration(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)
	generateConfigurationData(4)
	generateChangeData(4)

	configsClient := MockConfigDiagsGetConfigurationsClient{
		recvFn: recvCfgsMock,
	}

	setUpMockClients(MockClientsConfig{
		configDiagsClientConfigurations: &configsClient,
	})

	configs := getGetConfigsCommand()
	err := configs.RunE(configs, nil)
	assert.NilError(t, err)
	output := outputBuffer.String()
	assert.Equal(t, strings.Count(output, "-1.0.0"), len(configurations))
	assert.Equal(t, strings.Count(output, "(device-"), len(configurations))
	assert.Equal(t, strings.Count(output, "Y2hhbmdlLTA="), len(configurations))
}

func Test_GetChanges(t *testing.T) {
	outputBuffer := bytes.NewBufferString("")
	CaptureOutput(outputBuffer)
	generateConfigurationData(4)
	generateChangeData(4)

	changesClient := MockConfigDiagsGetChangesClient{
		recvFn: recvChangessMock,
	}

	setUpMockClients(MockClientsConfig{
		configDiagsClientChanges: &changesClient,
	})

	changes := getGetChangesCommand()
	err := changes.RunE(changes, nil)
	assert.NilError(t, err)
	output := outputBuffer.String()
	assert.Equal(t, strings.Count(output, "CHANGE"), len(configurations))
	assert.Equal(t, strings.Count(output, "|(STRING) Test"), len(configurations)*len(configurations))
	assert.Equal(t, strings.Count(output, "|/root/leaf-0"), len(configurations))
}
