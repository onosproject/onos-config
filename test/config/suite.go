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

package config

import (
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test/utils/charts"
)

type testSuite struct {
	test.Suite
}

// TestSuite is the onos-config GNMI test suite
type TestSuite struct {
	testSuite
	ConfigReplicaCount int64
}

func getInt(value interface{}) int64 {
	if i, ok := value.(int); ok {
		return int64(i)
	} else if i, ok := value.(float64); ok {
		return int64(i)
	} else if i, ok := value.(int64); ok {
		return i
	}
	return 0
}

// SetupTestSuite sets up the onos-config GNMI test suite
func (s *TestSuite) SetupTestSuite(c *input.Context) error {
	registry := c.GetArg("registry").String("")
	umbrella := charts.CreateUmbrellaRelease()
	r := umbrella.
		Set("global.image.registry", registry).
		Set("import.onos-cli.enabled", false). // not needed - can be enabled by adding '--set onos-umbrella.import.onos-cli.enabled=true' to helmit args for investigations
		Install(true)
	s.ConfigReplicaCount = getInt(umbrella.Get("onos-config.replicaCount"))
	return r
}
