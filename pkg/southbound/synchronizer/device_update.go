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
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cenkalti/backoff"

	"github.com/onosproject/onos-config/pkg/events"

	"github.com/onosproject/onos-api/go/onos/topo"
)

func (s *Session) updateDevice(connectivity topo.ConnectivityState, channel topo.ChannelState,
	service topo.ServiceState) error {
	log.Infof("Update device %s state", s.device.ID)

	id := s.device.ID
	topoDevice, err := s.deviceStore.Get(id)
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
		protocolState = new(topo.ProtocolState)
	}

	protocolState.Protocol = topo.Protocol_GNMI
	protocolState.ConnectivityState = connectivity
	protocolState.ChannelState = channel
	protocolState.ServiceState = service
	topoDevice.Protocols = append(topoDevice.Protocols, protocolState)

	// Read the current term for the given device
	currentTerm := topoDevice.MastershipTerm

	// Do not update the state of a device if the node encounters a mastership term greater than its own
	if uint64(s.mastershipState.Term) < currentTerm {
		return backoff.Permanent(errors.NewInvalid("device mastership term is greater than node mastership term"))
	}

	topoDevice.MastershipTerm = uint64(s.mastershipState.Term)
	topoDevice.MasterKey = string(s.mastershipState.Master)
	_, err = s.deviceStore.Update(topoDevice)
	if err != nil {
		log.Errorf("Device %s is not updated %s", id, err.Error())
		return err
	}
	log.Infof("Device %s is updated with states %s, %s, %s", id, connectivity, channel, service)
	return nil
}

func containsGnmi(protocols []*topo.ProtocolState) (*topo.ProtocolState, int) {
	for i, p := range protocols {
		if p.Protocol == topo.Protocol_GNMI {
			return p, i
		}
	}
	return nil, -1
}

func remove(s []*topo.ProtocolState, i int) []*topo.ProtocolState {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func (s *Session) updateConnectedDevice() error {
	err := s.updateDevice(topo.ConnectivityState_REACHABLE, topo.ChannelState_CONNECTED,
		topo.ServiceState_AVAILABLE)
	return err
}

func (s *Session) updateDisconnectedDevice() error {
	err := s.updateDevice(topo.ConnectivityState_UNREACHABLE, topo.ChannelState_DISCONNECTED,
		topo.ServiceState_UNAVAILABLE)
	return err

}

// updateDeviceState updates device state based on a device response event
func (s *Session) updateDeviceState() error {
	for event := range s.deviceResponseChan {
		switch event.EventType() {
		case events.EventTypeDeviceConnected:
			// TODO: Retry only on write conflicts
			_ = backoff.Retry(s.updateConnectedDevice, backoff.NewExponentialBackOff())
		case events.EventTypeErrorDeviceConnect:
			// TODO: Retry only on write conflicts
			_ = backoff.Retry(s.updateDisconnectedDevice, backoff.NewExponentialBackOff())

		default:

		}
	}

	return nil
}
