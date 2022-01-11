// Copyright 2020-present Open Networking Foundation.
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

package rbac

import (
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test/utils/charts"
)

type testSuite struct {
	test.Suite
}

// TestSuite is the onos-config HA test suite
type TestSuite struct {
	testSuite
}

// SetupTestSuite sets up the onos-config HA test suite
func (s *TestSuite) SetupTestSuite(c *input.Context) error {
	umbrella := charts.CreateUmbrellaRelease().
		Set("onos-config.openidc.issuer", "https://keycloak-dev.onlab.us/auth/realms/master")
	return umbrella.Install(true)
}
