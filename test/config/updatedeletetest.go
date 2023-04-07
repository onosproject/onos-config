// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
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
func (s *TestSuite) TestUpdateDelete() {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator1.Name))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Create interface tree using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: udtestNamePath, PathDataValue: udtestNameValue, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(s.T())

	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(s.T(), udtestNameValue)

	// Set initial values for Enabled and Description using gNMI client
	setInitialValuesPath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: udtestEnabledPath, PathDataValue: "true", PathDataType: proto.BoolVal},
		{TargetName: s.simulator1.Name, Path: udtestDescriptionPath, PathDataValue: udtestDescriptionValue, PathDataType: proto.StringVal},
	}
	setReq.UpdatePaths = setInitialValuesPath
	setReq.SetOrFail(s.T())

	// Update Enabled, delete Description using gNMI client
	updateEnabledPath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: udtestEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	deleteDescriptionPath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: udtestDescriptionPath},
	}
	setReq.UpdatePaths = updateEnabledPath
	setReq.DeletePaths = deleteDescriptionPath
	setReq.SetOrFail(s.T())

	// Check that the Enabled value is set correctly
	getConfigReq.Paths = updateEnabledPath
	getConfigReq.CheckValues(s.T(), "false")

	//  Make sure Description got removed
	getConfigReq.Paths = gnmiutils.GetTargetPath(s.simulator1.Name, udtestDescriptionPath)
	getConfigReq.CheckValues(s.T(), "")
}
