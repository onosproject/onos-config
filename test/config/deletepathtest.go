// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	"github.com/onosproject/onos-config/test/utils/proto"
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
func (s *TestSuite) TestDeletePathLeaf() {
	const (
		leafPath  = "/system/config/login-banner"
		leafValue = "123"
	)

	// Wait for config to connect to the target
	s.WaitForTargetAvailable(topo.ID(s.simulator1.Name))

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Set a value so onos-config starts to track the path
	targetPath := gnmiutils.GetTargetPathWithValue(s.simulator1.Name, leafPath, leafValue, proto.StringVal)
	setReq := &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: targetPath,
	}
	setReq.SetOrFail(s.T())

	getReq := &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
		Paths:      gnmiutils.GetTargetPath(s.simulator1.Name, "/"),
	}

	// Make sure that the path is there
	paths, err := getReq.Get()
	s.NoError(err)
	s.True(checkForPath(paths, leafPath))

	// Delete leaf path
	setReq.UpdatePaths = nil
	setReq.DeletePaths = targetPath
	setReq.SetOrFail(s.T())

	// Make sure the path got deleted
	paths, err = getReq.Get()
	s.NoError(err)
	s.False(checkForPath(paths, leafPath))
}

// TestDeleteRoot checks that when a root node is removed, its path is removed
func (s *TestSuite) TestDeleteRoot() {
	const (
		interfaceName   = "testinterface"
		rootPath        = "/interfaces/interface[name=" + interfaceName + "]"
		namePath        = rootPath + "/config/name"
		descriptionPath = rootPath + "/config/description"
	)

	// Wait for config to connect to the target
	s.WaitForTargetAvailable(topo.ID(s.simulator1.Name))

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	// Create new interface tree using gNMI client
	setNamePath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: namePath, PathDataValue: interfaceName, PathDataType: proto.StringVal},
	}
	setReq := &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setNamePath,
	}
	setReq.SetOrFail(s.T())

	// Set the description field of the new interface
	setDescriptionPath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: descriptionPath, PathDataValue: "123", PathDataType: proto.StringVal},
	}
	setReq = &gnmiutils.SetRequest{
		Ctx:         s.Context(),
		Client:      gnmiClient,
		Extensions:  s.SyncExtension(),
		Encoding:    gnmiapi.Encoding_PROTO,
		UpdatePaths: setDescriptionPath,
	}
	setReq.SetOrFail(s.T())

	// Make sure that the new paths are there by reading all the paths from the top
	getReq := &gnmiutils.GetRequest{
		Ctx:        s.Context(),
		Client:     gnmiClient,
		Extensions: s.SyncExtension(),
		Encoding:   gnmiapi.Encoding_PROTO,
		Paths:      gnmiutils.GetTargetPath(s.simulator1.Name, "/"),
	}
	paths, err := getReq.Get()
	s.NoError(err)
	s.True(checkForPath(paths, descriptionPath))
	s.True(checkForPath(paths, namePath))

	// Now delete the interface
	rootTargetPath := []proto.GNMIPath{
		{TargetName: s.simulator1.Name, Path: rootPath, PathDataValue: interfaceName, PathDataType: proto.StringVal},
	}
	setReq.UpdatePaths = nil
	setReq.DeletePaths = rootTargetPath
	setReq.SetOrFail(s.T())

	// Make sure that the paths are gone
	paths, err = getReq.Get()
	s.NoError(err)
	s.False(checkForPath(paths, descriptionPath))
	s.False(checkForPath(paths, namePath))
}
