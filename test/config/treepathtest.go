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
)

const (
	newRootName            = "new-root"
	newRootPath            = "/interfaces/interface[name=" + newRootName + "]"
	newRootConfigNamePath  = newRootPath + "/config/name"
	newRootEnabledPath     = newRootPath + "/config/enabled"
	newRootDescriptionPath = newRootPath + "/config/description"
	newDescription         = "description"
)

func (s *TestSuite) testTreePath(ctx context.Context, encoding gnmiapi.Encoding) {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(ctx, topoapi.ID(s.simulator1))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.NoRetry)

	getPath := gnmiutils.GetTargetPath(s.simulator1, newRootEnabledPath)

	// Set name of new root using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: s.simulator1, Path: newRootConfigNamePath, PathDataValue: newRootName, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(s.T())

	// Set values using gNMI client
	setPath := []proto.GNMIPath{
		{TargetName: s.simulator1, Path: newRootDescriptionPath, PathDataValue: newDescription, PathDataType: proto.StringVal},
		{TargetName: s.simulator1, Path: newRootEnabledPath, PathDataValue: "false", PathDataType: proto.BoolVal},
	}
	setReq.UpdatePaths = setPath
	setReq.SetOrFail(s.T())

	getConfigReq := &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: encoding,
	}

	// Check that the name value was set correctly
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(s.T(), newRootName)

	// Check that the enabled value was set correctly
	getConfigReq.Paths = getPath
	getConfigReq.CheckValues(s.T(), "false")

	// Remove the root path we added
	setReq.UpdatePaths = nil
	setReq.DeletePaths = getPath
	setReq.SetOrFail(s.T())

	//  Make sure child got removed
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(s.T(), newRootName)

	//  Make sure new root got removed
	getConfigReq.Paths = getPath
	getConfigReq.CheckValues(s.T(), "")
}

// TestTreePath tests create/set/delete of a tree of GNMI paths to a single device
func (s *TestSuite) TestTreePath(ctx context.Context) {
	s.Run("TestTreePath PROTO", func() {
		s.testTreePath(ctx, gnmiapi.Encoding_PROTO)
	})
	s.Run("TestTreePath JSON", func() {
		s.testTreePath(ctx, gnmiapi.Encoding_JSON)
	})
}
