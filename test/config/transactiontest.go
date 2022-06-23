// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/stretchr/testify/assert"
)

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) testTransaction(t *testing.T, encoding gnmiapi.Encoding) {
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

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create the first target simulator
	target1 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target1)

	// Wait for config to connect to the first simulator
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target1.Name()), time.Minute)

	// Create the second target simulator
	target2 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target2)

	// Wait for config to connect to the second simulator
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target2.Name()), time.Minute)

	// Set up paths for the two targets
	targets := []string{target1.Name(), target2.Name()}
	targetPathsForGet := gnmiutils.GetTargetPaths(targets, paths)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Set initial values
	targetPathsForInit := gnmiutils.GetTargetPathsWithValues(targets, paths, initialValues)

	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Encoding:    gnmiapi.Encoding_PROTO,
		Extensions:  gnmiutils.SyncExtension(t),
		UpdatePaths: targetPathsForInit,
	}
	setReq.SetOrFail(t)

	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Encoding:   encoding,
		Extensions: gnmiutils.SyncExtension(t),
	}
	targetPath1 := gnmiutils.GetTargetPath(target1.Name(), path1)
	targetPath2 := gnmiutils.GetTargetPath(target2.Name(), path2)

	getReq.Paths = targetPath1
	getReq.CheckValues(t, initValue1)
	getReq.Paths = targetPath2
	getReq.CheckValues(t, initValue2)

	// Create a change that can be rolled back
	targetPathsForSet := gnmiutils.GetTargetPathsWithValues(targets, paths, values)
	setReq.UpdatePaths = targetPathsForSet
	_, transactionIndex := setReq.SetOrFail(t)

	// Check that the values were set correctly
	getReq.Paths = targetPath1
	getReq.CheckValues(t, value1)
	getReq.Paths = targetPath2
	getReq.CheckValues(t, value2)

	// Check that the values are set on the targets
	target1GnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target1)
	target2GnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target2)

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
	target1GetReq.CheckValues(t, value1)
	target1GetReq.Paths = targetPathsForGet[1:2]
	target1GetReq.CheckValues(t, value2)
	target2GetReq.Paths = targetPathsForGet[2:3]
	target2GetReq.CheckValues(t, value1)
	target2GetReq.Paths = targetPathsForGet[3:4]
	target2GetReq.CheckValues(t, value2)

	// Now rollback the change
	adminClient, err := gnmiutils.NewAdminServiceClient(ctx)
	assert.NoError(t, err)
	rollbackResponse, rollbackError := adminClient.RollbackTransaction(
		context.Background(), &admin.RollbackRequest{Index: transactionIndex})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")

	// Check that the values were really rolled back in onos-config
	getReq.Paths = targetPath1
	getReq.CheckValues(t, initValue1)
	getReq.Paths = targetPath2
	getReq.CheckValues(t, initValue2)

	// Check that the values were rolled back on the targets
	target1GetReq.Paths = targetPathsForGet[0:1]
	target1GetReq.CheckValues(t, initValue1)
	target1GetReq.Paths = targetPathsForGet[1:2]
	target1GetReq.CheckValues(t, initValue2)
	target2GetReq.Paths = targetPathsForGet[2:3]
	target2GetReq.CheckValues(t, initValue1)
	target2GetReq.Paths = targetPathsForGet[3:4]
	target2GetReq.CheckValues(t, initValue2)
}

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) TestTransaction(t *testing.T) {
	t.Run("TestTransaction PROTO",
		func(t *testing.T) {
			s.testTransaction(t, gnmiapi.Encoding_PROTO)
		})
	t.Run("TestTransaction JSON",
		func(t *testing.T) {
			s.testTransaction(t, gnmiapi.Encoding_JSON)
		})
}
