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
//

package config

import (
	"testing"

	"github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-config/test/utils/proto"
	"github.com/openconfig/gnmi/proto/gnmi_ext"
	"github.com/stretchr/testify/assert"
)

const (
	unreachableTargetModPath          = "/system/clock/config/timezone-name"
	unreachableTargetModValue         = "Europe/Rome"
	unreachableTargetModTargetName    = "unreachable-dev-1"
	unreachableTargetModTargetVersion = "1.0.0"
	unreachableTargetModTargetType    = "devicesim-1.0.x"
	unreachableTargetAddress          = "198.18.0.1:4"
)

// TestUnreachableTarget tests set/query of a single GNMI path to a target that will never respond
func (s *TestSuite) TestUnreachableTarget(t *testing.T) {
	createOfflineTarget(t, unreachableTargetModTargetName, unreachableTargetModTargetType, unreachableTargetModTargetVersion, unreachableTargetAddress)

	// Make a GNMI client to use for requests
	gnmiClient := gnmi.GetGNMIClientOrFail(t)

	targetPath := gnmi.GetTargetPathWithValue(unreachableTargetModTargetName, unreachableTargetModPath, unreachableTargetModValue, proto.StringVal)

	// Set the value - should return a pending change
	transactionID, _ := gnmi.SetGNMIValueOrFail(t, gnmiClient, targetPath, gnmi.NoPaths, []*gnmi_ext.Extension{})
	assert.NotNil(t, transactionID)

	// Check that the value was set correctly in the cache
	gnmi.CheckGNMIValue(t, gnmiClient, targetPath, unreachableTargetModValue, 0, "Query after set returned the wrong value")

	// Validate that the network change is listed and shows as pending.
	//changeClient, err := gnmi.NewChangeServiceClient()
	//assert.NoError(t, err)
	//stream, err := changeClient.ListNetworkChanges(context.Background(), &diags.ListNetworkChangeRequest{ChangeID: changeID})
	//assert.NoError(t, err)

	//found := false
	//for {
	//	resp, err := stream.Recv()
	//	if err == io.EOF {
	//		break
	//	}
	//	assert.NoError(t, err)
	//	if resp.Change.ID == changeID && resp.Change.Status.State == change.State_PENDING {
	//		found = true
	//		break
	//	}
	//}
	//assert.True(t, found)

	// FIXME: Rollback won't be supported initially with the onos-config rewrite.
	//adminClient, err := gnmi.NewAdminServiceClient()
	//assert.NoError(t, err)

	// Rollback the set - should be possible even if target is unreachable
	//rollbackResponse, rollbackError := adminClient.RollbackNetworkChange(
	//	context.Background(), &admin.RollbackRequest{Name: string(changeID)})
	//assert.NoError(t, rollbackError, "Rollback returned an error")
	//assert.NotNil(t, rollbackResponse, "Response for rollback is nil")
	//assert.Contains(t, rollbackResponse.Message, changeID, "rollbackResponse message does not contain change ID")
}
