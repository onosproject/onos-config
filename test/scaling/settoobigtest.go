// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package scaling

import (
	"fmt"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

const (
	newRootName                = "new-root"
	newRootPath                = "/interfaces/interface[name=" + newRootName + "]"
	newRootConfigNamePath      = newRootPath + "/config/name"
	newRootEnabledPath         = newRootPath + "/config/enabled"
	newRootDescriptionPath     = newRootPath + "/config/description"
	newRootMtuPath             = newRootPath + "/config/mtu"
	newRootTypePath            = newRootPath + "/config/type"
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

func (s *TestSuite) testSetTooBig(t *testing.T, encoding gnmiapi.Encoding) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Make a simulated device
	simulator := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, simulator)

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(simulator.Name()), 1*time.Minute)
	assert.True(t, ready)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// First do a single set path by setting the name of new root using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: simulator.Name(), Path: newRootConfigNamePath, PathDataValue: newRootName, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(t)

	getConfigReq := &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: encoding,
	}

	// Check that the name value was set correctly
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(t, newRootName)
	t.Logf("successfully allowed gNMI set with less than %d updates", gnmiSetLimitForTest)

	// Now do a test of multiple paths that will exceed the limit using gNMI client
	setPath := []proto.GNMIPath{
		{TargetName: simulator.Name(), Path: newRootDescriptionPath, PathDataValue: newDescription, PathDataType: proto.StringVal},
		{TargetName: simulator.Name(), Path: newRootEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
		{TargetName: simulator.Name(), Path: newRootMtuPath, PathDataValue: "1000", PathDataType: proto.IntVal},
		{TargetName: simulator.Name(), Path: newRootTypePath, PathDataValue: "ethernet", PathDataType: proto.StringVal},
		{TargetName: simulator.Name(), Path: newRootHoldTimeUpPath, PathDataValue: "30", PathDataType: proto.IntVal},
		{TargetName: simulator.Name(), Path: newRootHoldTimeDownPath, PathDataValue: "31", PathDataType: proto.IntVal},
		{TargetName: simulator.Name(), Path: newRootSubIf1ConfigIdx, PathDataValue: "1", PathDataType: proto.IntVal},
		{TargetName: simulator.Name(), Path: newRootSubIf1ConfigDesc, PathDataValue: "Sub if 1", PathDataType: proto.StringVal},
		{TargetName: simulator.Name(), Path: newRootSubIf1ConfigEnabled, PathDataValue: "true", PathDataType: proto.BoolVal},
		{TargetName: simulator.Name(), Path: newRootSubIf2ConfigIdx, PathDataValue: "2", PathDataType: proto.IntVal},
		{TargetName: simulator.Name(), Path: newRootSubIf2ConfigDesc, PathDataValue: "Sub if 2", PathDataType: proto.StringVal},
		{TargetName: simulator.Name(), Path: newRootSubIf2ConfigEnabled, PathDataValue: "true", PathDataType: proto.BoolVal},
	}
	setReq.UpdatePaths = setPath
	err := setReq.SetExpectFail(t)
	assert.Equal(t, fmt.Sprintf("rpc error: code = InvalidArgument desc = "+
		"number of updates and deletes in a gNMI Set must not exceed %d. Target: %s Updates: %d, Deletes %d",
		gnmiSetLimitForTest, simulator.Name(), 12, 0), err.Error())
	t.Logf("successfully prevented gNMI set with more than %d updates", gnmiSetLimitForTest)
}

// TestSetTooBig tests create/set/delete of a tree of GNMI paths to a single device, where they exceed the size limit
func (s *TestSuite) TestSetTooBig(t *testing.T) {
	t.Run("TestSetTooBig PROTO",
		func(t *testing.T) {
			s.testSetTooBig(t, gnmiapi.Encoding_PROTO)
		})
}
