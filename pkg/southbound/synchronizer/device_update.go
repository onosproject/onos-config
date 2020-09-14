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
	"errors"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cenkalti/backoff"

	"github.com/onosproject/onos-config/pkg/events"

	topodevice "github.com/onosproject/onos-topo/api/device"
)

func (s *Session) getTermPerDevice(device *topodevice.Device) (int, error) {
	term := device.Attributes[mastershipTermKey]
	if term == "" {
		return 0, nil
	}
	return strconv.Atoi(term)

}

func (s *Session) updateDevice(connectivity topodevice.ConnectivityState, channel topodevice.ChannelState,
	service topodevice.ServiceState) error {
	log.Info("Update device %s state", s.device.ID)

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
		protocolState = new(topodevice.ProtocolState)
	}

	protocolState.Protocol = topodevice.Protocol_GNMI
	protocolState.ConnectivityState = connectivity
	protocolState.ChannelState = channel
	protocolState.ServiceState = service
	topoDevice.Protocols = append(topoDevice.Protocols, protocolState)

	// Read the current term for the given device
	currentTerm, err := s.getTermPerDevice(topoDevice)
	if err != nil {
		return err
	}

	// Do not update the state of a device if the node encounters a mastership term greater than its own
	if uint64(s.mastershipState.Term) < uint64(currentTerm) {
		return backoff.Permanent(errors.New("device mastership term is greater than node mastership term"))
	}

	if topoDevice.Attributes == nil {
		topoDevice.Attributes = make(map[string]string)
	}

	topoDevice.Attributes[mastershipTermKey] = strconv.FormatUint(uint64(s.mastershipState.Term), 10)
	topoDevice.Attributes[mastershipMasterKey] = string(s.mastershipState.Master)
	_, err = s.deviceStore.Update(topoDevice)
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

func (s *Session) updateConnectedDevice() error {
	err := s.updateDevice(topodevice.ConnectivityState_REACHABLE, topodevice.ChannelState_CONNECTED,
		topodevice.ServiceState_AVAILABLE)
	return err
}

func (s *Session) updateDisconnectedDevice() error {
	err := s.updateDevice(topodevice.ConnectivityState_UNREACHABLE, topodevice.ChannelState_DISCONNECTED,
		topodevice.ServiceState_UNAVAILABLE)
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
