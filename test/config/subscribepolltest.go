// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
)

// TestSubscribePoll tests subscribe NB API with client-side poll
func (s *TestSuite) TestSubscribePoll() {
	generateTimezoneName()

	// Wait for config to connect to the target
	ready := s.WaitForTargetAvailable(topoapi.ID(s.simulator1.Name))
	s.True(ready)
	ready = s.WaitForTargetAvailable(topoapi.ID(s.simulator2.Name))
	s.True(ready)

	// Make a GNMI client to use for requests
	gnmiClient := s.NewOnosConfigGNMIClientOrFail(test.NoRetry)

	sr := &gnmiapi.SubscribeRequest{
		Request: &gnmiapi.SubscribeRequest_Subscribe{
			Subscribe: &gnmiapi.SubscriptionList{
				Prefix: &gnmiapi.Path{Target: "", Elem: []*gnmiapi.PathElem{}},
				Mode:   gnmiapi.SubscriptionList_POLL,
				Subscription: []*gnmiapi.Subscription{{
					Path: getPath(s.simulator1.Name, "system", "state", "current-datetime"),
					Mode: gnmiapi.SubscriptionMode_SAMPLE,
				}, {
					Path: getPath(s.simulator2.Name, "system", "state", "current-datetime"),
					Mode: gnmiapi.SubscriptionMode_SAMPLE,
				}},
				Encoding:    gnmiapi.Encoding_PROTO,
				UpdatesOnly: false,
			},
		},
	}
	updates := make([]*gnmiapi.SubscribeResponse_Update, 0, 4)
	s.subscribe(gnmiClient, sr, &updates)
	s.waitForResponses(gnmiClient, &updates, 2)

	err := gnmiClient.Poll()
	s.NoError(err)
	s.waitForResponses(gnmiClient, &updates, 4)
}
