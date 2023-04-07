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
	dutestRootPath  = "/interfaces/interface[name=foo]"
	dutestNamePath  = dutestRootPath + "/config/name"
	dutestNameValue = "foo"
)

// TestDeleteUpdate tests update of a path after a previous deletion of a parent path
func (s *TestSuite) TestDeleteUpdate() {
	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator1.Name))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Create interface tree using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: dutestNamePath, PathDataValue: dutestNameValue, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(s.T())

	// Check the name is there...
	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      s.Context(),
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(s.T(), dutestNameValue)

	deleteAllPath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: dutestRootPath},
	}
	setReq.UpdatePaths = nil
	setReq.DeletePaths = deleteAllPath
	setReq.SetOrFail(s.T())

	//  Make sure everything got removed
	getConfigReq.Paths = gnmiutils.GetTargetPath(s.simulator1.Name, dutestRootPath)
	getConfigReq.CheckValues(s.T(), "")

	// Now recreate the same interface tree....
	setReq.UpdatePaths = setNamePath
	setReq.DeletePaths = nil
	setReq.SetOrFail(s.T())

	// And check it's there...
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(s.T(), dutestNameValue)
}
