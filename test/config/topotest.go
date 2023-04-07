// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
)

// checkControlRelation queries a single control relation between the config and target, and fails if not found
func (s *TestSuite) checkControlRelations(targetID topoapi.ID, c toposdk.Client) []topoapi.Object {
	numberOfConfigNodes := s.umbrella.Get("onos-config.replicaCount").Int()
	numberOfTargets := 1
	numberOfControlRelationships := numberOfTargets * numberOfConfigNodes

	relationObjects, err := c.List(s.Context(), toposdk.WithListFilters(&topoapi.Filters{
		KindFilter: &topoapi.Filter{
			Filter: &topoapi.Filter_Equal_{
				Equal_: &topoapi.EqualFilter{
					Value: topoapi.CONTROLS,
				},
			},
		},
	}))
	s.NoError(err)
	var filteredRelations []topoapi.Object
	for _, relationObject := range relationObjects {
		if relationObject.GetRelation().TgtEntityID == targetID {
			filteredRelations = append(filteredRelations, relationObject)
		}
	}
	s.Equal(numberOfControlRelationships, len(filteredRelations))
	return filteredRelations
}

// checkTopo makes sure that the topo entries are correct
func (s *TestSuite) checkTopo(targetID topoapi.ID) {

	// Get a topology API client
	client, err := s.NewTopoClient()
	s.NoError(err)
	s.NotNil(client)

	// check number of control relations from onos-config instances to the simulated target
	relationObjects := s.checkControlRelations(targetID, client)
	for _, relationObject := range relationObjects {
		controlRelation := relationObject.GetRelation()
		// Gets onos-config master node associated with the target
		onosConfigNode, err := client.Get(context.Background(), controlRelation.SrcEntityID)
		s.NoError(err)
		leaseAspect := &topoapi.Lease{}
		err = onosConfigNode.GetAspect(leaseAspect)
		s.NoError(err)
		// Find the entity for the simulated target
		targetObject, err := client.Get(context.Background(), controlRelation.TgtEntityID)
		s.NoError(err)
		// Check the target simulator aspects
		configurable := &topoapi.Configurable{}
		err = targetObject.GetAspect(configurable)
		s.NoError(err)
	}
}

// TestTopoIntegration checks that the correct topology entities and relations are created
func (s *TestSuite) TestTopoIntegration() {
	// Create simulated targets
	targetID := "test-topo-integration-target-1"
	s.SetupSimulator(targetID, true)
	s.WaitForTargetAvailable(topoapi.ID(targetID))
	defer s.TearDownSimulator(targetID)
	s.checkTopo(topoapi.ID(targetID))
}
