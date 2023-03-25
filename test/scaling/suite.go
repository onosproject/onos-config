// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package scaling

import (
	"context"
	helmit "github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test"
)

const gnmiSetLimitForTest = 10

// TestSuite is the onos-config GNMI test suite
type TestSuite struct {
	test.Suite
	replicaCount int64
	simulator    string
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

func (s *TestSuite) SetupTest(ctx context.Context) {
	s.simulator = s.SetupRandomSimulator(ctx, true)
}

func (s *TestSuite) TearDownTest(ctx context.Context) {
	s.TearDownSimulator(ctx, s.simulator)
	s.simulator = ""
}

var _ helmit.SetupSuite = (*TestSuite)(nil)
var _ helmit.SetupTest = (*TestSuite)(nil)
var _ helmit.TearDownTest = (*TestSuite)(nil)
