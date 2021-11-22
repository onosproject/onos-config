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

package v070

import (
	"strings"

	"github.com/onosproject/onos-lib-go/pkg/errors"

	"github.com/onosproject/onos-config/pkg/utils"
	gnmipb "github.com/openconfig/gnmi/proto/gnmi"
)

// NewSubscription creates a new subscription
func NewSubscription(options ...func(subscription *Subscription)) *Subscription {
	subscription := &Subscription{}
	for _, option := range options {
		option(subscription)
	}

	return subscription
}

// Subscription subscription request information
type Subscription struct {
	updatesOnly       bool
	prefix            string
	mode              string
	streamMode        string
	sampleInterval    uint64
	heartbeatInterval uint64
	paths             [][]string
	origin            string
}

// WithUpdatesOnly sets updatesOnly field
func WithUpdatesOnly(updatesOnly bool) func(subscription *Subscription) {
	return func(subscription *Subscription) {
		subscription.updatesOnly = updatesOnly
	}
}

// WithPrefix sets subscription prefix field
func WithPrefix(prefix string) func(subscription *Subscription) {
	return func(subscription *Subscription) {
		subscription.prefix = prefix
	}
}

// WithMode sets subscription mode field
func WithMode(mode string) func(subscription *Subscription) {
	return func(subscription *Subscription) {
		subscription.mode = mode
	}
}

// WithStreamMode sets subscription stream mode
func WithStreamMode(streamMode string) func(subscription *Subscription) {
	return func(subscription *Subscription) {
		subscription.streamMode = streamMode
	}
}

// WithSampleInterval sets subscription sample interval
func WithSampleInterval(sampleInterval uint64) func(subscription *Subscription) {
	return func(subscription *Subscription) {
		subscription.sampleInterval = sampleInterval
	}
}

// WithHeartbeatInterval sets heartbeat interval
func WithHeartbeatInterval(heartbeatInterval uint64) func(subscription *Subscription) {
	return func(subscription *Subscription) {
		subscription.heartbeatInterval = heartbeatInterval
	}
}

// WithPaths sets subscription paths
func WithPaths(paths [][]string) func(subscription *Subscription) {
	return func(subscription *Subscription) {
		subscription.paths = paths
	}
}

// WithOrigin sets origin
func WithOrigin(origin string) func(subscription *Subscription) {
	return func(subscription *Subscription) {
		subscription.origin = origin
	}
}

// Build builds a gNMI subscription request
func (s *Subscription) Build() (*gnmipb.SubscribeRequest, error) {
	var mode gnmipb.SubscriptionList_Mode
	switch strings.ToUpper(s.mode) {
	case gnmipb.SubscriptionList_ONCE.String():
		mode = gnmipb.SubscriptionList_ONCE
	case gnmipb.SubscriptionList_POLL.String():
		mode = gnmipb.SubscriptionList_POLL
	case "":
		fallthrough
	case gnmipb.SubscriptionList_STREAM.String():
		mode = gnmipb.SubscriptionList_STREAM
	default:
		return nil, errors.NewInvalid("subscribe mode (%s) invalid", s.mode)
	}

	var streamMode gnmipb.SubscriptionMode
	switch strings.ToUpper(s.streamMode) {
	case gnmipb.SubscriptionMode_ON_CHANGE.String():
		streamMode = gnmipb.SubscriptionMode_ON_CHANGE
	case gnmipb.SubscriptionMode_SAMPLE.String():
		streamMode = gnmipb.SubscriptionMode_SAMPLE
	case "":
		fallthrough
	case gnmipb.SubscriptionMode_TARGET_DEFINED.String():
		streamMode = gnmipb.SubscriptionMode_TARGET_DEFINED
	default:
		return nil, errors.NewInvalid("subscribe stream mode (%s) invalid", s.streamMode)
	}

	prefixPath, err := utils.ParseGNMIElements(utils.SplitPath(s.prefix))
	if err != nil {
		return nil, err
	}
	subList := &gnmipb.SubscriptionList{
		Subscription: make([]*gnmipb.Subscription, len(s.paths)),
		Mode:         mode,
		UpdatesOnly:  s.updatesOnly,
		Prefix:       prefixPath,
	}
	for i, p := range s.paths {
		gnmiPath, err := utils.ParseGNMIElements(p)
		if err != nil {
			return nil, err
		}
		gnmiPath.Origin = s.origin
		subList.Subscription[i] = &gnmipb.Subscription{
			Path:              gnmiPath,
			Mode:              streamMode,
			SampleInterval:    s.sampleInterval,
			HeartbeatInterval: s.heartbeatInterval,
		}
	}
	return &gnmipb.SubscribeRequest{Request: &gnmipb.SubscribeRequest_Subscribe{
		Subscribe: subList}}, nil
}
