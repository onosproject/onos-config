// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"fmt"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-test/pkg/onostest"
	"os"
	"time"
)

const (
	// SimulatorTargetVersion default version for simulated target
	SimulatorTargetVersion = "1.0.x"
	// SimulatorTargetType type for simulated target
	SimulatorTargetType = "devicesim"

	defaultGNMITimeout = time.Second * 30
)

// SetupSimulators sets up the given number of device simulators
func (s *Suite) SetupSimulators(names ...string) []*helm.Release {
	releases, err := callAsync[*helm.Release](len(names), func(i int) (*helm.Release, error) {
		return s.setupSimulator(names[i], true)
	})
	s.NoError(err)
	return releases
}

// SetupRandomSimulators creates the given number of randomly named simulators
func (s *Suite) SetupRandomSimulators(num int) []*helm.Release {
	var names []string
	for i := 0; i < num; i++ {
		names = append(names, petname.Generate(2, "-"))
	}
	return s.SetupSimulators(names...)
}

// SetupRandomSimulator creates a simulator with a random name
func (s *Suite) SetupRandomSimulator(createTopoEntity bool) *helm.Release {
	name := petname.Generate(2, "-")
	return s.SetupSimulator(name, createTopoEntity)
}

// SetupSimulator creates a device simulator
func (s *Suite) SetupSimulator(name string, createTopoEntity bool) *helm.Release {
	release, err := s.setupSimulator(name, createTopoEntity)
	s.NoError(err)
	return release
}

// setupSimulator creates a device simulator
func (s *Suite) setupSimulator(name string, createTopoEntity bool) (*helm.Release, error) {
	registry := s.Arg("registry").String()
	install := s.Helm().Install(name, "device-simulator").
		RepoURL(onostest.OnosChartRepo).
		Set("image.tag", "latest")
	if registry != "" {
		fmt.Fprintf(os.Stderr, "Registry is set to %s", registry)
		install.Set("install.image.repository", registry+"/onosproject/device-simulator")
	}
	release, err := install.
		Wait().
		Get(s.Context())
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	if createTopoEntity {
		simulatorTarget, err := s.NewSimulatorTargetEntity(name, SimulatorTargetType, SimulatorTargetVersion)
		s.NoError(err, "could not make target for simulator %v", err)

		err = s.AddTargetToTopo(simulatorTarget)
		s.NoError(err, "could not add target to topo for simulator %v", err)
	}
	return release, nil
}

// TearDownSimulators deletes all the simulator pods
func (s *Suite) TearDownSimulators(names ...string) {
	s.NoError(iterAsync(len(names), func(i int) error {
		return s.tearDownSimulator(names[i])
	}))
}

// TearDownSimulator shuts down the simulator pod and removes the target from topology
func (s *Suite) TearDownSimulator(name string) {
	s.NoError(s.tearDownSimulator(name))
}

// tearDownSimulator shuts down the simulator pod and removes the target from topology
func (s *Suite) tearDownSimulator(name string) error {
	return s.Helm().Uninstall(name).Do(s.Context())
}
