package test

import (
	"context"
	petname "github.com/dustinkirkland/golang-petname"
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
func (s *Suite) SetupSimulators(ctx context.Context, names ...string) {
	s.NoError(iterAsync(len(names), func(i int) error {
		return s.setupSimulator(ctx, names[i], true)
	}))
}

// SetupRandomSimulators creates the given number of randomly named simulators
func (s *Suite) SetupRandomSimulators(ctx context.Context, num int) []string {
	var names []string
	for i := 0; i < num; i++ {
		names = append(names, petname.Generate(2, "-"))
	}
	s.SetupSimulators(ctx, names...)
	return names
}

// SetupRandomSimulator creates a simulator with a random name
func (s *Suite) SetupRandomSimulator(ctx context.Context, createTopoEntity bool) string {
	name := petname.Generate(2, "-")
	s.SetupSimulator(ctx, name, createTopoEntity)
	return name
}

// SetupSimulator creates a device simulator
func (s *Suite) SetupSimulator(ctx context.Context, name string, createTopoEntity bool) {
	s.NoError(s.setupSimulator(ctx, name, createTopoEntity))
}

// setupSimulator creates a device simulator
func (s *Suite) setupSimulator(ctx context.Context, name string, createTopoEntity bool) error {
	err := s.Helm().Install(name, "device-simulator").
		RepoURL(onostest.OnosChartRepo).
		Set("image.tag", "latest").
		Wait().
		Do(ctx)
	if err != nil {
		return err
	}

	time.Sleep(2 * time.Second)

	if createTopoEntity {
		simulatorTarget, err := s.NewSimulatorTargetEntity(name, SimulatorTargetType, SimulatorTargetVersion)
		s.NoError(err, "could not make target for simulator %v", err)

		err = s.AddTargetToTopo(ctx, simulatorTarget)
		s.NoError(err, "could not add target to topo for simulator %v", err)
	}
	return nil
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
