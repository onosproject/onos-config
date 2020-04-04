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
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/util/random"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

type pluginAttributes struct {
	pluginVersion string
	pluginSource  string
}

type pluginsTestCase struct {
	pluginName    string
	pluginVersion string
	pluginObject  string
	yangName      string
	attributes    pluginAttributes
}

// makeKey : The plugin key is represented as name-version-sharedObjectName
func makeKey(pluginName string, pluginVersion string, pluginObject string) string {
	return pluginName + "-" + pluginVersion + "-" + pluginObject
}

func makeDescription(path string) string {
	if len(path) <= 25 {
		return path
	}
	return "..." + path[len(path)-22:]
}

func parsePluginsCommandOutput(t *testing.T, output []string) map[string]map[string]pluginAttributes {
	t.Helper()
	var pluginKey string
	plugins := make(map[string]map[string]pluginAttributes)
	for _, line := range output {
		if strings.HasPrefix(line, "YANGS:") ||
			len(line) == 0 {
			//  Skip YANGS: header and separators
			continue
		}

		if strings.Contains(line, ":") {
			// This is a new plugin
			pluginName := line[:strings.Index(line, ":")]
			tokens := strings.Split(line, " ")
			if len(tokens) < 4 {
				//  This is not a plugin description; ignore it
				continue
			}
			pluginVersion := tokens[1]
			pluginObject := tokens[3]
			pluginKey = makeKey(pluginName, pluginVersion, pluginObject)
		} else {
			// This is a YANG description
			yangTokens := strings.Split(line, "	")
			yangName := yangTokens[1]
			yangVersion := yangTokens[2]
			yangSource := yangTokens[3]

			if plugins[pluginKey] == nil {
				plugins[pluginKey] = make(map[string]pluginAttributes)
			}

			p := pluginAttributes{
				pluginVersion: yangVersion,
				pluginSource:  yangSource,
			}

			plugins[pluginKey][yangName] = p
		}
	}

	return plugins
}

// TestPluginsGetCLI tests the config service's plugin CLI commands
func (s *TestSuite) TestPluginsGetCLI(t *testing.T) {
	// Create a device simulator
	device1 := helm.
		Chart("device-simulator").
		Release(random.NewPetName(2))
	err := device1.Install(true)
	assert.NoError(t, err)

	time.Sleep(60 * time.Second)

	// Get one of the onos-cli pods
	release := helm.Chart("onos-cli").
		Release("onos-cli")
	client := kubernetes.NewForReleaseOrDie(release)
	pods, err := client.
		CoreV1().
		Pods().
		List()
	assert.NoError(t, err)
	pod := pods[0]

	output, code, err := pod.Containers()[0].Exec(fmt.Sprintf("onos config get plugins %s", device1.Name()))
	assert.NoError(t, err)
	assert.Equal(t, 0, code)

	plugins := parsePluginsCommandOutput(t, output)

	testCases := []pluginsTestCase{
		{
			pluginName:    "Stratum",
			pluginVersion: "1.0.0",
			pluginObject:  "stratum.so.1.0.0",
			yangName:      "openconfig-interfaces-stratum",
			attributes: pluginAttributes{
				pluginVersion: "0.1.0",
				pluginSource:  "Open Networking Foundation",
			},
		},
		{
			pluginName:    "TestDevice",
			pluginVersion: "1.0.0",
			pluginObject:  "testdevice.so.1.0.0",
			yangName:      "test1",
			attributes: pluginAttributes{
				pluginVersion: "2018-02-20",
				pluginSource:  "Open Networking Foundation",
			},
		},
		{
			pluginName:    "TestDevice",
			pluginVersion: "2.0.0",
			pluginObject:  "testdevice.so.2.0.0",
			yangName:      "test1",
			attributes: pluginAttributes{
				pluginVersion: "2019-06-10",
				pluginSource:  "Open Networking Foundation",
			},
		},
		{
			pluginName:    "Devicesim",
			pluginVersion: "1.0.0",
			pluginObject:  "devicesim.so.1.0.0",
			yangName:      "openconfig-openflow",
			attributes: pluginAttributes{
				pluginVersion: "2017-06-01",
				pluginSource:  "OpenConfig working group",
			},
		},
		{
			pluginName:    "Stratum",
			pluginVersion: "1.0.0",
			pluginObject:  "stratum.so.1.0.0",
			yangName:      "openconfig-if-ip",
			attributes: pluginAttributes{
				pluginVersion: "3.0.0",
				pluginSource:  "OpenConfig working group",
			},
		},
		{
			pluginName:    "Stratum",
			pluginVersion: "1.0.0",
			pluginObject:  "stratum.so.1.0.0",
			yangName:      "openconfig-hercules-platform-node",
			attributes: pluginAttributes{
				pluginVersion: "0.2.0",
				pluginSource:  "OpenConfig working group",
			},
		},
	}

	// Run the test cases
	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		description := makeDescription(testCase.yangName)
		t.Run(description,
			func(t *testing.T) {
				t.Parallel()
				pluginName := makeKey(testCase.pluginName, testCase.pluginVersion, testCase.pluginObject)
				expectedPluginVersion := testCase.attributes.pluginVersion
				expectedPluginSource := testCase.attributes.pluginSource
				assert.NotNil(t, plugins[pluginName])
				pluginVersion := plugins[pluginName][testCase.yangName].pluginVersion
				pluginSource := plugins[pluginName][testCase.yangName].pluginSource
				assert.Equal(t, expectedPluginSource, pluginSource)
				assert.Equal(t, expectedPluginVersion, pluginVersion)
			})
	}
}
