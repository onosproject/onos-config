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

// Package diags implements the diagnostic gRPC service for the configuration subsystem.
package diags

import (
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/util/metautils"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	nbutils "github.com/onosproject/onos-config/pkg/northbound/utils"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"sync"

	"github.com/onosproject/onos-api/go/onos/config/admin"
	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	networkchange "github.com/onosproject/onos-api/go/onos/config/change/network"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	"github.com/onosproject/onos-api/go/onos/config/diags"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	streams "github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound", "diags")

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
	deviceChangesStore        device.Store
	deviceCache               cache.Cache
	networkChangesStore       network.Store
	dispatcher                *dispatcher.Dispatcher
	deviceStore               devicestore.Store
	operationalStateCache     *map[topodevice.ID]devicechange.TypedValueMap
	operationalStateCacheLock *sync.RWMutex
}

// Server implements the gRPC service for diagnostic facilities.
type Server struct {
	deviceChangesStore        device.Store
	networkChangesStore       network.Store
	deviceCache               cache.Cache
	dispatcher                *dispatcher.Dispatcher
	deviceStore               devicestore.Store
	operationalStateCache     *map[topodevice.ID]devicechange.TypedValueMap
	operationalStateCacheLock *sync.RWMutex
}

// NewService allocates a Service struct with the given parameters
func NewService(deviceChangesStore device.Store,
	deviceCache cache.Cache,
	networkChangesStore network.Store,
	dispatcher *dispatcher.Dispatcher,
	deviceStore devicestore.Store,
	operationalStateCache *map[topodevice.ID]devicechange.TypedValueMap,
	operationalStateCacheLock *sync.RWMutex) Service {
	return Service{
		deviceChangesStore:        deviceChangesStore,
		deviceCache:               deviceCache,
		networkChangesStore:       networkChangesStore,
		dispatcher:                dispatcher,
		deviceStore:               deviceStore,
		operationalStateCache:     operationalStateCache,
		operationalStateCacheLock: operationalStateCacheLock,
	}
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	server := &Server{
		deviceChangesStore:        s.deviceChangesStore,
		deviceCache:               s.deviceCache,
		networkChangesStore:       s.networkChangesStore,
		dispatcher:                s.dispatcher,
		deviceStore:               s.deviceStore,
		operationalStateCache:     s.operationalStateCache,
		operationalStateCacheLock: s.operationalStateCacheLock,
	}
	diags.RegisterOpStateDiagsServer(r, server)
	diags.RegisterChangeServiceServer(r, server)
}

// GetOpState provides a stream of Operational and State data
func (s Server) GetOpState(r *diags.OpStateRequest, stream diags.OpStateDiags_GetOpStateServer) error {
	if stream.Context() != nil {
		if md := metautils.ExtractIncoming(stream.Context()); md != nil && md.Get("name") != "" {
			log.Infof("diags GetOpState() called by '%s (%s)' with token %s",
				md.Get("name"), md.Get("email"), md.Get("at_hash"))
		}
	}

	s.operationalStateCacheLock.RLock()
	deviceCache, ok := (*s.operationalStateCache)[topodevice.ID(r.DeviceId)]
	s.operationalStateCacheLock.RUnlock()
	if !ok {
		return fmt.Errorf("no Operational State cache available for %s", r.DeviceId)
	}

	for path, value := range deviceCache {
		pathValue := &devicechange.PathValue{
			Path:  path,
			Value: value,
		}

		msg := &diags.OpStateResponse{Type: admin.Type_NONE, Pathvalue: pathValue}
		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}

	if r.Subscribe {
		streamID := fmt.Sprintf("diags-%p", stream)
		listener, err := s.dispatcher.RegisterOpState(streamID)
		defer s.dispatcher.UnregisterOperationalState(streamID)
		if err != nil {
			log.Warnf("Failed setting up a listener for OpState events on %s", r.DeviceId)
			return err
		}
		log.Infof("NBI Diags OpState started on %s for %s", streamID, r.DeviceId)
		for {
			select {
			case opStateEvent := <-listener:
				if opStateEvent.Subject() != r.DeviceId {
					// If the event is not for this device then ignore it
					continue
				}
				log.Infof("Event received NBI Diags OpState subscribe channel %s for %s",
					streamID, r.DeviceId)

				pathValue := &devicechange.PathValue{
					Path:  opStateEvent.Path(),
					Value: opStateEvent.Value(),
				}

				msg := &diags.OpStateResponse{Type: admin.Type_ADDED, Pathvalue: pathValue}
				err = stream.SendMsg(msg)
				if err != nil {
					log.Warnf("Error sending message on stream %s. Closing. %v",
						streamID, msg)
					return err
				}
			case <-stream.Context().Done():
				log.Infof("NBI Diags OpState subscribe channel %s for %s closed",
					streamID, r.DeviceId)
				return nil
			}
		}
	}

	log.Infof("Closing NBI Diags OpState stream (no subscribe) for %s", r.DeviceId)
	return nil
}

