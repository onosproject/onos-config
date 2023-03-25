// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package scaling

import (
	"context"
	"github.com/onosproject/helmit/pkg/helm"
	helmit "github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test"
)

const gnmiSetLimitForTest = 10

// TestSuite is the scaling test suite.
type TestSuite struct {
	test.Suite
	umbrella  *helm.Release
	simulator *helm.Release
}

// SetupSuite sets up the onos-config GNMI test suite
func (s *TestSuite) SetupSuite(ctx context.Context) {
	release, err := s.InstallUmbrella().
		Set("import.onos-cli.enabled", false). // not needed - can be enabled by adding '--set onos-umbrella.import.onos-cli.enabled=true' to helmit args for investigations
		Set("onos-config.gnmiSet.sizeLimit", 0).
		Wait().
		Get(ctx)
	s.NoError(err)
	s.umbrella = release
}

// TearDownSuite tears down the test suite
func (s *TestSuite) TearDownSuite(ctx context.Context) {
	s.NoError(s.Helm().Uninstall(s.umbrella.Name).Do(ctx))
}

// SetupTest sets up a simulator for tests
func (s *TestSuite) SetupTest(ctx context.Context) {
	s.simulator = s.SetupRandomSimulator(ctx, true)
}

// TearDownTest tears down the simulator for tests
func (s *TestSuite) TearDownTest(ctx context.Context) {
	s.TearDownSimulator(ctx, s.simulator.Name)
	s.simulator = nil
}

var _ helmit.SetupSuite = (*TestSuite)(nil)
var _ helmit.SetupTest = (*TestSuite)(nil)
var _ helmit.TearDownTest = (*TestSuite)(nil)
