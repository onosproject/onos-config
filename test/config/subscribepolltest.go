// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// TestSubscribePoll tests subscribe NB API with client-side poll
func (s *TestSuite) TestSubscribePoll(t *testing.T) {
	generateTimezoneName()

	ctx, cancel := gnmiutils.MakeContext()
	defer cancel()

	// Create two simulated devices
	target1 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target1)
	target2 := gnmiutils.CreateSimulator(ctx, t)
	defer gnmiutils.DeleteSimulator(t, target2)

	// Wait for config to connect to the target
	ready := gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(target1.Name()), 1*time.Minute)
	assert.True(t, ready)
	ready = gnmiutils.WaitForTargetAvailable(ctx, t, topoapi.ID(target2.Name()), 1*time.Minute)
	assert.True(t, ready)

	// Make a GNMI client to use for requests
	gnmiClient := gnmiutils.NewOnosConfigGNMIClientOrFail(ctx, t, gnmiutils.NoRetry)

	sr := &gnmiapi.SubscribeRequest{
		Request: &gnmiapi.SubscribeRequest_Subscribe{
			Subscribe: &gnmiapi.SubscriptionList{
				Prefix: &gnmiapi.Path{Target: "", Elem: []*gnmiapi.PathElem{}},
				Mode:   gnmiapi.SubscriptionList_POLL,
				Subscription: []*gnmiapi.Subscription{{
					Path: getPath(target1.Name(), "system", "state", "current-datetime"),
					Mode: gnmiapi.SubscriptionMode_SAMPLE,
				}, {
					Path: getPath(target2.Name(), "system", "state", "current-datetime"),
					Mode: gnmiapi.SubscriptionMode_SAMPLE,
				}},
				Encoding:    gnmiapi.Encoding_PROTO,
				UpdatesOnly: false,
			},
		},
	}
	updates := make([]*gnmiapi.SubscribeResponse_Update, 0, 4)
	subscribe(ctx, t, gnmiClient, sr, &updates)
	waitForResponses(t, gnmiClient, &updates, 2)

	err := gnmiClient.Poll()
	assert.NoError(t, err)
	waitForResponses(t, gnmiClient, &updates, 4)
}
