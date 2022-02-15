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
	"github.com/onosproject/onos-config/test/utils/proto"
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

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Get the configured target from the environment.
	target1 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target1)

	// Wait for config to connect to the target
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target1.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Set values
	var targetPath = gnmiutils.GetTargetPathWithValue(target1.Name(), newPath, newValue, proto.StringVal)
	var setReq = &gnmiutils.SetRequest{
		Ctx:         ctx,
		Client:      gnmiClient,
		Extensions:  gnmiutils.SyncExtension(t),
		Encoding:    gbp.Encoding_PROTO,
		UpdatePaths: targetPath,
	}
	_, transactionIndex := setReq.SetOrFail(t)

	// Check that the values were set correctly
	var getConfigReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   gnmiClient,
		Encoding: gbp.Encoding_PROTO,
		Paths:    targetPath,
	}
	getConfigReq.CheckValue(t, newValue)

	// Check that the values are set on the target
	target1GnmiClient := gnmiutils.NewSimulatorGNMIClientOrFail(ctx, t, target1)
	var getTargetReq = &gnmiutils.GetRequest{
		Ctx:      ctx,
		Client:   target1GnmiClient,
		Encoding: gbp.Encoding_JSON,
		Paths:    targetPath,
	}
	getTargetReq.CheckValue(t, newValue)

	// Now rollback the change
	adminClient, err := gnmiutils.NewAdminServiceClient(ctx)
	assert.NoError(t, err)
	rollbackResponse, rollbackError := adminClient.RollbackTransaction(context.Background(), &admin.RollbackRequest{Index: transactionIndex})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")

	// Check that the value was really rolled back- should be an error here since the node was deleted
	_, _, err = getTargetReq.Get()
	assert.Error(t, err)
}
