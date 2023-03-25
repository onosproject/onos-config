// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	helmit "github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test"
	"golang.org/x/net/context"
)

// TestSuite is the onos-config GNMI test suite
type TestSuite struct {
	test.Suite
	replicaCount int64
	simulator1   string
	simulator2   string
}

// SetupSuite sets up the onos-config GNMI test suite
func (s *TestSuite) SetupSuite(ctx context.Context) {
	release, err := s.InstallUmbrella().
		Set("import.onos-cli.enabled", false). // not needed - can be enabled by adding '--set onos-umbrella.import.onos-cli.enabled=true' to helmit args for investigations
		Set("onos-config.gnmiSet.sizeLimit", 0).
		Wait().
		Get(ctx)
	s.NoError(err)
	s.replicaCount = release.Get("onos-config.replicaCount").Int64()
}

// TearDownSuite tears down the test suite
func (s *TestSuite) TearDownSuite(ctx context.Context) {
	s.NoError(s.Helm().Uninstall("onos-umbrella").Do(ctx))
}

func (s *TestSuite) SetupTest(ctx context.Context) {
	simulators := s.SetupRandomSimulators(ctx, 2)
	s.simulator1 = simulators[0]
	s.simulator2 = simulators[1]
}

func (s *TestSuite) TearDownTest(ctx context.Context) {
	s.TearDownSimulators(ctx, s.simulator1, s.simulator2)
}

var _ helmit.SetupSuite = (*TestSuite)(nil)
var _ helmit.SetupTest = (*TestSuite)(nil)
var _ helmit.TearDownTest = (*TestSuite)(nil)
