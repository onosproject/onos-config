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
	"fmt"
	"github.com/onosproject/helmit/pkg/helm"
	gnmi2 "github.com/onosproject/onos-config/test/utils/gnmi"

	"github.com/golang/protobuf/proto"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"

	"strings"
	"testing"
)

var log = logging.GetLogger("test", "gnmi")

const (
	subTzValue      = "Europe/Madrid"
	subTzPath       = "/system/clock/config/timezone-name"
	subDateTimePath = "/system/state/current-datetime"
)

type subscribeRequest struct {
	path          *gnmi.Path
	subListMode   gnmi.SubscriptionList_Mode // Mode of the subscription: ONCE, STREAM, POLL
	subStreamMode gnmi.SubscriptionMode      // mode of subscribe STREAM: ON_CHANGE, SAMPLE, TARGET_DEFINED
}

func buildRequest(subReq subscribeRequest) (*gnmi.SubscribeRequest, error) {
	prefixPath, err := utils.ParseGNMIElements(utils.SplitPath(""))
	if err != nil {
		return nil, err
	}
	subscription := &gnmi.Subscription{}
	switch subReq.subListMode {
	case gnmi.SubscriptionList_STREAM:
		subscription = &gnmi.Subscription{
			Path: subReq.path,
			Mode: subReq.subStreamMode,
		}
	case gnmi.SubscriptionList_ONCE:
		subscription = &gnmi.Subscription{
			Path: subReq.path,
		}
	case gnmi.SubscriptionList_POLL:
		subscription = &gnmi.Subscription{
			Path: subReq.path,
		}
	}
	subscriptions := make([]*gnmi.Subscription, 0)
	subscriptions = append(subscriptions, subscription)
	subList := &gnmi.SubscriptionList{
		Subscription: subscriptions,
		Mode:         subReq.subListMode,
		UpdatesOnly:  true,
		Prefix:       prefixPath,
	}
	request := &gnmi.SubscribeRequest{
		Request: &gnmi.SubscribeRequest_Subscribe{
			Subscribe: subList,
		},
	}
	return request, nil
}

func buildQuery(request *gnmi.SubscribeRequest, simulator *helm.HelmRelease) (*client.Query, chan *gnmi.SubscribeResponse, error) {
	q, errQuery := client.NewQuery(request)
	if errQuery != nil {
		return nil, nil, errQuery
	}

	dest, _ := gnmi2.GetDeviceDestination(simulator)

	q.Addrs = dest.Addrs
	q.Timeout = dest.Timeout
	q.Target = dest.Target
	q.Credentials = dest.Credentials
	q.TLS = dest.TLS

	respChan := make(chan *gnmi.SubscribeResponse)

	q.ProtoHandler = func(msg proto.Message) error {
		resp, ok := msg.(*gnmi.SubscribeResponse)
		if !ok {
			return fmt.Errorf("failed to type assert message %#v", msg)
		}
		respChan <- resp
		return nil
	}

	return &q, respChan, nil
}

func buildQueryRequest(subReq subscribeRequest, simulator *helm.HelmRelease) (*client.Query, chan *gnmi.SubscribeResponse, error) {
	request, errReq := buildRequest(subReq)
	if errReq != nil {
		return nil, nil, errReq
	}

	q, respChan, errQuery := buildQuery(request, simulator)
	if errQuery != nil {
		return nil, nil, errReq
	}
	return q, respChan, nil
}

func assertPathSegments(t *testing.T, pathResponse *gnmi.Path, path string) {
	pathSegments := strings.Split(path, "/")[1:]
	assert.Equal(t, len(pathSegments), len(pathResponse.Elem), "Path element count is incorrect")

	for i, segment := range pathSegments {
		assert.Equal(t, segment, pathResponse.Elem[i].Name)
	}
}

func assertUpdateResponse(t *testing.T, response *gnmi.SubscribeResponse_Update, device string, path string, expectedValue string) {
	assert.True(t, response.Update != nil, "Update should not be nil")
	assert.Equal(t, len(response.Update.GetUpdate()), 1)
	if response.Update.GetUpdate()[0] == nil {
		log.Error("response should contain at least one update and that should not be nil")
		t.FailNow()
	}
	pathResponse := response.Update.GetUpdate()[0].Path
	assert.Equal(t, pathResponse.Target, device)
	assertPathSegments(t, pathResponse, path)

	if expectedValue != "" {
		assert.Equal(t, expectedValue, response.Update.GetUpdate()[0].Val.GetStringVal())
	}
}

func assertDeleteResponse(t *testing.T, response *gnmi.SubscribeResponse_Update, device string, path string) {
	assert.True(t, response.Update != nil, "Delete should not be nil")
	assert.Equal(t, len(response.Update.GetDelete()), 1)
	if response.Update.GetDelete()[0] == nil {
		log.Error("response should contain at least one delete path")
		t.FailNow()
	}
	pathResponse := response.Update.GetDelete()[0]
	assert.Equal(t, pathResponse.Target, device)
	pathSegments := strings.Split(path, "/")[1:]
	assert.Equal(t, len(pathSegments), len(pathResponse.Elem), "Path element count is incorrect")

	for i, segment := range pathSegments {
		assert.Equal(t, segment, pathResponse.Elem[i].Name)
	}
	assertPathSegments(t, pathResponse, path)
}

func assertSyncResponse(t *testing.T, sync *gnmi.SubscribeResponse_SyncResponse) {
	t.Helper()
	assert.True(t, sync.SyncResponse)
}
