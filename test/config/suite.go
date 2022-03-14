// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/onosproject/helmit/pkg/input"
	"github.com/onosproject/helmit/pkg/test"
	"github.com/onosproject/onos-config/test/utils/charts"
)

type testSuite struct {
	test.Suite
}

// TestSuite is the onos-config GNMI test suite
type TestSuite struct {
	testSuite
	ConfigReplicaCount int64
}

func getInt(value interface{}) int64 {
	if i, ok := value.(int); ok {
		return int64(i)
	} else if i, ok := value.(float64); ok {
		return int64(i)
	} else if i, ok := value.(int64); ok {
		return i
	}
	return 0
}

// SetupTestSuite sets up the onos-config GNMI test suite
func (s *TestSuite) SetupTestSuite(c *input.Context) error {
	registry := c.GetArg("registry").String("")
	umbrella := charts.CreateUmbrellaRelease()
	r := umbrella.
		Set("global.image.registry", registry).
		Set("import.onos-cli.enabled", false). // not needed - can be enabled by adding '--set onos-umbrella.import.onos-cli.enabled=true' to helmit args for investigations
		Install(true)
	s.ConfigReplicaCount = getInt(umbrella.Get("onos-config.replicaCount"))
	return r
}
