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
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"

	"github.com/onosproject/onos-api/go/onos/topo"

	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
)

// TestGetAllTargets tests retrieval of all target IDs via path.Target="*"
func (s *TestSuite) TestGetAllTargets(t *testing.T) {
	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create two target simulators
	target1 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target1)
	target2 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target2)

	// Wait for config to connect to both simulators
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target1.Name()), time.Minute)
	gnmiutils.WaitForTargetAvailable(ctx, t, topo.ID(target2.Name()), time.Minute)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	// Get the list of all targets via get query on target "*"
	var getReq = &gnmiutils.GetRequest{
		Ctx:        ctx,
		Client:     gnmiClient,
		Encoding:   gnmiapi.Encoding_PROTO,
		Extensions: gnmiutils.SyncExtension(t),
		Paths:      gnmiutils.GetTargetPath("*", ""),
	}
	getValue, err := getReq.Get()
	assert.NoError(t, err)
	assert.Len(t, getValue, 1)
	assert.Equal(t, "/all-targets", getValue[0].Path)
	assert.True(t, strings.Contains(getValue[0].PathDataValue, target1.Name()))
	assert.True(t, strings.Contains(getValue[0].PathDataValue, target2.Name()))
}
