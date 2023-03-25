// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"github.com/onosproject/onos-config/test"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
)

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) testTransaction(ctx context.Context, encoding gnmiapi.Encoding) {
	const (
		value1     = "test-motd-banner"
		path1      = "/system/config/motd-banner"
		value2     = "test-login-banner"
		path2      = "/system/config/login-banner"
		initValue1 = "1"
		initValue2 = "2"
	)

	var (
		paths         = []string{path1, path2}
		values        = []string{value1, value2}
		initialValues = []string{initValue1, initValue2}
	)

	// Wait for config to connect to the first simulator
	s.WaitForTargetAvailable(ctx, topo.ID(s.simulator1), time.Minute)

	// Wait for config to connect to the second simulator
	s.WaitForTargetAvailable(ctx, topo.ID(s.simulator2), time.Minute)

	// Set up paths for the two targets
	targets := []string{s.simulator1, s.simulator2}
	targetPathsForGet := gnmiutils.GetTargetPaths(targets, paths)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(ctx, test.NoRetry)

	// Set initial values
	targetPathsForInit := gnmiutils.GetTargetPathsWithValues(targets, paths, initialValues)

	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		Extensions:  s.SyncExtension(),
		UpdatePaths: targetPathsForInit,
	}
	setReq.SetOrFail(s.T())

	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Encoding:   encoding,
		Extensions: s.SyncExtension(),
	}
	targetPath1 := gnmiutils.GetTargetPath(s.simulator1, path1)
	targetPath2 := gnmiutils.GetTargetPath(s.simulator2, path2)

	getReq.Paths = targetPath1
	getReq.CheckValues(s.T(), initValue1)
	getReq.Paths = targetPath2
	getReq.CheckValues(s.T(), initValue2)

	// Create a change that can be rolled back
	targetPathsForSet := gnmiutils.GetTargetPathsWithValues(targets, paths, values)
	setReq.UpdatePaths = targetPathsForSet
	_, transactionIndex := setReq.SetOrFail(s.T())

	// Check that the values were set correctly
	getReq.Paths = targetPath1
	getReq.CheckValues(s.T(), value1)
	getReq.Paths = targetPath2
	getReq.CheckValues(s.T(), value2)

	// Check that the values are set on the targets
	target1GnmiClient := s.NewSimulatorGNMIClientOrFail(ctx, s.simulator1)
	target2GnmiClient := s.NewSimulatorGNMIClientOrFail(ctx, s.simulator2)

	var target1GetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   target1GnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
	}
	var target2GetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   target2GnmiClient,
		Encoding: gnmiapi.Encoding_JSON,
	}
	target1GetReq.Paths = targetPathsForGet[0:1]
	target1GetReq.CheckValues(s.T(), value1)
	target1GetReq.Paths = targetPathsForGet[1:2]
	target1GetReq.CheckValues(s.T(), value2)
	target2GetReq.Paths = targetPathsForGet[2:3]
	target2GetReq.CheckValues(s.T(), value1)
	target2GetReq.Paths = targetPathsForGet[3:4]
	target2GetReq.CheckValues(s.T(), value2)

	// Now rollback the change
	adminClient, err := s.NewAdminServiceClient(ctx)
	s.NoError(err)
	rollbackResponse, rollbackError := adminClient.RollbackTransaction(
		context.Background(), &admin.RollbackRequest{Index: transactionIndex})

	s.NoError(rollbackError, "Rollback returned an error")
	s.NotNil(rollbackResponse, "Response for rollback is nil")

	// Check that the values were really rolled back in onos-config
	getReq.Paths = targetPath1
	getReq.CheckValues(s.T(), initValue1)
	getReq.Paths = targetPath2
	getReq.CheckValues(s.T(), initValue2)

	// Check that the values were rolled back on the targets
	target1GetReq.Paths = targetPathsForGet[0:1]
	target1GetReq.CheckValues(s.T(), initValue1)
	target1GetReq.Paths = targetPathsForGet[1:2]
	target1GetReq.CheckValues(s.T(), initValue2)
	target2GetReq.Paths = targetPathsForGet[2:3]
	target2GetReq.CheckValues(s.T(), initValue1)
	target2GetReq.Paths = targetPathsForGet[3:4]
	target2GetReq.CheckValues(s.T(), initValue2)
}

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) TestTransaction(ctx context.Context) {
	s.Run("TestTransaction PROTO", func() {
		s.testTransaction(ctx, gnmiapi.Encoding_PROTO)
	})
	s.Run("TestTransaction JSON", func() {
		s.testTransaction(ctx, gnmiapi.Encoding_JSON)
	})
}
