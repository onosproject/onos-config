// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestCascadingDelete checks that when a root leaf or an intermediate leaf is removed, all its children are removed too
func (s *TestSuite) TestCascadingDelete(t *testing.T) {
	const (
		switchName      = "testswitch"
		rootPath        = "/switches/switch[name=" + switchName + "]"
		namePath        = rootPath + "/config/name"
		descriptionPath = rootPath + "/config/description"
		port1Name       = "testswitchPort1"
		port1Path       = rootPath + "/ports/port1[name=" + port1Name + "]"
		port2Name       = "testswitchPort2"
		port2Path       = rootPath + "/ports/port2[name=" + port2Name + "]"
	)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create a simulated target
	target := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target)

	// Wait for config to connect to the target
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Create new switch tree using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: target.Name(), Path: namePath, PathDataValue: switchName, PathDataType: proto.StringVal},
	}
	setReq := &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(t)

	// Set the description field of the new switch
	setDescriptionPath := []proto.GNMIPath{
		{TargetName: target.Name(), Path: descriptionPath, PathDataValue: "123", PathDataType: proto.StringVal},
	}
	setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setDescriptionPath,
	}
	setReq.SetOrFail(t)

	// Set the port1 field of the new switch
	setPort1Path := []proto.GNMIPath{
		{TargetName: target.Name(), Path: port1Path, PathDataValue: port1Name, PathDataType: proto.StringVal},
	}
	setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setPort1Path,
	}
	setReq.SetOrFail(t)

	// Set the port2 field of the new switch
	setPort2Path := []proto.GNMIPath{
		{TargetName: target.Name(), Path: port2Path, PathDataValue: port2Name, PathDataType: proto.StringVal},
	}
	setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setPort2Path,
	}
	setReq.SetOrFail(t)

	// Make sure that the new paths are there by reading all the paths from the top
	getReq := &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Extensions: gnmiutils.SyncExtension(t),
		Encoding:   gnmiapi.Encoding_PROTO,
		Paths:      gnmiutils.GetTargetPath(target.Name(), "/"),
	}
	paths, err := getReq.Get()
	assert.NoError(t, err)
	assert.True(t, checkForPath(paths, descriptionPath))
	assert.True(t, checkForPath(paths, namePath))
	assert.True(t, checkForPath(paths, port1Path))
	assert.True(t, checkForPath(paths, port2Path))

	// Now delete the switch
	rootTargetPath := []proto.GNMIPath{
		{TargetName: target.Name(), Path: rootPath, PathDataValue: switchName, PathDataType: proto.StringVal},
	}
	setReq.UpdatePaths = nil
	setReq.DeletePaths = rootTargetPath
	setReq.SetOrFail(t)

	// Make sure that the paths are gone
	paths, err = getReq.Get()
	assert.NoError(t, err)
	assert.False(t, checkForPath(paths, descriptionPath))
	assert.False(t, checkForPath(paths, namePath))
	assert.False(t, checkForPath(paths, port1Path))
	assert.False(t, checkForPath(paths, port2Path))
}