// ListNetworkChanges provides a stream of Network Changes
// If the optional `subscribe` flag is true, then get then return the list of
// changes first, and then hold the connection open and send on
// further updates until the client hangs up
func (s Server) ListNetworkChanges(r *diags.ListNetworkChangeRequest, stream diags.ChangeService_ListNetworkChangesServer) error {
	if stream.Context() != nil {
		if md := metautils.ExtractIncoming(stream.Context()); md != nil && md.Get("name") != "" {
			log.Infof("diags ListNetworkChanges() called by '%s (%s)' with token %s",
				md.Get("name"), md.Get("email"), md.Get("at_hash"))
		}
	}
	log.Infof("ListNetworkChanges called with %s. Subscribe %v", r.ChangeID, r.Subscribe)

	// There may be a wildcard given - we only want to reply with changes that match
	matcher := utils.MatchWildcardChNameRegexp(string(r.ChangeID), false)
	var watchOpts []network.WatchOption
	if !r.WithoutReplay {
		watchOpts = append(watchOpts, network.WithReplay())
	}

	if r.Subscribe {
		eventCh := make(chan streams.Event)
		ctx, err := s.networkChangesStore.Watch(eventCh, watchOpts...)
		if err != nil {
			log.Errorf("Error watching Network Changes %s", err)
			return err
		}
		defer ctx.Close()

		for {
			breakout := false
			select { // Blocks until one of the following are received
			case event, ok := <-eventCh:
				if !ok { // Will happen at the end of stream
					breakout = true
					break
				}

				change := event.Object.(*networkchange.NetworkChange)

				if matcher.MatchString(string(change.ID)) {
					msg := &diags.ListNetworkChangeResponse{
						Change: change,
						Type:   streamTypeToResponseType(event.Type),
					}
					log.Infof("Sending matching change %v", change.ID)
					err := stream.Send(msg)
					if err != nil {
						log.Errorf("Error sending NetworkChanges %v %v", change.ID, err)
						return err
					}
				}
			case <-stream.Context().Done():
				log.Infof("ListNetworkChanges remote client closed connection")
				return nil
			}
			if breakout {
				break
			}
		}
	} else {
		changeCh := make(chan *networkchange.NetworkChange)
		ctx, err := s.networkChangesStore.List(changeCh)
		if err != nil {
			log.Errorf("Error listing Network Changes %s", err)
			return err
		}
		defer ctx.Close()

		for {
			breakout := false
			select { // Blocks until one of the following are received
			case change, ok := <-changeCh:
				if !ok { // Will happen at the end of stream
					breakout = true
					break
				}

				if matcher.MatchString(string(change.ID)) {
					msg := &diags.ListNetworkChangeResponse{
						Change: change,
						Type:   diags.Type_NONE,
					}
					log.Infof("Sending matching change %v", change.ID)
					err := stream.Send(msg)
					if err != nil {
						log.Errorf("Error sending NetworkChanges %v %v", change.ID, err)
						return err
					}
				}
			case <-stream.Context().Done():
				log.Infof("ListNetworkChanges remote client closed connection")
				return nil
			}
			if breakout {
				break
			}
		}
	}
	log.Infof("Closing ListNetworkChanges for %s", r.ChangeID)
	return nil
}

