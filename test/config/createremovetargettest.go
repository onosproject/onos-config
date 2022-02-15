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
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"testing"
	"time"

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
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	simulator := gnmiutils.CreateSimulatorWithName(ctx, t, createRemoveTargetModTargetName, true)
	assert.NotNil(t, simulator)

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, createRemoveTargetModTargetName, 1*time.Minute)
	assert.True(t, ready)

	targetPath := gnmiutils.GetTargetPathWithValue(createRemoveTargetModTargetName, createRemoveTargetModPath, createRemoveTargetModValue1, proto.StringVal)

	// Set a value using gNMI client - target is up
	c := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      c,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: targetPath,
	}
	setReq.SetOrFail(t)

	// Check that the value was set correctly
	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     c,
		Paths:      targetPath,
		Extensions: gnmiutils.SyncExtension(t),
		Encoding:   gnmiapi.Encoding_PROTO,
	}
	getReq.CheckValue(t, createRemoveTargetModValue1)

	// interrogate the target to check that the value was set properly
	targetGnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetGnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValue(t, createRemoveTargetModValue1)

	//  Shut down the simulator
	gnmiutils.DeleteSimulator(t, simulator)
	unavailable := gnmiutils.WaitForTargetUnavailable(ctx, t, createRemoveTargetModTargetName, 2*time.Minute)
	assert.True(t, unavailable)

	// Set a value using gNMI client - target is down
	setPath2 := gnmiutils.GetTargetPathWithValue(createRemoveTargetModTargetName, createRemoveTargetModPath, createRemoveTargetModValue2, proto.StringVal)

	setReq.UpdatePaths = setPath2
	setReq.Extensions = nil
	setReq.SetOrFail(t)

	//  Restart simulated target
	simulator = gnmiutils.CreateSimulatorWithName(ctx, t, createRemoveTargetModTargetName, false)
	assert.NotNil(t, simulator)

	// Wait for config to connect to the target
	ready = gnmiutils.WaitForTargetAvailable(ctx, t, createRemoveTargetModTargetName, 2*time.Minute)
	assert.True(t, ready)
	// Check that the value was set correctly
	getReq.CheckValue(t, createRemoveTargetModValue2)

	// interrogate the target to check that the value was set properly
	targetGnmiClient2 := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, simulator)
	getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   targetGnmiClient2,
		Encoding: gnmiapi.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValue(t, createRemoveTargetModValue2)

	gnmiutils.DeleteSimulator(t, simulator)
}
