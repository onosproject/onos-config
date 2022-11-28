// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiutils "github.com/onosproject/onos-config/test/utils/gnmi"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	baseClient "github.com/openconfig/gnmi/client"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"testing"
	"time"
)

var log = logging.GetLogger("northbound", "gnmi")

// TestSubscribe tests subscribe NB API
func (s *TestSuite) TestSubscribe(t *testing.T) {
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
				Mode:   gnmiapi.SubscriptionList_ONCE,
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
}

func getPath(target string, pe ...string) *gnmiapi.Path {
	elem := make([]*gnmiapi.PathElem, 0, len(pe))
	for _, e := range pe {
		elem = append(elem, &gnmiapi.PathElem{Name: e})
	}
	return &gnmiapi.Path{Target: target, Elem: elem}
}

func subscribe(ctx context.Context, t *testing.T, gnmiClient baseClient.Impl, req *gnmiapi.SubscribeRequest,
	updates *[]*gnmiapi.SubscribeResponse_Update) {

	q, err := baseClient.NewQuery(req)
	assert.NoError(t, err)

	q.NotificationHandler = nil
	q.ProtoHandler = func(msg proto.Message) error {
		log.Debugf("Received: %+v", msg)
		resp := msg.(*gnmiapi.SubscribeResponse)
		if update, ok := resp.GetResponse().(*gnmiapi.SubscribeResponse_Update); ok {
			*updates = append(*updates, update)
		}
		return nil
	}

	err = gnmiClient.Subscribe(ctx, q)
	assert.NoError(t, err)
}

func waitForResponses(t *testing.T, gnmiClient baseClient.Impl, updates *[]*gnmiapi.SubscribeResponse_Update, count int) {
	log.Debugf("Subscribe issued... waiting for responses")
	for {
		err := gnmiClient.Recv()
		log.Debugf("Updates: %+v", *updates)
		if err != nil || len(*updates) >= count {
			break
		}
	}

	assert.Len(t, *updates, count)
}
