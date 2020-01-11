// Copyright 2020-present Open Networking Foundation.
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

package gnmi

import (
	"context"
	"github.com/onosproject/onos-config/api/admin"
	"github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestSnapshotErrors tests the Snapshot RPCs on the Admin gRPC interface
// Se also TestCompactChanges
func (s *TestSuite) TestSnapshotErrors(t *testing.T) {
	// Data to run the test cases
	testCases := []struct {
		description   string
		id            string
		version       string
		expectedError string
	}{
		{description: "No version", id: "devicesim-1", version: "", expectedError: "An ID and Version must be given"},
		{description: "No snapshot", id: "devicesim-1", version: "1.0.0", expectedError: "No snapshot found"},
	}

	adminClient, err := env.Config().NewAdminServiceClient()
	assert.NoError(t, err)

	// Run the test cases
	for _, testCase := range testCases {
		thisTestCase := testCase
		t.Run(thisTestCase.description,
			func(t *testing.T) {
				description := device.ID(thisTestCase.description)
				id := device.ID(thisTestCase.id)
				version := device.Version(thisTestCase.version)
				expectedError := thisTestCase.expectedError

				t.Logf("testing %q", description)

				snapshot, err := adminClient.GetSnapshot(context.Background(), &admin.GetSnapshotRequest{
					DeviceID:      id,
					DeviceVersion: version,
				})
				assert.Nil(t, snapshot, "Expected snapshot to be nil")
				assert.Error(t, err, "Expected error response", expectedError)
			})
	}
}
