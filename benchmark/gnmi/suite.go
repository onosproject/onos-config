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

package gnmi

import (
	"fmt"
	"github.com/onosproject/onos-test/pkg/benchmark"
	"github.com/onosproject/onos-test/pkg/helm"
	"github.com/onosproject/onos-test/pkg/input"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/onosproject/onos-test/pkg/util/random"
	"github.com/openconfig/gnmi/client/gnmi"
)

// BenchmarkSuite is an onos-config gNMI benchmark suite
type BenchmarkSuite struct {
	benchmark.Suite
	simulator *helm.Release
	client    *client.Client
	value     input.Source
}

// SetupSuite :: benchmark
func (s *BenchmarkSuite) SetupSuite(c *benchmark.Context) error {
	namespace := helm.Namespace()

	// Setup the Atomix controller
	err := namespace.Chart("/etc/charts/atomix-controller").
		Release("atomix-controller").
		Set("namespace", namespace.Namespace()).
		Install(true)
	if err != nil {
		return err
	}

	// Install the onos-topo chart
	err = namespace.Chart("/etc/charts/onos-topo").
		Release("onos-topo").
		Set("replicaCount", 2).
		Set("store.controller", fmt.Sprintf("atomix-controller.%s.svc.cluster.local:5679", namespace.Namespace())).
		Install(false)
	if err != nil {
		return err
	}

	// Install the onos-config chart
	err = namespace.Chart("/etc/charts/onos-config").
		Release("onos-config").
		Set("replicaCount", 2).
		Set("store.controller", fmt.Sprintf("atomix-controller.%s.svc.cluster.local:5679", namespace.Namespace())).
		Install(true)
	if err != nil {
		return err
	}
	return nil
}

// SetupWorker :: benchmark
func (s *BenchmarkSuite) SetupWorker(c *benchmark.Context) error {
	s.value = input.RandomString(8)
	s.simulator = helm.Namespace().
		Chart("/etc/charts/device-simulator").
		Release(random.NewPetName(2))
	if err := s.simulator.Install(true); err != nil {
		return err
	}
	client, err := env.Config().NewGNMIClient()
	if err != nil {
		panic(err)
	}
	s.client = client
	return nil
}

// TearDownWorker :: benchmark
func (s *BenchmarkSuite) TearDownWorker(c *benchmark.Context) error {
	s.client.Close()
	return s.simulator.Uninstall()
}

var _ benchmark.SetupWorker = &BenchmarkSuite{}
