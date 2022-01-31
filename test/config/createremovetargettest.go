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

	"github.com/onosproject/onos-config/test/utils/gnmi"
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
	simulator := gnmi.CreateSimulatorWithName(t, createRemoveTargetModTargetName, true)
	assert.NotNil(t, simulator)
	defer gnmi.DeleteSimulator(t, simulator)

	// Wait for config to connect to the target
	ready := gnmi.WaitForTargetAvailable(t, createRemoveTargetModTargetName, 1*time.Minute)
	assert.True(t, ready)

	targetPath := gnmi.GetTargetPathWithValue(createRemoveTargetModTargetName, createRemoveTargetModPath, createRemoveTargetModValue1, proto.StringVal)

	// Set a value using gNMI client - target is up
	c := gnmi.GetGNMIClientOrFail(t)
	_, _ = gnmi.SetGNMIValueOrFail(t, c, targetPath, gnmi.NoPaths, gnmi.NoExtensions)

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, c, targetPath, createRemoveTargetModValue1, 0, "Query after set returned the wrong value")

	// interrogate the target to check that the value was set properly
	targetGnmiClient := gnmi.GetTargetGNMIClientOrFail(t, simulator)
	gnmi.CheckTargetValue(t, targetGnmiClient, targetPath, createRemoveTargetModValue1)

	//  Shut down the simulator
	gnmi.DeleteSimulator(t, simulator)
	gnmi.WaitForTargetUnavailable(t, createRemoveTargetModTargetName, 1*time.Minute)

	// Set a value using gNMI client - target is down
	setPath2 := gnmi.GetTargetPathWithValue(createRemoveTargetModTargetName, createRemoveTargetModPath, createRemoveTargetModValue2, proto.StringVal)
	_, _ = gnmi.SetGNMIValueOrFail(t, c, setPath2, gnmi.NoPaths, gnmi.NoExtensions)

	//  Restart simulated target
	simulator = gnmi.CreateSimulatorWithName(t, createRemoveTargetModTargetName, false)
	assert.NotNil(t, simulator)

	// Wait for config to connect to the target
	ready = gnmi.WaitForTargetAvailable(t, createRemoveTargetModTargetName, 2*time.Minute)
	assert.True(t, ready)

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, c, targetPath, createRemoveTargetModValue2, 0, "Query after set 2 returns wrong value")

	// interrogate the target to check that the value was set properly
	targetGnmiClient2 := gnmi.GetTargetGNMIClientOrFail(t, simulator)
	gnmi.CheckTargetValue(t, targetGnmiClient2, targetPath, createRemoveTargetModValue2)
}
