// Copyright 2021-present Open Networking Foundation.
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
	"strings"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/onosproject/onos-config/pkg/utils"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// Subscription subscription request information
type Subscription struct {
	UpdatesOnly       bool
	Prefix            string
	Mode              string
	StreamMode        string
	SampleInterval    uint64
	HeartbeatInterval uint64
	Paths             [][]string
	Origin            string
}

// Build builds a gNMI subscription request
func (s *Subscription) Build() (*gnmipb.SubscribeRequest, error) {
	var mode gnmipb.SubscriptionList_Mode
	switch strings.ToUpper(s.Mode) {
	case gnmipb.SubscriptionList_ONCE.String():
		mode = gnmipb.SubscriptionList_ONCE
	case gnmipb.SubscriptionList_POLL.String():
		mode = gnmipb.SubscriptionList_POLL
	case "":
		fallthrough
	case gnmipb.SubscriptionList_STREAM.String():
		mode = gnmipb.SubscriptionList_STREAM
	default:
		return nil, errors.NewInvalid("subscribe mode (%s) invalid", s.Mode)
	}

	var streamMode gnmipb.SubscriptionMode
	switch strings.ToUpper(s.StreamMode) {
	case gnmipb.SubscriptionMode_ON_CHANGE.String():
		streamMode = gnmipb.SubscriptionMode_ON_CHANGE
	case gnmipb.SubscriptionMode_SAMPLE.String():
		streamMode = gnmipb.SubscriptionMode_SAMPLE
	case "":
		fallthrough
	case gnmipb.SubscriptionMode_TARGET_DEFINED.String():
		streamMode = gnmipb.SubscriptionMode_TARGET_DEFINED
	default:
		return nil, errors.NewInvalid("subscribe stream mode (%s) invalid", s.StreamMode)
	}

	prefixPath, err := utils.ParseGNMIElements(utils.SplitPath(s.Prefix))
	if err != nil {
		return nil, err
	}
	subList := &gnmipb.SubscriptionList{
		Subscription: make([]*gnmipb.Subscription, len(s.Paths)),
		Mode:         mode,
		UpdatesOnly:  s.UpdatesOnly,
		Prefix:       prefixPath,
	}
	for i, p := range s.Paths {
		gnmiPath, err := utils.ParseGNMIElements(p)
		if err != nil {
			return nil, err
		}
		gnmiPath.Origin = s.Origin
		subList.Subscription[i] = &gnmipb.Subscription{
			Path:              gnmiPath,
			Mode:              streamMode,
			SampleInterval:    s.SampleInterval,
			HeartbeatInterval: s.HeartbeatInterval,
		}
	}
	return &gnmipb.SubscribeRequest{Request: &gnmipb.SubscribeRequest_Subscribe{
		Subscribe: subList}}, nil
}
