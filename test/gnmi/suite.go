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
	"github.com/onosproject/helmit/pkg/test"
)

type testSuite struct {
	test.Suite
}

// TestSuite is the onos-config CLI test suite
type TestSuite struct {
	testSuite
}

const (
	atomixChartRepo      = "https://charts.atomix.io"
	onosChartRepo        = "https://charts.onosproject.org"
	atomixName           = "cli-test-onos-config-atomix"
	atomixControllerName = atomixName + "-" + "kubernetes-controller"
	atomixControllerPort = "5679"
	atomixController     = atomixControllerName + ":" + atomixControllerPort
)

// SetupTestSuite sets up the onos-config CLI test suite
func (s *TestSuite) SetupTestSuite() error {
	err := helm.Chart("kubernetes-controller", atomixChartRepo).
		Release(atomixName).
		Set("scope", "Namespace").
		Install(true)
	if err != nil {
		return err
	}

	err = helm.Chart("raft-storage-controller", atomixChartRepo).
		Release("onos-config-raft").
		Set("scope", "Namespace").
		Install(true)
	if err != nil {
		return err
	}

	err = helm.Chart("onos-topo", onosChartRepo).
		Release("onos-topo").
		Set("image.tag", "latest").
		Set("storage.controller", atomixController).
		Install(true)
	if err != nil {
		return err
	}

	err = helm.Chart("onos-config", onosChartRepo).
		Release("onos-config").
		Set("image.tag", "latest").
		Set("storage.controller", atomixController).
		Install(true)
	if err != nil {
		return err
	}

	return nil
}
