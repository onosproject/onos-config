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
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"
	gbp "github.com/openconfig/gnmi/proto/gnmi"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	"github.com/onosproject/onos-config/test/utils/gnmi"
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

	// Get the configured targets from the environment.
	target1 := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, target1)
	targets := make([]string, 1)
	targets[0] = target1.Name()

	// Wait for config to connect to the target
	gnmi.WaitForTargetAvailable(t, topo.ID(target1.Name()), 10*time.Second)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	// Set values
	var targetPathsForSet = gnmi.GetTargetPathsWithValues(targets, newPaths, newValues)
	_, transactionIndex := gnmi.SetGNMIValueOrFail(t, gnmiClient, targetPathsForSet, gnmi.NoPaths, gnmi.NoExtensions)

	err := gnmi.WaitForConfigurationCompleteOrFail(t, configapi.ConfigurationID(target1.Name()), time.Minute)
	assert.NoError(t, err)

	targetPathsForGet := gnmi.GetTargetPaths(targets, newPaths)

	// Check that the values were set correctly
	expectedValues := []string{newValue}
	gnmi.CheckGNMIValues(t, gnmiClient, targetPathsForGet, expectedValues, 0, "Query after set returned the wrong value")

	// Check that the values are set on the targets
	target1GnmiClient := gnmi.GetTargetGNMIClientOrFail(t, target1)
	gnmi.CheckTargetValue(t, target1GnmiClient, targetPathsForGet[0:1], newValue)

	// Now rollback the change
	adminClient, err := gnmi.NewAdminServiceClient()
	assert.NoError(t, err)
	rollbackResponse, rollbackError := adminClient.RollbackTransaction(context.Background(), &admin.RollbackRequest{Index: transactionIndex})

	assert.NoError(t, rollbackError, "Rollback returned an error")
	assert.NotNil(t, rollbackResponse, "Response for rollback is nil")

	err = gnmi.WaitForConfigurationCompleteOrFail(t, configapi.ConfigurationID(target1.Name()), time.Minute)
	assert.NoError(t, err)

	// Check that the value was really rolled back- should be an error here since the node was deleted
	_, _, err = gnmi.GetGNMIValue(gnmi.MakeContext(), target1GnmiClient, targetPathsForGet, gbp.Encoding_PROTO)
	assert.Error(t, err)
}
