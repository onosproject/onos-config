// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/test"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	baseClient "github.com/openconfig/gnmi/client"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/protobuf/proto"
)

var log = logging.GetLogger("northbound", "gnmi")

// TestSubscribe tests subscribe NB API
func (s *TestSuite) TestSubscribe() {
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
				Mode:   gnmiapi.SubscriptionList_ONCE,
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
}

func getPath(target string, pe ...string) *gnmiapi.Path {
	elem := make([]*gnmiapi.PathElem, 0, len(pe))
	for _, e := range pe {
		elem = append(elem, &gnmiapi.PathElem{Name: e})
	}
	return &gnmiapi.Path{Target: target, Elem: elem}
}

func (s *TestSuite) subscribe(gnmiClient baseClient.Impl, req *gnmiapi.SubscribeRequest,
	updates *[]*gnmiapi.SubscribeResponse_Update) {

	q, err := baseClient.NewQuery(req)
	s.NoError(err)

	q.NotificationHandler = nil
	q.ProtoHandler = func(msg proto.Message) error {
		log.Debugf("Received: %+v", msg)
		resp := msg.(*gnmiapi.SubscribeResponse)
		if update, ok := resp.GetResponse().(*gnmiapi.SubscribeResponse_Update); ok {
			*updates = append(*updates, update)
		}
		return nil
	}

	err = gnmiClient.Subscribe(s.Context(), q)
	s.NoError(err)
}

func (s *TestSuite) waitForResponses(gnmiClient baseClient.Impl, updates *[]*gnmiapi.SubscribeResponse_Update, count int) {
	log.Debugf("Subscribe issued... waiting for responses")
	for {
		err := gnmiClient.Recv()
		log.Debugf("Updates: %+v", *updates)
		if err != nil || len(*updates) >= count {
			break
		}
	}

	s.Len(*updates, count)
}
