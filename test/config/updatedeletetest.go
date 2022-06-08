// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	udtestRootPath         = "/interfaces/interface[name=test]"
	udtestNamePath         = udtestRootPath + "/config/name"
	udtestEnabledPath      = udtestRootPath + "/config/enabled"
	udtestDescriptionPath  = udtestRootPath + "/config/description"
	udtestNameValue        = "test"
	udtestDescriptionValue = "description"
)

// TestUpdateDelete tests update and delete paths in a single GNMI request
func (s *TestSuite) TestUpdateDelete(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Get the first configured simulator from the environment.
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(simulator.Name()), 1*time.Minute)
	assert.True(t, ready)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Create interface tree using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: simulator.Name(), Path: udtestNamePath, PathDataValue: udtestNameValue, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(t)

	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(t, udtestNameValue)

	// Set initial values for Enabled and Description using gNMI client
	setInitialValuesPath := []proto.GNMIPath{
		{TargetName: simulator.Name(), Path: udtestEnabledPath, PathDataValue: "true", PathDataType: proto.BoolVal},
		{TargetName: simulator.Name(), Path: udtestDescriptionPath, PathDataValue: udtestDescriptionValue, PathDataType: proto.StringVal},
	}
	setReq.UpdatePaths = setInitialValuesPath
	setReq.SetOrFail(t)

	// Update Enabled, delete Description using gNMI client
	updateEnabledPath := []proto.GNMIPath{
		{TargetName: simulator.Name(), Path: udtestEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	deleteDescriptionPath := []proto.GNMIPath{
		{TargetName: simulator.Name(), Path: udtestDescriptionPath},
	}
	setReq.UpdatePaths = updateEnabledPath
	setReq.DeletePaths = deleteDescriptionPath
	setReq.SetOrFail(t)

	// Check that the Enabled value is set correctly
	getConfigReq.Paths = updateEnabledPath
	getConfigReq.CheckValues(t, "false")

	//  Make sure Description got removed
	getConfigReq.Paths = gnmiutils.GetTargetPath(simulator.Name(), udtestDescriptionPath)
	getConfigReq.CheckValues(t, "")
}