// ListDeviceChanges provides a stream of Device Changes
func (s Server) ListDeviceChanges(r *diags.ListDeviceChangeRequest, stream diags.ChangeService_ListDeviceChangesServer) error {
	if stream.Context() != nil {
		if md := metautils.ExtractIncoming(stream.Context()); md != nil && md.Get("name") != "" {
			log.Infof("diags ListDeviceChanges() called by '%s (%s)' with token %s",
				md.Get("name"), md.Get("email"), md.Get("at_hash"))
		}
	}
	log.Infof("ListDeviceChanges called with %s %s. Subscribe %v", r.DeviceID, r.DeviceVersion, r.Subscribe)

	var watchOpts []device.WatchOption
	if !r.WithoutReplay {
		watchOpts = append(watchOpts, device.WithReplay())
	}

	// Look in the deviceCache for a version
	_, version, err := nbutils.CheckCacheForDevice(r.DeviceID, "", r.DeviceVersion, s.deviceCache, s.deviceStore)
	if err != nil {
		log.Errorf(err.Error())
		return err
	}

	if r.Subscribe {
		eventCh := make(chan streams.Event)
		ctx, err := s.deviceChangesStore.Watch(devicetype.NewVersionedID(r.DeviceID, version), eventCh, watchOpts...)
		if err != nil {
			log.Errorf("Error watching Network Changes %s", err)
			return err
		}
		defer ctx.Close()

		for {
			breakout := false
			select { // Blocks until one of the following are received
			case event, ok := <-eventCh:
				if !ok { // Will happen at the end of stream
					breakout = true
					break
				}

				change := event.Object.(*devicechange.DeviceChange)
				msg := &diags.ListDeviceChangeResponse{
					Change: change,
					Type:   streamTypeToResponseType(event.Type),
				}
				log.Infof("Sending matching change %v", change.ID)
				err := stream.Send(msg)
				if err != nil {
					log.Errorf("Error sending NetworkChanges %v %v", change.ID, err)
					return err
				}
			case <-stream.Context().Done():
				log.Infof("ListDeviceChanges Remote client closed connection")
				return nil
			}
			if breakout {
				break
			}
		}
	} else {
		changeCh := make(chan *devicechange.DeviceChange)
		ctx, err := s.deviceChangesStore.List(devicetype.NewVersionedID(r.DeviceID, version), changeCh)
		if err != nil {
			log.Errorf("Error listing Network Changes %s", err)
			return err
		}
		defer ctx.Close()

		for {
			breakout := false
			select { // Blocks until one of the following are received
			case change, ok := <-changeCh:
				if !ok { // Will happen at the end of stream
					breakout = true
					break
				}

				msg := &diags.ListDeviceChangeResponse{
					Change: change,
					Type:   diags.Type_NONE,
				}
				log.Infof("Sending matching change %v", change.ID)
				err := stream.Send(msg)
				if err != nil {
					log.Errorf("Error sending NetworkChanges %v %v", change.ID, err)
					return err
				}
			case <-stream.Context().Done():
				log.Infof("ListDeviceChanges remote client closed connection")
				return nil
			}
			if breakout {
				break
			}
		}
	}
	log.Infof("Closing ListDeviceChanges for %s", r.DeviceID)
	return nil
}

func streamTypeToResponseType(eventType streams.EventType) diags.Type {
	switch eventType {
	case streams.Created:
		return diags.Type_ADDED
	case streams.Updated:
		return diags.Type_UPDATED
	case streams.Deleted:
		return diags.Type_REMOVED
	default:
		return diags.Type_NONE
	}
}
