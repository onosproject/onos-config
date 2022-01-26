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
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test/utils/gnmi"
	toposdk "github.com/onosproject/onos-ric-sdk-go/pkg/topo"
	testutils "github.com/onosproject/onos-test/pkg/utils"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// getEntityOrDie queries an entity of the given kind and fails if it isn't found
func getEntityOrDie(ctx context.Context, t *testing.T, c toposdk.Client, kind string) topo.Object {
	filter := &topo.Filters{
		KindFilter: &topo.Filter{
			Filter: &topo.Filter_Equal_{Equal_: &topo.EqualFilter{Value: kind}},
		}}
	entities, err := c.List(ctx, toposdk.WithListFilters(filter))
	assert.NoError(t, err)
	assert.Len(t, entities, 1)
	return entities[0]
}

// getOnosConfigEntityOrDie queries a single onos-config instance and fails if it isn't found
func getOnosConfigEntityOrDie(ctx context.Context, t *testing.T, c toposdk.Client) topo.Object {
	return getEntityOrDie(ctx, t, c, topo.ONOS_CONFIG)
}

// getDevicesimEntityOrDie queries a single device simulator instance and fails if it isn't found
func getDevicesimEntityOrDie(ctx context.Context, t *testing.T, c toposdk.Client) topo.Object {
	const (
		devicesimKind = "devicesim-1.0.x"
	)
	return getEntityOrDie(ctx, t, c, devicesimKind)
}

// getRelationOrDie queries a single control relation between the config and target, and fails if not found
func getRelationOrDie(ctx context.Context, t *testing.T, configID topo.ID, targetID topo.ID, c toposdk.Client) topo.Object {
	filter := &topo.Filters{
		RelationFilter: &topo.RelationFilter{
			SrcId:        string(configID),
			TargetId:     string(targetID),
			Scope:        topo.RelationFilterScope_ALL,
			RelationKind: topo.CONTROLS,
		},
	}

	objs, err := c.List(ctx, toposdk.WithListFilters(filter))
	assert.NoError(t, err)
	assert.Len(t, objs, 3)

	for _, obj := range objs {
		if rel := obj.GetRelation(); rel != nil {
			return obj
		}
	}

	assert.Fail(t, "Could not find relation")
	return topo.Object{}
}

// waiForTopoEntries waits for the topo entries to be populated
func waitForTopoEntriesOrDie(t *testing.T, err error, client toposdk.Client) {
	var objs []topo.Object
	maxWaitSeconds := 30
	assert.True(t, testutils.Retry(maxWaitSeconds, time.Second,
		func() bool {
			objs, err = client.List(context.Background())
			assert.NoError(t, err)
			return len(objs) >= 3
		}))
	assert.NoError(t, err)
}

// checkTopo makes sure that the topo entries are correct
func checkTopo(t *testing.T) {
	var (
		configEntity *topo.Entity
		configObject topo.Object
		configID     topo.ID

		targetEntity *topo.Entity
		targetObject topo.Object
		targetID     topo.ID

		relation       *topo.Relation
		relationObject topo.Object
	)

	// Get a topology API client
	client, err := gnmi.NewTopoClient()
	assert.NoError(t, err)
	assert.NotNil(t, client)

	// Wait for the topo entries to be set up
	waitForTopoEntriesOrDie(t, err, client)

	// Find the entity for onos-config
	configObject = getOnosConfigEntityOrDie(context.Background(), t, client)
	configID = configObject.ID
	configEntity = configObject.GetEntity()

	// Find the entity for the simulated target
	targetObject = getDevicesimEntityOrDie(context.Background(), t, client)
	targetID = targetObject.ID
	targetEntity = targetObject.GetEntity()

	// Find the control relation from onos-config to the simulated target
	relationObject = getRelationOrDie(context.Background(), t, configID, targetID, client)
	relation = relationObject.GetRelation()

	// Check that we found a controls relation
	assert.NotNil(t, relation)
	assert.Equal(t, topo.ID(topo.CONTROLS), relation.KindID)

	// Check that the source and target IDs are correct
	assert.Len(t, configEntity.SrcRelationIDs, 1)
	assert.Len(t, targetEntity.TgtRelationIDs, 1)

	configRelationID := configEntity.SrcRelationIDs[0]
	targetRelationID := targetEntity.TgtRelationIDs[0]

	assert.Equal(t, configRelationID, targetRelationID)
	assert.Equal(t, configID, relation.SrcEntityID)
	assert.Equal(t, targetID, relation.TgtEntityID)

	// Check onos-config aspects
	assert.Len(t, configObject.Aspects, 1)
	lease := &topo.Lease{}
	err = configObject.GetAspect(lease)
	assert.NoError(t, err)

	// Check the target simulator aspects
	assert.Greater(t, len(targetObject.Aspects), 1)
	configurable := &topo.Configurable{}
	err = targetObject.GetAspect(configurable)
	assert.NoError(t, err)

	mastershipState := &topo.MastershipState{}
	err = targetObject.GetAspect(mastershipState)
	assert.NoError(t, err)
	assert.Equal(t, relationObject.ID, topo.ID(mastershipState.NodeId))
}

// TestTopoIntegration checks that the correct topology entities and relations are created
func (s *TestSuite) TestTopoIntegration(t *testing.T) {
	// Create a simulated device
	simulator := gnmi.CreateSimulator(t)
	defer gnmi.DeleteSimulator(t, simulator)

	assert.NotNil(t, simulator)

	checkTopo(t)
}
