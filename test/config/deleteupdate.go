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
	dutestRootPath  = "/interfaces/interface[name=foo]"
	dutestNamePath  = dutestRootPath + "/config/name"
	dutestNameValue = "foo"
)

// TestDeleteUpdate tests update of a path after a previous deletion of a parent path
func (s *TestSuite) TestDeleteUpdate(t *testing.T) {
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
		{TargetName: simulator.Name(), Path: dutestNamePath, PathDataValue: dutestNameValue, PathDataType: proto.StringVal},
	}
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(t)

	// Check the name is there...
	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gnmiapi.Encoding_PROTO,
	}
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(t, dutestNameValue)

	deleteAllPath := []proto.GNMIPath{
		{TargetName: simulator.Name(), Path: dutestRootPath},
	}
	setReq.UpdatePaths = nil
	setReq.DeletePaths = deleteAllPath
	setReq.SetOrFail(t)

	//  Make sure everything got removed
	getConfigReq.Paths = gnmiutils.GetTargetPath(simulator.Name(), dutestRootPath)
	getConfigReq.CheckValues(t, "")

	// Now recreate the same interface tree....
	setReq.UpdatePaths = setNamePath
	setReq.DeletePaths = nil
	setReq.SetOrFail(t)

	// And check it's there...
	getConfigReq.Paths = setNamePath
	getConfigReq.CheckValues(t, dutestNameValue)
}
