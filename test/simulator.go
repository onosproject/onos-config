// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"context"
	petname "github.com/dustinkirkland/golang-petname"
	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-test/pkg/onostest"
	"time"
)

const (
	// SimulatorTargetVersion default version for simulated target
	SimulatorTargetVersion = "1.0.0"
	// SimulatorTargetType type for simulated target
	SimulatorTargetType = "devicesim"

	defaultGNMITimeout = time.Second * 30
)

// SetupSimulators sets up the given number of device simulators
func (s *Suite) SetupSimulators(ctx context.Context, names ...string) []*helm.Release {
	releases, err := callAsync[*helm.Release](len(names), func(i int) (*helm.Release, error) {
		return s.setupSimulator(ctx, names[i], true)
	})
	s.NoError(err)
	return releases
}

// SetupRandomSimulators creates the given number of randomly named simulators
func (s *Suite) SetupRandomSimulators(ctx context.Context, num int) []*helm.Release {
	var names []string
	for i := 0; i < num; i++ {
		names = append(names, petname.Generate(2, "-"))
	}
	return s.SetupSimulators(ctx, names...)
}

// SetupRandomSimulator creates a simulator with a random name
func (s *Suite) SetupRandomSimulator(ctx context.Context, createTopoEntity bool) *helm.Release {
	name := petname.Generate(2, "-")
	return s.SetupSimulator(ctx, name, createTopoEntity)
}

// SetupSimulator creates a device simulator
func (s *Suite) SetupSimulator(ctx context.Context, name string, createTopoEntity bool) *helm.Release {
	release, err := s.setupSimulator(ctx, name, createTopoEntity)
	s.NoError(err)
	return release
}

// setupSimulator creates a device simulator
func (s *Suite) setupSimulator(ctx context.Context, name string, createTopoEntity bool) (*helm.Release, error) {
	release, err := s.Helm().Install(name, "device-simulator").
		RepoURL(onostest.OnosChartRepo).
		Set("image.tag", "latest").
		Wait().
		Get(ctx)
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	if createTopoEntity {
		simulatorTarget, err := s.NewSimulatorTargetEntity(name, SimulatorTargetType, SimulatorTargetVersion)
		s.NoError(err, "could not make target for simulator %v", err)

		err = s.AddTargetToTopo(ctx, simulatorTarget)
		s.NoError(err, "could not add target to topo for simulator %v", err)
	}
	return release, nil
}

// TearDownSimulators deletes all the simulator pods
func (s *Suite) TearDownSimulators(ctx context.Context, names ...string) {
	s.NoError(iterAsync(len(names), func(i int) error {
		return s.tearDownSimulator(ctx, names[i])
	}))
}

// TearDownSimulator shuts down the simulator pod and removes the target from topology
func (s *Suite) TearDownSimulator(ctx context.Context, name string) {
	s.NoError(s.tearDownSimulator(ctx, name))
}

// tearDownSimulator shuts down the simulator pod and removes the target from topology
func (s *Suite) tearDownSimulator(ctx context.Context, name string) error {
	return s.Helm().Uninstall(name).Do(ctx)
}
