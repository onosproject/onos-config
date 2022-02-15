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
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"
	gbp "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/stretchr/testify/assert"
)

// TestDeleteAndRollback tests target deletion and rollback
func (s *TestSuite) TestDeleteAndRollback(t *testing.T) {
	const (
		newValue = "new-value"
		newPath  = "/system/config/login-banner"
	)

	var (
		newPaths  = []string{newPath}
		newValues = []string{newValue}
	)

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Get the configured targets from the environment.
	target1 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target1)
	targets := make([]string, 1)
	targets[0] = target1.Name()

	// Wait for config to connect to the target
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target1.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Set values
	var targetPathsForSet = gnmiutils.GetTargetPathsWithValues(targets, newPaths, newValues)
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gbp.Encoding_PROTO,
		UpdatePaths: targetPathsForSet,
	}
	_, transactionIndex := setReq.SetOrFail(t)

	targetPathsForGet := gnmiutils.GetTargetPaths(targets, newPaths)

	// Check that the values were set correctly
	expectedValues := []string{newValue}
	gnmiutils.CheckGNMIValues(ctx, t, gnmiClient, targetPathsForGet, gnmiutils.NoExtensions, expectedValues, 0, "Query after set returned the wrong value")

	// Check that the values are set on the targets
	target1GnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target1)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   target1GnmiClient,
		Encoding: gbp.Encoding_JSON,
		Paths:    targetPathsForGet[0:1],
	}
	getTargetReq.CheckValue(t, newValue, 0, "value not set on target")

	// Now rollback the change
	adminClient, err := gnmiutils.NewAdminServiceClient(ctx)
	assert.NoError(t, err)
	rollbackResponse, rollbackError := adminClient.RollbackTransaction(context.Background(), &admin.RollbackRequest{Index: transactionIndex})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")

	// Check that the value was really rolled back- should be an error here since the node was deleted
	var onosConfigGetReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     target1GnmiClient,
		Paths:      targetPathsForGet,
		Encoding:   gbp.Encoding_PROTO,
		DataType:   gbp.GetRequest_CONFIG,
		Extensions: gnmiutils.SyncExtension(t),
	}
	_, _, err = onosConfigGetReq.Get()
	assert.Error(t, err)
}
