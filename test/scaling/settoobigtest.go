// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package scaling

import (
	"fmt"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

const (
	newRootName                = "new-root"
	newRootPath                = "/interfaces/interface[name=" + newRootName + "]"
	newRootConfigNamePath      = newRootPath + "/config/name"
	newRootEnabledPath         = newRootPath + "/config/enabled"
	newRootDescriptionPath     = newRootPath + "/config/description"
	newRootMtuPath             = newRootPath + "/config/mtu"
	newRootHoldTimeUpPath      = newRootPath + "/hold-time/config/up"
	newRootHoldTimeDownPath    = newRootPath + "/hold-time/config/down"
	subInterface1              = "/subinterfaces/subinterface[index=1]"
	newRootSubIf1ConfigIdx     = newRootPath + subInterface1 + "/config/index"
	newRootSubIf1ConfigDesc    = newRootPath + subInterface1 + "/config/description"
	newRootSubIf1ConfigEnabled = newRootPath + subInterface1 + "/config/enabled"
	subInterface2              = "/subinterfaces/subinterface[index=2]"
	newRootSubIf2ConfigIdx     = newRootPath + subInterface2 + "/config/index"
	newRootSubIf2ConfigDesc    = newRootPath + subInterface2 + "/config/description"
	newRootSubIf2ConfigEnabled = newRootPath + subInterface2 + "/config/enabled"
	newDescription             = "description"
)

func (s *TestSuite) testSetTooBig(encoding gnmiapi.Encoding) {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator.Name))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// First do a single set path by setting the name of new root using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: s.simulator.Name, Path: newRootConfigNamePath, PathDataValue: newRootName, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(s.T())

	getConfigReq := &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Encoding: encoding,
	}

	// Check that the name value was set correctly
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(s.T(), newRootName)
	s.T().Logf("successfully allowed gNMI set with less than %d updates", gnmiSetLimitForTest)

	// Now do a test of multiple paths that will exceed the limit using gNMI client
	setPath := []proto.GNMIPath{
		{TargetName: s.simulator.Name, Path: newRootDescriptionPath, PathDataValue: newDescription, PathDataType: proto.StringVal},
		{TargetName: s.simulator.Name, Path: newRootEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
		{TargetName: s.simulator.Name, Path: newRootMtuPath, PathDataValue: "1000", PathDataType: proto.IntVal},
		{TargetName: s.simulator.Name, Path: newRootHoldTimeUpPath, PathDataValue: "30", PathDataType: proto.IntVal},
		{TargetName: s.simulator.Name, Path: newRootHoldTimeDownPath, PathDataValue: "31", PathDataType: proto.IntVal},
		{TargetName: s.simulator.Name, Path: newRootSubIf1ConfigIdx, PathDataValue: "1", PathDataType: proto.IntVal},
		{TargetName: s.simulator.Name, Path: newRootSubIf1ConfigDesc, PathDataValue: "Sub if 1", PathDataType: proto.StringVal},
		{TargetName: s.simulator.Name, Path: newRootSubIf1ConfigEnabled, PathDataValue: "true", PathDataType: proto.BoolVal},
		{TargetName: s.simulator.Name, Path: newRootSubIf2ConfigIdx, PathDataValue: "2", PathDataType: proto.IntVal},
		{TargetName: s.simulator.Name, Path: newRootSubIf2ConfigDesc, PathDataValue: "Sub if 2", PathDataType: proto.StringVal},
		{TargetName: s.simulator.Name, Path: newRootSubIf2ConfigEnabled, PathDataValue: "true", PathDataType: proto.BoolVal},
	}
	setReq.UpdatePaths = setPath
	err := setReq.SetExpectFail(s.T())
	s.Equal(fmt.Sprintf("rpc error: code = InvalidArgument desc = "+
		"number of updates and deletes in a gNMI Set must not exceed %d. Target: %s Updates: %d, Deletes %d",
		gnmiSetLimitForTest, s.simulator.Name, 11, 0), err.Error())
	s.T().Logf("successfully prevented gNMI set with more than %d updates", gnmiSetLimitForTest)
}

// TestSetTooBig tests create/set/delete of a tree of GNMI paths to a single device, where they exceed the size limit
func (s *TestSuite) TestSetTooBig() {
	s.Run("TestSetTooBig PROTO", func() {
		s.testSetTooBig(gnmiapi.Encoding_PROTO)
	})
}
