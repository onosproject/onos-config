// SPDX-FileCopyrightText: 2020-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	baseClient "github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"io"
	"strings"
)

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	log.Info("Received gNMI Subscribe stream")
	groups := make([]string, 0)
	if md := metautils.ExtractIncoming(stream.Context()); md != nil && md.Get("name") != "" {
		groups = append(groups, strings.Split(md.Get("groups"), ";")...)
		log.Debugf("gNMI Get() called by '%s (%s)'. Groups %v. Token %s",
			md.Get("name"), md.Get("email"), groups, md.Get("at_hash"))
	}

	log.Info("Waiting for subscription messages")
	subscribed := false
	for {
		req, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				// Cancel SB requests and exit normally
				log.Info("Client closed the subscription stream")
				return nil
			}
			// Cancel SB requests and exit with error
			log.Warn(err)
			return err
		}

		log.Info("Received gNMI Subscribe Request: %+v", req)
		if !subscribed {
			subscribed = true
			err = s.processSubscribeRequest(stream.Context(), stream, req)
			if err != nil {
				log.Warn(err)
				return err
			}
		}
	}
}

// Determine the target, pass the request onto it and relay any events onto the NB stream
func (s *Server) processSubscribeRequest(ctx context.Context, stream gnmi.GNMI_SubscribeServer, req *gnmi.SubscribeRequest) error {
	targetReqs, err := splitRequest(req)
	if err != nil {
		return err
	}

	log.Info(targetReqs)
	for target, targetReq := range targetReqs {
		_ = s.sendSubscriptionRequest(ctx, stream, target, targetReq)
	}
	return nil
}

// Send the specified request to the target, creating new subscribe stream if needed together with a watcher
// that relay any SB events onto the NB stream
func (s *Server) sendSubscriptionRequest(ctx context.Context, stream gnmi.GNMI_SubscribeServer,
	target string, req *gnmi.SubscribeRequest) error {
	// Check if there is already a stream for the specified target; if not, create one
	client, err := s.conns.GetByTarget(ctx, topo.ID(target))
	if err != nil {
		return err
	}

	// Queue up the request to the target
	query, err := baseClient.NewQuery(req)
	if err != nil {
		log.Warn("Unable to create query", err)
		return err
	}

	query.NotificationHandler = nil
	query.ProtoHandler = func(msg proto.Message) error {
		log.Infof("Received response from target %s: %+v", target, msg)
		resp, ok := msg.(*gnmi.SubscribeResponse)
		if !ok {
			log.Warn("Failed to type assert message %#v", msg)
			return errors.NewInvalid("Failed to type assert message %#v", msg)
		}
		log.Infof("Forwarding response from target %s to client: %+v", target, resp)
		return stream.Send(resp)
	}

	log.Infof("Forwarding subscription query to target %s: %+v", target, query)
	return client.Subscribe(ctx, query)
}

// Iterate over the paths in the subscription list and split the request into a multiple requests of the same type,
// each for a single target.
func splitRequest(req *gnmi.SubscribeRequest) (map[string]*gnmi.SubscribeRequest, error) {
	if req.GetSubscribe() != nil {
		return splitSubscribeRequest(req)
	} else if req.GetPoll() != nil {
		return splitPollRequest(req)
	}
	return nil, errors.NewInvalid("Request is neither subscribe nor poll")
}

func splitSubscribeRequest(req *gnmi.SubscribeRequest) (map[string]*gnmi.SubscribeRequest, error) {
	targets := make(map[string]*gnmi.SubscribeRequest)
	subs := req.GetSubscribe()

	prefixTarget := subs.Prefix.Target // fallback target for a single-target request

	// If the prefix names a target, it is assumed this is a single-target request and the original request
	// becomes the request for that target.
	if prefixTarget != "" {
		targets[prefixTarget] = req
		return targets, nil
	}

	// Otherwise, iterate over the subscriptions and separate them into multiple subscription requests
	// based on the target specified in each path using the original request as a template.
	for _, sub := range subs.Subscription {
		target := sub.Path.Target
		var tr *gnmi.SubscribeRequest
		if target != "" {
			ok := false
			if tr, ok = targets[target]; !ok {
				tr = &gnmi.SubscribeRequest{
					Request: &gnmi.SubscribeRequest_Subscribe{
						Subscribe: &gnmi.SubscriptionList{
							Prefix:           copyPrefix(subs.Prefix, target),
							Subscription:     make([]*gnmi.Subscription, 0, 1),
							UseAliases:       subs.UseAliases,
							Qos:              subs.Qos,
							Mode:             subs.Mode,
							AllowAggregation: subs.AllowAggregation,
							UseModels:        subs.UseModels,
							Encoding:         subs.Encoding,
							UpdatesOnly:      subs.UpdatesOnly,
						},
					},
					Extension: req.Extension,
				}
				targets[target] = tr
			}
			tr.GetSubscribe().Subscription = append(tr.GetSubscribe().Subscription, sub)
		}
	}

	if len(targets) == 0 {
		return nil, errors.NewInvalid("Prefix or at least one path must specify a target")
	} else if prefixTarget != "" {
		return nil, errors.NewInvalid("Prefix not supported for multi-target request")
	}
	return targets, nil
}

func copyPrefix(prefix *gnmi.Path, target string) *gnmi.Path {
	return &gnmi.Path{
		Origin: prefix.Origin,
		Elem:   prefix.Elem,
		Target: target,
	}
}

func splitPollRequest(req *gnmi.SubscribeRequest) (map[string]*gnmi.SubscribeRequest, error) {
	targets := make(map[string]*gnmi.SubscribeRequest)
	return targets, nil
}
