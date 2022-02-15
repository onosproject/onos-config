// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	gbp "github.com/openconfig/gnmi/proto/gnmi"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/stretchr/testify/assert"
)

// TestTransaction tests setting multiple paths in a single request and rolling it back
func (s *TestSuite) TestTransaction(t *testing.T) {
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
		Encoding:    gbp.Encoding_PROTO,
		Extensions:  gnmiutils.SyncExtension(t),
		UpdatePaths: targetPathsForInit,
	}
	setReq.SetOrFail(t)

	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Encoding:   gbp.Encoding_PROTO,
		Extensions: gnmiutils.SyncExtension(t),
	}
	targetPath1 := gnmiutils.GetTargetPath(target1.Name(), path1)
	targetPath2 := gnmiutils.GetTargetPath(target2.Name(), path2)

	getReq.Paths = targetPath1
	getReq.CheckValue(t, initValue1, 0, "value1 incorrect")
	getReq.Paths = targetPath2
	getReq.CheckValue(t, initValue2, 0, "value2 incorrect")

	// Create a change that can be rolled back
	targetPathsForSet := gnmiutils.GetTargetPathsWithValues(targets, paths, values)
	setReq.UpdatePaths = targetPathsForSet
	_, transactionIndex := setReq.SetOrFail(t)

	// Check that the values were set correctly
	getReq.Paths = targetPath1
	getReq.CheckValue(t, value1, 0, "value1 incorrect")
	getReq.Paths = targetPath2
	getReq.CheckValue(t, value2, 0, "value2 incorrect")

	// Check that the values are set on the targets
	target1GnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target1)
	target2GnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target2)

	var target1GetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   target1GnmiClient,
		Encoding: gbp.Encoding_JSON,
	}
	var target2GetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   target2GnmiClient,
		Encoding: gbp.Encoding_JSON,
	}
	target1GetReq.Paths = targetPathsForGet[0:1]
	target1GetReq.CheckValue(t, value1, 0, "value is wrong on target")
	target1GetReq.Paths = targetPathsForGet[1:2]
	target1GetReq.CheckValue(t, value2, 0, "value is wrong on target")
	target2GetReq.Paths = targetPathsForGet[2:3]
	target2GetReq.CheckValue(t, value1, 0, "value is wrong on target")
	target2GetReq.Paths = targetPathsForGet[3:4]
	target2GetReq.CheckValue(t, value2, 0, "value is wrong on target")

	// Now rollback the change
	adminClient, err := gnmiutils.NewAdminServiceClient(ctx)
	assert.NoError(t, err)
	rollbackResponse, rollbackError := adminClient.RollbackTransaction(
		context.Background(), &admin.RollbackRequest{Index: transactionIndex})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")

	// Check that the values were really rolled back in onos-config
	getReq.Paths = targetPath1
	getReq.CheckValue(t, initValue1, 0, "value1 incorrect")
	getReq.Paths = targetPath2
	getReq.CheckValue(t, initValue2, 0, "value2 incorrect")

	// Check that the values were rolled back on the targets
	target1GetReq.Paths = targetPathsForGet[0:1]
	target1GetReq.CheckValue(t, initValue1, 0, "value is wrong on target")
	target1GetReq.Paths = targetPathsForGet[1:2]
	target1GetReq.CheckValue(t, initValue2, 0, "value is wrong on target")
	target2GetReq.Paths = targetPathsForGet[2:3]
	target2GetReq.CheckValue(t, initValue1, 0, "value is wrong on target")
	target2GetReq.Paths = targetPathsForGet[3:4]
	target2GetReq.CheckValue(t, initValue2, 0, "value is wrong on target")
}
