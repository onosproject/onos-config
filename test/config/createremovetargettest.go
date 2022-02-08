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

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/stretchr/testify/assert"
)

const (
	createRemoveTargetModPath       = "/system/clock/config/timezone-name"
	createRemoveTargetModValue1     = "Europe/Paris"
	createRemoveTargetModValue2     = "Europe/London"
	createRemoveTargetModTargetName = "reincarnated-target"
)

// TestCreatedRemovedTarget tests set/query of a single GNMI path to a single target that is created, removed, then created again
func (s *TestSuite) TestCreatedRemovedTarget(t *testing.T) {
	simulator := gnmiutils.CreateSimulatorWithName(t, createRemoveTargetModTargetName, true)
	assert.NotNil(t, simulator)

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(t, createRemoveTargetModTargetName, 1*time.Minute)
	assert.True(t, ready)

	targetPath := gnmiutils.GetTargetPathWithValue(createRemoveTargetModTargetName, createRemoveTargetModPath, createRemoveTargetModValue1, proto.StringVal)

	// Set a value using gNMI client - target is up
	c := gnmiutils.GetGNMIClientOrFail(t)
	_, _ = gnmiutils.SetGNMIValueOrFail(t, c, targetPath, gnmiutils.NoPaths, gnmiutils.SyncExtension(t))

	// Check that the value was set correctly
	gnmiutils.CheckGNMIValue(t, c, targetPath, createRemoveTargetModValue1, 0, "Query after set returned the wrong value")

	// interrogate the target to check that the value was set properly
	targetGnmiClient := gnmiutils.GetTargetGNMIClientOrFail(t, simulator)
	gnmiutils.CheckTargetValue(t, targetGnmiClient, targetPath, createRemoveTargetModValue1)

	//  Shut down the simulator
	gnmiutils.DeleteSimulator(t, simulator)
	unavailable := gnmiutils.WaitForTargetUnavailable(t, createRemoveTargetModTargetName, 2*time.Minute)
	assert.True(t, unavailable)

	// Set a value using gNMI client - target is down
	setPath2 := gnmiutils.GetTargetPathWithValue(createRemoveTargetModTargetName, createRemoveTargetModPath, createRemoveTargetModValue2, proto.StringVal)

	_, _ = gnmiutils.SetGNMIValueOrFail(t, c, setPath2, gnmiutils.NoPaths, gnmiutils.NoExtensions)

	//  Restart simulated target
	simulator = gnmiutils.CreateSimulatorWithName(t, createRemoveTargetModTargetName, false)
	assert.NotNil(t, simulator)

	// Wait for config to connect to the target
	ready = gnmiutils.WaitForTargetAvailable(t, createRemoveTargetModTargetName, 2*time.Minute)
	assert.True(t, ready)

	err := gnmiutils.WaitForConfigurationCompleteOrFail(t, configapi.ConfigurationID(simulator.Name()), time.Minute)
	assert.NoError(t, err)
	// Check that the value was set correctly
	gnmiutils.CheckGNMIValue(t, c, targetPath, createRemoveTargetModValue2, 0, "Query after set 2 returns wrong value")

	// interrogate the target to check that the value was set properly
	targetGnmiClient2 := gnmiutils.GetTargetGNMIClientOrFail(t, simulator)
	gnmiutils.CheckTargetValue(t, targetGnmiClient2, targetPath, createRemoveTargetModValue2)
	gnmiutils.DeleteSimulator(t, simulator)

}
