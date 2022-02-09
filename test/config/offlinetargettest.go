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
	"context"
	"testing"
	"time"

	configapi "github.com/onosproject/onos-api/go/onos/config/v2"

	"github.com/stretchr/testify/assert"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
)

const (
	modPath           = "/system/clock/config/timezone-name"
	modValue          = "Europe/Rome"
	offlineTargetName = "offline-target-device-simulator"
)

// TestOfflineTarget tests set/query of a single GNMI path to a single target that is initially not connected to onos-config
func (s *TestSuite) TestOfflineTarget(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// create a target entity in topo
	createOfflineTarget(t, offlineTargetName, "devicesim", "1.0.0", offlineTargetName+":11161")

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.GetGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Sends a set request using onos-config NB
	targetPath := gnmiutils.GetTargetPathWithValue(offlineTargetName, modPath, modValue, proto.StringVal)
	gnmiutils.SetGNMIValueOrFail(ctx, t, gnmiClient, targetPath, gnmiutils.NoPaths, gnmiutils.NoExtensions)

	// Install and start target simulator
	simulator := gnmiutils.CreateSimulatorWithName(t, offlineTargetName, false)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Wait for config to connect to the target
	gnmiutils.WaitForTargetAvailable(t, topoapi.ID(simulator.Name()), time.Minute)
	err := gnmiutils.WaitForConfigurationCompleteOrFail(t, configapi.ConfigurationID(simulator.Name()), time.Minute)
	assert.NoError(t, err)

	gnmiutils.CheckGNMIValue(ctx, t, gnmiClient, targetPath, modValue, 0, "Query after set returned the wrong value")

	// Check that the value was set properly on the target, wait for configuration gets completed
	targetGnmiClient := gnmiutils.GetTargetGNMIClientOrFail(ctx, t, simulator)
	gnmiutils.CheckTargetValue(ctx, t, targetGnmiClient, targetPath, modValue)
}

func createOfflineTarget(t *testing.T, targetID topoapi.ID, targetType string, targetVersion string, targetAddress string) {
	topoClient, err := gnmiutils.NewTopoClient()
	assert.NotNil(t, topoClient)
	assert.Nil(t, err)

	newTarget, err := gnmiutils.NewTargetEntity(string(targetID), targetType, targetVersion, targetAddress)
	assert.NoError(t, err)
	err = topoClient.Create(context.Background(), newTarget)
	assert.NoError(t, err)
}
