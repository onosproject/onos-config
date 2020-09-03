// Copyright 2020-present Open Networking Foundation.
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

package synchronizer

import (
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cenkalti/backoff"

	"github.com/onosproject/onos-config/pkg/events"

	topodevice "github.com/onosproject/onos-topo/api/device"
)

func (sm *SessionManager) updateDevice(id topodevice.ID, connectivity topodevice.ConnectivityState, channel topodevice.ChannelState,
	service topodevice.ServiceState) error {
	log.Info("Update device state")

	topoDevice, err := sm.deviceStore.Get(id)
	st, ok := status.FromError(err)

	// If the device doesn't exist then we should not update its state
	if ok && err != nil && st.Code() == codes.NotFound {
		return nil
	}

	if err != nil {
		return err
	}

	protocolState, index := containsGnmi(topoDevice.Protocols)
	if protocolState != nil {
		topoDevice.Protocols = remove(topoDevice.Protocols, index)
	} else {
		protocolState = new(topodevice.ProtocolState)
	}

	protocolState.Protocol = topodevice.Protocol_GNMI
	protocolState.ConnectivityState = connectivity
	protocolState.ChannelState = channel
	protocolState.ServiceState = service
	topoDevice.Protocols = append(topoDevice.Protocols, protocolState)
	mastershipState, err := sm.mastershipStore.GetMastership(topoDevice.ID)
	if err != nil {
		return err
	}

	if topoDevice.Attributes == nil {
		topoDevice.Attributes = make(map[string]string)
	}

	topoDevice.Attributes[mastershipTermKey] = strconv.FormatUint(uint64(mastershipState.Term), 10)
	_, err = sm.deviceStore.Update(topoDevice)
	if err != nil {
		log.Errorf("Device %s is not updated %s", id, err.Error())
		return err
	}
	log.Infof("Device %s is updated with states %s, %s, %s", id, connectivity, channel, service)
	return nil
}

func containsGnmi(protocols []*topodevice.ProtocolState) (*topodevice.ProtocolState, int) {
	for i, p := range protocols {
		if p.Protocol == topodevice.Protocol_GNMI {
			return p, i
		}
	}
	return nil, -1
}

func remove(s []*topodevice.ProtocolState, i int) []*topodevice.ProtocolState {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func (sm *SessionManager) updateDeviceState() error {
	for event := range sm.southboundErrorChan {
		log.Info("update event received")
		switch event.EventType() {
		case events.EventTypeDeviceConnected:
			log.Info("Device connected")
			id := topodevice.ID(event.Subject())
			// TODO: Retry only on write conflicts
			return backoff.Retry(func() error {
				/* This entire read-modify-write sequence has to be retried until one of two conditions is met:
				   - The update is successful or
				   - Upon retry, the node encounters a mastership term greater than its own */

				currentTerm, err := sm.sessions[id].getCurrentTerm()
				if err != nil {
					return nil
				}
				mastershipState, err := sm.mastershipStore.GetMastership(id)
				if err != nil {
					return nil
				}
				if uint64(mastershipState.Term) < uint64(currentTerm) {
					return nil
				}
				err = sm.updateDevice(id, topodevice.ConnectivityState_REACHABLE, topodevice.ChannelState_CONNECTED,
					topodevice.ServiceState_AVAILABLE)
				return err

			}, backoff.NewExponentialBackOff())
		case events.EventTypeErrorDeviceConnect:
			id := topodevice.ID(event.Subject())
			// TODO: Retry only on write conflicts
			return backoff.Retry(func() error {

				currentTerm, err := sm.sessions[id].getCurrentTerm()
				if err != nil {
					return nil
				}
				mastershipState, err := sm.mastershipStore.GetMastership(id)
				if err != nil {
					return nil
				}
				if uint64(mastershipState.Term) < uint64(currentTerm) {
					return nil
				}

				err = sm.updateDevice(id, topodevice.ConnectivityState_UNREACHABLE, topodevice.ChannelState_DISCONNECTED,
					topodevice.ServiceState_UNAVAILABLE)
				return err
			}, backoff.NewExponentialBackOff())

		default:
		}
	}

	return nil
}
