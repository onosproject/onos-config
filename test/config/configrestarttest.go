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

package config

import (
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	hautils "github.com/onosproject/onos-config/test/utils/ha"
	"github.com/onosproject/onos-config/test/utils/proto"
	"testing"
)

const (
	restartTzValue = "Europe/Milan"
	restartTzPath  = "/system/clock/config/timezone-name"
)

// TestGetOperationAfterNodeRestart tests a Get operation after restarting the onos-config node
func (s *TestSuite) TestGetOperationAfterNodeRestart(t *testing.T) {
	// Create a simulated target
	simulator := gnmiutils.CreateSimulator(t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Make a GNMI client to use for onos-config requests
	gnmiClient := gnmiutils.GetGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)

	targetPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), restartTzPath, restartTzValue, proto.StringVal)

	// Set a value using onos-config
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, targetPath, gnmiutils.NoPaths, gnmiutils.SyncExtension(t))

	// Check that the value was set correctly
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, targetPath, restartTzValue, 0, "Query after set returned the wrong value")

	// Restart onos-config
	configPod := hautils.FindPodWithPrefix(t, "onos-config")
	hautils.CrashPodOrFail(t, configPod)

	// Check that the value was set correctly in the new onos-config instance
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, targetPath, restartTzValue, 0, "Query after restart returned the wrong value")

	// Check that the value is set on the target
	targetGnmiClient := gnmiutils.GetTargetGNMIClientOrFail(ctx, t, simulator)
	gnmiutils.CheckTargetValue(ctx, t, targetGnmiClient, targetPath, restartTzValue)
}

// TestSetOperationAfterNodeRestart tests a Set operation after restarting the onos-config node
func (s *TestSuite) TestSetOperationAfterNodeRestart(t *testing.T) {
	// Create a simulated target
	simulator := gnmiutils.CreateSimulator(t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Make a GNMI client to use for onos-config requests
	gnmiClient := gnmiutils.GetGNMIClientOrFail(ctx, t, gnmiutils.WithRetry)

	targetPath := gnmiutils.GetTargetPathWithValue(simulator.Name(), restartTzPath, restartTzValue, proto.StringVal)

	// Restart onos-config
	configPod := hautils.FindPodWithPrefix(t, "onos-config")
	hautils.CrashPodOrFail(t, configPod)

	// Set a value using onos-config
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, targetPath, gnmiutils.NoPaths, gnmiutils.SyncExtension(t))

	// Check that the value was set correctly
	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, targetPath, restartTzValue, 0, "Query after set returned the wrong value")

	// Check that the value is set on the target
	targetGnmiClient := gnmiutils.GetTargetGNMIClientOrFail(ctx, t, simulator)
	gnmiutils.CheckTargetValue(ctx, t, targetGnmiClient, targetPath, restartTzValue)
}
