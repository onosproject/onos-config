// SPDX-FileCopyrightText: 2020-present Intel Corporation
// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package gnmi

import (
	"context"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	baseClient "github.com/openconfig/gnmi/client"
	"github.com/openconfig/gnmi/proto/gnmi"
	"google.golang.org/protobuf/proto"
	"io"
	"strings"
)

// Tracks the NB stream, the originating NB subscription request and the split SB requests
// for the referenced targets and possibly other request related context.
type subContext struct {
	stream gnmi.GNMI_SubscribeServer
	req    *gnmi.SubscribeRequest
	treqs  map[string]*gnmi.SubscribeRequest
}

// Subscribe implements gNMI Subscribe
func (s *Server) Subscribe(stream gnmi.GNMI_SubscribeServer) error {
	log.Info("Received gNMI Subscribe stream")
	groups := make([]string, 0)
	if md := metautils.ExtractIncoming(stream.Context()); md != nil && md.Get("name") != "" {
		groups = append(groups, strings.Split(md.Get("groups"), ";")...)
		log.Debugf("gNMI Get() called by '%s (%s)'. Groups %v. Token %s",
			md.Get("name"), md.Get("email"), groups, md.Get("at_hash"))
	}

	sctx := &subContext{stream: stream}

	log.Info("Waiting for subscription messages")
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

		log.Infof("Received gNMI Subscribe Request: %+v", req)
		err = s.processSubscribeRequest(stream.Context(), sctx, req)
		if err != nil {
			log.Warn(err)
			return err
		}
	}
}

// Determine the target, pass the request onto it and relay any events onto the NB stream
func (s *Server) processSubscribeRequest(ctx context.Context, sctx *subContext, req *gnmi.SubscribeRequest) error {
	if req.GetSubscribe() != nil && sctx.req != nil {
		return errors.NewInvalid("duplicate subscription message detected")
	} else if req.GetPoll() != nil && sctx.req == nil {
		return errors.NewInvalid("subscription request not received yet")

	} else if req.GetSubscribe() != nil {
		// If the request is the subscription, remember it and split it between the different targets
		sctx.req = req
		err := splitSubscribeRequest(sctx, req)
		if err != nil {
			return err
		}

		// ... and relay it to each target using its specific subscription
		log.Debugf("Split target requests: %+v", sctx.treqs)
		for target, targetReq := range sctx.treqs {
			_ = s.sendSubscriptionRequest(ctx, sctx, target, targetReq)
		}

	} else if req.GetPoll() != nil {
		// If the request is a poll, relay it to the previously subscribed targets
		log.Debugf("Relaying poll to targets")
		for target := range sctx.treqs {
			_ = s.sendPollRequest(ctx, target)
		}
	} else {
		return errors.NewInvalid("unknown subscription message type")
	}

	return nil
}

// Send the specified request to the target, creating new subscribe stream if needed together with a watcher
// that relay any SB events onto the NB stream
func (s *Server) sendSubscriptionRequest(ctx context.Context, sctx *subContext, target string, req *gnmi.SubscribeRequest) error {
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
		return sctx.stream.Send(resp)
	}

	log.Infof("Forwarding subscription query to target %s: %+v", target, query)
	return client.Subscribe(ctx, query)
}

// Send a poll request to the target
func (s *Server) sendPollRequest(ctx context.Context, target string) error {
	// Check if there is already a stream for the specified target; if not, create one
	client, err := s.conns.GetByTarget(ctx, topo.ID(target))
	if err != nil {
		return err
	}

	log.Infof("Forwarding poll request to target %s", target)
	return client.Poll()
}

func splitSubscribeRequest(sctx *subContext, req *gnmi.SubscribeRequest) error {
	sctx.treqs = make(map[string]*gnmi.SubscribeRequest)

	subs := req.GetSubscribe()
	prefixTarget := subs.Prefix.Target // fallback target for a single-target request

	// If the prefix names a target, it is assumed this is a single-target request and the original request
	// becomes the request for that target.
	if prefixTarget != "" {
		sctx.treqs[prefixTarget] = req
		return nil
	}

	// Otherwise, iterate over the subscriptions and separate them into multiple subscription requests
	// based on the target specified in each path using the original request as a template.
	for _, sub := range subs.Subscription {
		target := sub.Path.Target
		var tr *gnmi.SubscribeRequest
		if target != "" {
			ok := false
			if tr, ok = sctx.treqs[target]; !ok {
				tr = &gnmi.SubscribeRequest{
					Request: &gnmi.SubscribeRequest_Subscribe{
						Subscribe: &gnmi.SubscriptionList{
							Prefix:           copyPrefix(subs.Prefix, target),
							Subscription:     make([]*gnmi.Subscription, 0, 1),
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
				sctx.treqs[target] = tr
			}
			tr.GetSubscribe().Subscription = append(tr.GetSubscribe().Subscription, sub)
		}
	}

	if len(sctx.treqs) == 0 {
		return errors.NewInvalid("Prefix or at least one path must specify a target")
	} else if prefixTarget != "" {
		return errors.NewInvalid("Prefix not supported for multi-target request")
	}
	return nil
}

func copyPrefix(prefix *gnmi.Path, target string) *gnmi.Path {
	return &gnmi.Path{
		Origin: prefix.Origin,
		Elem:   prefix.Elem,
		Target: target,
	}
}
