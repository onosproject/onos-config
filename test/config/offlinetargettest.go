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
//

package config

import (
	"testing"
	"time"

	"github.com/onosproject/onos-test/pkg/onostest"
	"github.com/stretchr/testify/assert"

	"github.com/onosproject/helmit/pkg/helm"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
)

const (
	modPath           = "/system/clock/config/timezone-name"
	modValue          = "Europe/Rome"
	offlineTargetName = "test-offline-device-1"
)

// TestOfflineTarget tests set/query of a single GNMI path to a single target that is initially not in the config
func (s *TestSuite) TestOfflineTarget(t *testing.T) {
	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.GetGNMIClientOrFail(t)
	// create a target entity in topo
	createOfflineTarget(t, offlineTargetName, "devicesim-1.0.x", "1.0.0", "")
	// Sends a set request using onos-config NB
	targetPath := gnmiutils.GetTargetPathWithValue(offlineTargetName, modPath, modValue, proto.StringVal)
	gnmiutils.SetGNMIValueOrFail(t, gnmiClient, targetPath, gnmiutils.NoPaths, []*gnmi_ext.Extension{})

	// Install simulator
	simulator := helm.
		Chart("device-simulator", onostest.OnosChartRepo).
		Release(offlineTargetName).
		Set("image.tag", "latest")
	err := simulator.Install(true)
	assert.NoError(t, err, "could not install device simulator %v", err)

	// Wait for config to connect to the target
	gnmiutils.WaitForTargetAvailable(t, topoapi.ID(simulator.Name()), time.Minute)
	gnmiutils.CheckGNMIValue(t, gnmiClient, targetPath, modValue, 0, "Query after set returned the wrong value")

	// Check that the value was set properly on the target, wait for configuration gets completed
	targetGnmiClient := gnmiutils.GetTargetGNMIClientOrFail(t, simulator)
	gnmiutils.CheckTargetValue(t, targetGnmiClient, targetPath, modValue)
	err = simulator.Uninstall()
	assert.NoError(t, err)
}
