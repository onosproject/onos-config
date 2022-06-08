// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
)

func checkForPath(paths []proto.GNMIPath, wantedPath string) bool {
	for _, p := range paths {
		if p.Path == wantedPath {
			return true
		}
	}
	return false
}

// TestDeletePathLeaf checks that when a leaf node is removed, its path is removed
func (s *TestSuite) TestDeletePathLeaf(t *testing.T) {
	const (
		leafPath  = "/system/config/login-banner"
		leafValue = "123"
	)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create simulated target
	target := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target)

	// Wait for config to connect to the target
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Set a value so onos-config starts to track the path
	targetPath := gnmiutils.GetTargetPathWithValue(target.Name(), leafPath, leafValue, proto.StringVal)
	setReq := &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: targetPath,
	}
	setReq.SetOrFail(t)

	getReq := &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Extensions: gnmiutils.SyncExtension(t),
		Encoding:   gnmiapi.Encoding_PROTO,
		Paths:      gnmiutils.GetTargetPath(target.Name(), "/"),
	}

	// Make sure that the path is there
	paths, err := getReq.Get()
	assert.NoError(t, err)
	assert.True(t, checkForPath(paths, leafPath))

	// Delete leaf path
	setReq.UpdatePaths = nil
	setReq.DeletePaths = targetPath
	setReq.SetOrFail(t)

	// Make sure the path got deleted
	paths, err = getReq.Get()
	assert.NoError(t, err)
	assert.False(t, checkForPath(paths, leafPath))
}

// TestDeleteRoot checks that when a root node is removed, its path is removed
func (s *TestSuite) TestDeleteRoot(t *testing.T) {
	const (
		interfaceName   = "testinterface"
		rootPath        = "/interfaces/interface[name=" + interfaceName + "]"
		namePath        = rootPath + "/config/name"
		descriptionPath = rootPath + "/config/description"
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

	// Create new interface tree using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: target.Name(), Path: namePath, PathDataValue: interfaceName, PathDataType: proto.StringVal},
	}
	setReq := &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(t)

	// Set the description field of the new interface
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

	// Now delete the interface
	rootTargetPath := []proto.GNMIPath{
		{TargetName: target.Name(), Path: rootPath, PathDataValue: interfaceName, PathDataType: proto.StringVal},
	}
	setReq.UpdatePaths = nil
	setReq.DeletePaths = rootTargetPath
	setReq.SetOrFail(t)

	// Make sure that the paths are gone
	paths, err = getReq.Get()
	assert.NoError(t, err)
	assert.False(t, checkForPath(paths, descriptionPath))
	assert.False(t, checkForPath(paths, namePath))
}
