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
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/helmit/pkg/kubernetes"
	"github.com/onosproject/helmit/pkg/util/random"
	"github.com/onosproject/onos-test/pkg/onostest"
	"github.com/stretchr/testify/assert"
	"strconv"
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

type yangAttributes struct {
	name string
	file string
	revision string
	organization string
}

type pluginMetadata struct {
	name string
	version string
	yangs []yangAttributes
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

func parsePluginsCommandOutput(t *testing.T, output []string) []pluginMetadata {
	t.Helper()

	plugins := make([]pluginMetadata, 0)

	for lineIndex, line := range output {

		if strings.Contains(line, ":") {
			var newPlugin pluginMetadata
			// extract data from the plugin header
			newPlugin.name = strings.Fields(strings.Replace(line, ":", " ", -1))[0]
			headerData := strings.Fields(line)
			newPlugin.version = headerData[1]

			yangCount, err := strconv.Atoi(headerData[2])
			if err != nil {
				return plugins
			}
			newPlugin.yangs = make([]yangAttributes, yangCount)
			for yangIndex := 0; yangIndex < yangCount; yangIndex++ {
				yangData := strings.Fields(output[lineIndex+2+yangIndex])

				newPlugin.yangs[yangIndex].name = yangData[0]
				newPlugin.yangs[yangIndex].file = yangData[1]
				newPlugin.yangs[yangIndex].revision = yangData[2]

				var organization strings.Builder
				for i := 3; i < len(yangData); i++ {
					if i != 3 {
						organization.WriteString(" ")
					}
					organization.WriteString(yangData[i])
				}
				newPlugin.yangs[yangIndex].organization = organization.String()
			}
			lineIndex += yangCount + 2
			plugins = append(plugins, newPlugin)
		}
	}

	return plugins
}

// TestPluginsGetCLI tests the config service's plugin CLI commands
func (s *TestSuite) TestPluginsGetCLI(t *testing.T) {
	// Create a device simulator
	device1 := helm.
		Chart("device-simulator", "https://charts.onosproject.org").
		Release(random.NewPetName(2))
	err := device1.Install(true)
	assert.NoError(t, err)

	time.Sleep(60 * time.Second)

	// Get one of the onos-cli pods
	release := helm.Chart("onos-umbrella", onostest.OnosChartRepo).Release("onos-umbrella")
	client := kubernetes.NewForReleaseOrDie(release)
	dep, err := client.AppsV1().Deployments().Get("onos-cli")
	assert.NoError(t, err)
	pods, err := dep.Pods().List()
	assert.NoError(t, err)
	pod := pods[0]

	output, code, err := pod.Containers()[0].Exec("onos modelregistry list")
	assert.NoError(t, err)
	assert.Equal(t, 0, code)

	plugins := parsePluginsCommandOutput(t, output)

	testCases := []pluginMetadata{
		{
			name:    "Stratum",
			version: "1.0.0",
			yangs: []yangAttributes{
				{
					name:         "openconfig-hercules-interfaces",
					file:         "openconfig-hercules-interfaces.yang",
					revision:     "2018-06-01",
					organization: "OpenConfig working group",
				},
				{
					name:         "openconfig-platform",
					file:         "openconfig-platform.yang",
					revision:     "2019-04-16",
					organization: "OpenConfig working group",
				},
				{
					name:         "openconfig-hercules-platform-port",
					file:         "openconfig-hercules-platform-port.yang",
					revision:     "2018-06-01",
					organization: "OpenConfig working group",
				},
			},
		},
		{
			name:    "Devicesim",
			version: "1.0.0",
			yangs: []yangAttributes{
				{
					name: "openconfig-interfaces",
					file: "openconfig-interfaces.yang",
					revision: "2017-07-14",
					organization: "OpenConfig working group",
				},
				{
					name: "openconfig-openflow",
					file: "openconfig-openflow.yang",
					revision: "2017-06-01",
					organization: "OpenConfig working group",
				},
				{
					name: "openconfig-platform",
					file: "openconfig-platform.yang",
					revision: "2016-12-22",
					organization: "OpenConfig working group",
				},
				{
					name: "openconfig-system",
					file: "openconfig-system.yang",
					revision: "2017-07-06",
					organization: "OpenConfig working group",
				},
			},
		},
	}
	assert.NotNil(t, testCases)

	// Run the test cases
	for testCaseIndex := range testCases {
		testCase := testCases[testCaseIndex]
		description := makeDescription(testCase.name)
		t.Run(description,
			func(t *testing.T) {
				pluginFound := false
				for _, plugin := range plugins {
					if plugin.name == testCase.name && plugin.version == testCase.version {
						assert.False(t, pluginFound, "Plugin already found")
						pluginFound = true
						for _, testCaseYang := range testCase.yangs {
							yangFound := false
							for _, pluginYang := range plugin.yangs {
								if testCaseYang.name == pluginYang.name {
									assert.False(t, yangFound, "Yang already found")
									assert.Equal(t, testCaseYang.organization, pluginYang.organization)
									assert.Equal(t, testCaseYang.revision, pluginYang.revision)
									assert.Equal(t, testCaseYang.file, pluginYang.file)
									yangFound = true
								}
							}
							assert.True(t, yangFound, "Yang not found %s", testCaseYang.name)
						}
					}
				}
				assert.True(t, pluginFound, "Plugin %s not found", testCase.name)
			})
	}
}
