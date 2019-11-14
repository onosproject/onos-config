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
	"github.com/onosproject/onos-config/api/admin"
	"github.com/onosproject/onos-config/api/diags"
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	networkchange "github.com/onosproject/onos-config/api/types/change/network"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	streams "github.com/onosproject/onos-config/pkg/store/stream"
	"github.com/onosproject/onos-config/pkg/utils"
	"github.com/onosproject/onos-config/pkg/utils/logging"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound", "diags")

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	diags.RegisterOpStateDiagsServer(r, Server{})
	diags.RegisterChangeServiceServer(r, Server{})
}

// Server implements the gRPC service for diagnostic facilities.
type Server struct {
}

// GetOpState provides a stream of Operational and State data
func (s Server) GetOpState(r *diags.OpStateRequest, stream diags.OpStateDiags_GetOpStateServer) error {
	manager.GetManager().OperationalStateCacheLock.RLock()
	deviceCache, ok := manager.GetManager().OperationalStateCache[topodevice.ID(r.DeviceId)]
	manager.GetManager().OperationalStateCacheLock.RUnlock()
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
		listener, err := manager.GetManager().Dispatcher.RegisterOpState(streamID)
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
				manager.GetManager().Dispatcher.UnregisterOperationalState(streamID)
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
	log.Infof("ListNetworkChanges called with %s. Subscribe %v", r.ChangeID, r.Subscribe)

	// There may be a wildcard given - we only want to reply with changes that match
	matcher := utils.MatchWildcardChNameRegexp(string(r.ChangeID))
	var watchOpts []network.WatchOption
	if !r.WithoutReplay {
		watchOpts = append(watchOpts, network.WithReplay())
	}

	if r.Subscribe {
		eventCh := make(chan streams.Event)
		ctx, err := manager.GetManager().NetworkChangesStore.Watch(eventCh, watchOpts...)
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
		ctx, err := manager.GetManager().NetworkChangesStore.List(changeCh)
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
	log.Infof("ListDeviceChanges called with %s %s. Subscribe %v", r.DeviceID, r.DeviceVersion, r.Subscribe)

	var watchOpts []device.WatchOption
	if !r.WithoutReplay {
		watchOpts = append(watchOpts, device.WithReplay())
	}

	// Look in the deviceCache for a version
	_, version, err := manager.GetManager().CheckCacheForDevice(r.DeviceID, "", r.DeviceVersion)
	if err != nil {
		log.Errorf(err.Error())
		return err
	}

	if r.Subscribe {
		eventCh := make(chan streams.Event)
		ctx, err := manager.GetManager().DeviceChangesStore.Watch(devicetype.NewVersionedID(r.DeviceID, version), eventCh, watchOpts...)
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
		ctx, err := manager.GetManager().DeviceChangesStore.List(devicetype.NewVersionedID(r.DeviceID, r.DeviceVersion), changeCh)
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
