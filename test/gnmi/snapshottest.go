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

package gnmi

import (
	"context"
	"github.com/google/uuid"
	"github.com/onosproject/onos-config/api/admin"
	"github.com/onosproject/onos-test/pkg/onit/env"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestSnapshot : test
func (s *TestSuite) TestSnapshot(t *testing.T) {
	add := env.AddSimulators()
	for i := 0; i < 20; i++ {
		add.With(env.NewSimulator())
	}
	simulators := add.AddAllOrDie()

	// Make a GNMI client to use for requests
	c, err := env.Config().NewGNMIClient()
	assert.NoError(t, err)
	assert.True(t, c != nil, "Fetching client returned nil")

	for i := 0; i < 100; i++ {
		for _, simulator := range simulators {
			setPath := makeDevicePath(simulator.Name(), "/system/config/motd-banner")
			setPath[0].pathDataValue = uuid.New().String()
			setPath[0].pathDataType = StringVal
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, _, err := gNMISet(ctx, c, setPath, noPaths)
			cancel()
			assert.NoError(t, err)
		}
	}

	time.Sleep(10 * time.Second)

	adminClient, err := env.Config().NewAdminServiceClient()
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	retention := 5 * time.Second
	_, err = adminClient.CompactChanges(ctx, &admin.CompactChangesRequest{
		RetentionPeriod: &retention,
	})
	cancel()
	assert.NoError(t, err)
}
