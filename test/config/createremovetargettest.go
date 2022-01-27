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

	"github.com/onosproject/helmit/pkg/helm"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/stretchr/testify/assert"
)

const (
	createRemoveTargetModPath          = "/system/clock/config/timezone-name"
	createRemoveTargetModValue1        = "Europe/Paris"
	createRemoveTargetModValue2        = "Europe/London"
	createRemoveTargetModTargetName    = "offline-sim-crd"
	createRemoveTargetModTargetVersion = "1.0.0"
	createRemoveTargetModTargetType    = "devicesim-1.0.x"
)

// TestCreatedRemovedTarget tests set/query of a single GNMI path to a single target that is created, removed, then created again
func (s *TestSuite) TestCreatedRemovedTarget(t *testing.T) {
	t.Skip()
	topoClient, err := gnmi.NewTopoClient()
	assert.NotNil(t, topoClient)
	assert.Nil(t, err)

	newTarget := &topo.Object{
		ID:   createRemoveTargetModTargetName,
		Type: topo.Object_ENTITY,
		Obj: &topo.Object_Entity{
			Entity: &topo.Entity{
				KindID: createRemoveTargetModTargetType,
			},
		},
	}

	_ = newTarget.SetAspect(&topo.Configurable{
		Type:    createRemoveTargetModTargetType,
		Address: createRemoveTargetModTargetName + ":11161",
		Version: createRemoveTargetModTargetVersion,
		Timeout: uint64((10 * time.Second).Milliseconds()),
	})

	_ = newTarget.SetAspect(&topo.TLSOptions{Plain: true})

	err = topoClient.Create(context.Background(), newTarget)
	assert.NoError(t, err)

	//  Start a new simulated target
	simulator := helm.
		Chart("device-simulator").
		Release(offlineTargetName)
	err = simulator.Install(true)
	assert.NoError(t, err)

	// Wait for config to connect to the target
	gnmi.WaitForTargetAvailable(t, createRemoveTargetModTargetName, 1*time.Minute)

	// Make a GNMI client to use for requests
	c := gnmi.GetGNMIClientOrFail(t)

	targetPath := gnmi.GetTargetPathWithValue(createRemoveTargetModTargetName, createRemoveTargetModPath, createRemoveTargetModValue1, proto.StringVal)

	// Set a value using gNMI client - target is up
	gnmi.SetGNMIValueOrFail(t, c, targetPath, gnmi.NoPaths, gnmi.NoExtensions)

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, c, targetPath, createRemoveTargetModValue1, 0, "Query after set returned the wrong value")

	// Wait for config to reconnect to the target
	gnmi.WaitForTargetAvailable(t, createRemoveTargetModTargetName, 1*time.Minute)

	// interrogate the target to check that the value was set properly
	targetGnmiClient := gnmi.GetTargetGNMIClientOrFail(t, simulator)
	gnmi.CheckTargetValue(t, targetGnmiClient, targetPath, createRemoveTargetModValue1)

	//  Shut down the simulator
	gnmi.DeleteSimulator(t, simulator)
	gnmi.WaitForTargetUnavailable(t, createRemoveTargetModTargetName, 1*time.Minute)

	// Set a value using gNMI client - target is down
	setPath2 := gnmi.GetTargetPathWithValue(createRemoveTargetModTargetName, createRemoveTargetModPath, createRemoveTargetModValue2, proto.StringVal)
	gnmi.SetGNMIValueOrFail(t, c, setPath2, gnmi.NoPaths, gnmi.NoExtensions)

	//  Restart simulated target
	err = simulator.Install(true)
	assert.NoError(t, err)

	// Wait for config to connect to the target
	gnmi.WaitForTargetAvailable(t, createRemoveTargetModTargetName, 2*time.Minute)

	// Check that the value was set correctly
	gnmi.CheckGNMIValue(t, c, targetPath, createRemoveTargetModValue2, 0, "Query after set 2 returns wrong value")

	// interrogate the target to check that the value was set properly
	targetGnmiClient2 := gnmi.GetTargetGNMIClientOrFail(t, simulator)
	gnmi.CheckTargetValue(t, targetGnmiClient2, targetPath, createRemoveTargetModValue2)

	// Clean up the simulator
	gnmi.DeleteSimulator(t, simulator)
}
