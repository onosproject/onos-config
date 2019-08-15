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
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/manager"
	"github.com/onosproject/onos-config/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	log "k8s.io/klog"
)

// Service is a Service implementation for administration.
type Service struct {
	northbound.Service
}

// Register registers the Service with the gRPC server.
func (s Service) Register(r *grpc.Server) {
	RegisterConfigDiagsServer(r, Server{})
	RegisterOpStateDiagsServer(r, Server{})
}

// Server implements the gRPC service for diagnostic facilities.
type Server struct {
}

// GetChanges provides a stream of submitted network changes.
func (s Server) GetChanges(r *ChangesRequest, stream ConfigDiags_GetChangesServer) error {
	for _, c := range manager.GetManager().ChangeStore.Store {
		if len(r.ChangeIDs) > 0 && !stringInList(r.ChangeIDs, store.B64(c.ID)) {
			continue
		}

		errInvalid := c.IsValid()
		if errInvalid != nil {
			return errInvalid
		}

		changeValues := make([]*admin.ChangeValue, 0)

		for _, cv := range c.Config {
			to := make([]int32, len(cv.TypeOpts))
			for i, t := range cv.TypeOpts {
				to[i] = int32(t)
			}
			changeValues = append(changeValues, &admin.ChangeValue{
				Path:      cv.Path,
				Value:     cv.Value,
				ValueType: admin.ChangeValueType(cv.Type),
				TypeOpts:  to,
				Removed:   cv.Remove,
			})
		}

		// Build a change message
		msg := &admin.Change{
			Time:         &c.Created,
			Id:           store.B64(c.ID),
			Desc:         c.Description,
			ChangeValues: changeValues,
		}
		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetConfigurations provides a stream of submitted network changes.
func (s Server) GetConfigurations(r *ConfigRequest, stream ConfigDiags_GetConfigurationsServer) error {
	for _, c := range manager.GetManager().ConfigStore.Store {
		if len(r.DeviceIDs) > 0 && !stringInList(r.DeviceIDs, c.Device) {
			continue
		}

		changeIDs := make([]string, len(c.Changes))

		for idx, cid := range c.Changes {
			changeIDs[idx] = store.B64(cid)
		}

		msg := &Configuration{
			Name:       string(c.Name),
			DeviceID:   c.Device,
			Version:    c.Version,
			DeviceType: c.Type,
			Updated:    &c.Updated,
			ChangeIDs:  changeIDs,
		}
		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetOpState provides a stream of Operational and State data
func (s Server) GetOpState(r *OpStateRequest, stream OpStateDiags_GetOpStateServer) error {
	deviceCache, ok := manager.GetManager().OperationalStateCache[device.ID(r.DeviceId)]
	if !ok {
		return fmt.Errorf("no Operational State cache available for %s", r.DeviceId)
	}

	for path, value := range deviceCache {
		typeOpts := make([]int32, len(value.TypeOpts))
		for i, v := range value.TypeOpts {
			typeOpts[i] = int32(v)
		}

		pathValue := &admin.ChangeValue{
			Path:      path,
			ValueType: admin.ChangeValueType(value.Type),
			Value:     value.Value,
			TypeOpts:  typeOpts,
		}

		msg := &OpStateResponse{Type: admin.Type_NONE, Pathvalue: pathValue}
		err := stream.Send(msg)
		if err != nil {
			return err
		}
	}

	if r.Subscribe {
		streamID := fmt.Sprintf("diags-%p", stream)
		listener, err := manager.GetManager().Dispatcher.RegisterOpState(streamID)
		if err != nil {
			log.Warning("Failed setting up a listener for OpState events on ", r.DeviceId)
			return err
		}
		log.Infof("NBI Diags OpState started on %s for %s", streamID, r.DeviceId)
		for {
			select {
			case opStateEvent := <-listener:
				event := events.Event(opStateEvent)
				log.Infof("Event received NBI Diags OpState subscribe channel %s for %s",
					streamID, r.DeviceId)
				if event.Subject() != r.DeviceId {
					// If the event is not for this device then ignore it
					continue
				}
				// At the moment the event is capable of holding many paths
				// send one message each
				for k, v := range *event.Values() {
					// Do the best we can - the event only holds a string version of the value at the moment
					pathValue := &admin.ChangeValue{
						Path:      k,
						ValueType: admin.ChangeValueType(change.ValueTypeSTRING),
						Value:     []byte(v),
						TypeOpts:  []int32{},
					}

					msg := &OpStateResponse{Type: admin.Type_ADDED, Pathvalue: pathValue}
					err = stream.SendMsg(msg)
					if err != nil {
						log.Warningf("Error sending message on stream %s. Closing. %v",
							streamID, msg)
						return err
					}
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

func stringInList(haystack []string, needle string) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}
	return false
}
