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
	"fmt"
	"github.com/onosproject/onos-test/pkg/helm"
	"github.com/onosproject/onos-test/pkg/test"
)

// TestSuite is the primary onos-config test suite
type TestSuite struct {
	test.Suite
}

// SetupTestSuite sets up the onos-config gNMI test suite
func (s *TestSuite) SetupTestSuite() error {
	// Setup the Atomix controller
	err := helm.Helm().
		Chart("/etc/charts/atomix-controller").
		Release("atomix-controller").
		Set("namespace", helm.Namespace()).
		Install(true)
	if err != nil {
		return err
	}

	// Install the onos-topo chart
	err = helm.Helm().
		Chart("/etc/charts/onos-topo").
		Release("onos-topo").
		Set("replicaCount", 2).
		Set("store.controller", fmt.Sprintf("atomix-controller.%s.svc.cluster.local:5679", helm.Namespace())).
		Install(false)
	if err != nil {
		return err
	}

	// Install the onos-config chart
	err = helm.Helm().
		Chart("/etc/charts/onos-config").
		Release("onos-config").
		Set("replicaCount", 2).
		Set("store.controller", fmt.Sprintf("atomix-controller.%s.svc.cluster.local:5679", helm.Namespace())).
		Install(true)
	if err != nil {
		return err
	}
	return nil
}

var _ test.SetupTestSuite = &TestSuite{}
