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

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	topoutils "github.com/onosproject/onos-config/test/utils/topo"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
	"github.com/stretchr/testify/assert"
)

// checkControlRelation queries a single control relation between the config and target, and fails if not found
func (s *TestSuite) checkControlRelations(ctx context.Context, t *testing.T, targetID topoapi.ID, c toposdk.Client) []topoapi.Object {
	numberOfConfigNodes := int(s.ConfigReplicaCount)
	numberOfTargets := 1
	numberOfControlRelationships := numberOfTargets * numberOfConfigNodes

	relationObjects, err := c.List(ctx, toposdk.WithListFilters(topoutils.GetControlRelationFilter()))
	assert.NoError(t, err)
	var filteredRelations []topoapi.Object
	for _, relationObject := range relationObjects {
		if relationObject.GetRelation().TgtEntityID == targetID {
			filteredRelations = append(filteredRelations, relationObject)
		}
	}
	assert.Equal(t, numberOfControlRelationships, len(filteredRelations))
	return filteredRelations
}

// checkTopo makes sure that the topo entries are correct
func (s *TestSuite) checkTopo(t *testing.T, targetID topoapi.ID) {

	// Get a topology API client
	client, err := gnmiutils.NewTopoClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// check number of control relations from onos-config instances to the simulated target
	relationObjects := s.checkControlRelations(context.Background(), t, targetID, client)
	for _, relationObject := range relationObjects {
		controlRelation := relationObject.GetRelation()
		onosConfigNode, err := client.Get(context.Background(), controlRelation.SrcEntityID)
		assert.NoError(t, err)
		leaseAspect := &topoapi.Lease{}
		err = onosConfigNode.GetAspect(leaseAspect)
		assert.NoError(t, err)
		// Find the entity for the simulated target
		targetObject, err := client.Get(context.Background(), controlRelation.TgtEntityID)
		assert.NoError(t, err)
		// Check the target simulator aspects
		configurable := &topoapi.Configurable{}
		err = targetObject.GetAspect(configurable)
		assert.NoError(t, err)
	}
}

// TestTopoIntegration checks that the correct topology entities and relations are created
func (s *TestSuite) TestTopoIntegration(t *testing.T) {
	// Create simulated targets
	targetID := "test-topo-integration-target-1"
	simulator := gnmiutils.CreateSimulatorWithName(t, targetID)
	assert.NotNil(t, simulator)
	gnmiutils.WaitForTargetAvailable(t, topoapi.ID(targetID), 2*time.Minute)
	defer gnmiutils.DeleteSimulator(t, simulator)
	s.checkTopo(t, topoapi.ID(targetID))

}
