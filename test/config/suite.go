// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/onosproject/helmit/pkg/helm"
	helmit "github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test"
)

// TestSuite is the onos-config GNMI test suite
type TestSuite struct {
	test.Suite
	umbrella   *helm.Release
	simulator1 *helm.Release
	simulator2 *helm.Release
}

// SetupSuite sets up the onos-config GNMI test suite
func (s *TestSuite) SetupSuite() {
	registry := s.Arg("registry").String()

	install := s.InstallUmbrella().
		Set("import.onos-cli.enabled", false). // not needed - can be enabled by adding '--set onos-umbrella.import.onos-cli.enabled=true' to helmit args for investigations
		Set("onos-config.gnmiSet.sizeLimit", 0)

	if registry != "" {
		install.Set("onos-topo.global.image.registry", registry).
			Set("onos-config.global.image.registry", registry).
			Set("onos-umbrella.global.image.registry", registry).
			Set("topo-discovery.global.image.registry", registry).
			Set("device-provisioner.global.image.registry", registry).
			Set("onos-cli.global.image.registry", registry)
	}
	
	release, err := install.
		Wait().
		Get(s.Context())
	s.NoError(err)
	s.umbrella = release
}

// TearDownSuite tears down the test suite
func (s *TestSuite) TearDownSuite() {
	s.NoError(s.Helm().Uninstall(s.umbrella.Name).Do(s.Context()))
}

// SetupTest sets up simulators for tests
func (s *TestSuite) SetupTest() {
	simulators := s.SetupRandomSimulators(2)
	s.simulator1 = simulators[0]
	s.simulator2 = simulators[1]
}

// TearDownTest tears down simulators for tests
func (s *TestSuite) TearDownTest() {
	s.TearDownSimulators(s.simulator1.Name, s.simulator2.Name)
}

var _ helmit.SetupSuite = (*TestSuite)(nil)
var _ helmit.SetupTest = (*TestSuite)(nil)
var _ helmit.TearDownTest = (*TestSuite)(nil)
