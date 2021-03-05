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

package gnmi

import (
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-config/test/utils/charts"
	"github.com/onosproject/onos-test/pkg/onostest"
	"sync"

	"github.com/onosproject/helmit/pkg/test"
)

type testSuite struct {
	test.Suite
}

// TestSuite is the onos-config GNMI test suite
type TestSuite struct {
	testSuite
	mux sync.Mutex
}

// SetupTestSuite sets up the onos-config GNMI test suite
func (s *TestSuite) SetupTestSuite() error {
	devsim := helm.Chart("config-model-devicesim", onostest.OnosChartRepo).Release("config-model-devicesim").Install(true)
	if devsim != nil {
		return devsim
	}
	umbrella := charts.CreateUmbrellaRelease()
	return umbrella.Install(true)
}
