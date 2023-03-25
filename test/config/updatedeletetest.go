// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
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
func (s *TestSuite) TestUpdateDelete(ctx context.Context) {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(ctx, topoapi.ID(s.simulator1), 1*time.Minute)
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.NoRetry)

	// Create interface tree using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: s.simulator1, Path: udtestNamePath, PathDataValue: udtestNameValue, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(s.T())

	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(s.T(), udtestNameValue)

	// Set initial values for Enabled and Description using gNMI client
	setInitialValuesPath := []proto.GNMIPath{
		{TargetName: s.simulator1, Path: udtestEnabledPath, PathDataValue: "true", PathDataType: proto.BoolVal},
		{TargetName: s.simulator1, Path: udtestDescriptionPath, PathDataValue: udtestDescriptionValue, PathDataType: proto.StringVal},
	}
	setReq.UpdatePaths = setInitialValuesPath
	setReq.SetOrFail(s.T())

	// Update Enabled, delete Description using gNMI client
	updateEnabledPath := []proto.GNMIPath{
		{TargetName: s.simulator1, Path: udtestEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	deleteDescriptionPath := []proto.GNMIPath{
		{TargetName: s.simulator1, Path: udtestDescriptionPath},
	}
	setReq.UpdatePaths = updateEnabledPath
	setReq.DeletePaths = deleteDescriptionPath
	setReq.SetOrFail(s.T())

	// Check that the Enabled value is set correctly
	getConfigReq.Paths = updateEnabledPath
	getConfigReq.CheckValues(s.T(), "false")

	//  Make sure Description got removed
	getConfigReq.Paths = gnmiutils.GetTargetPath(s.simulator1, udtestDescriptionPath)
	getConfigReq.CheckValues(s.T(), "")
}
