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

package manager

import (
	"github.com/cenkalti/backoff"
	"github.com/onosproject/onos-config/pkg/store/device"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"github.com/onosproject/onos-topo/api/topo"
)

// DeviceConnected signals the corresponding topology service that the device connected.
func (m *Manager) DeviceConnected(id topodevice.ID) error {
	log.Infof("Device %s connected", id)
	// TODO: Retry only on write conflicts
	return backoff.Retry(func() error {
		return updateDevice(m.DeviceStore, id, topodevice.ConnectivityState_REACHABLE, topodevice.ChannelState_CONNECTED,
			topodevice.ServiceState_AVAILABLE)
	}, backoff.NewExponentialBackOff())
}

// DeviceDisconnected signal the corresponding topology service that the device disconnected.
func (m *Manager) DeviceDisconnected(id topodevice.ID, err error) error {
	log.Infof("Device %s disconnected or had error in connection %s", id, err)
	// TODO: Retry only on write conflicts
	return backoff.Retry(func() error {
		return updateDevice(m.DeviceStore, id, topodevice.ConnectivityState_UNREACHABLE, topodevice.ChannelState_DISCONNECTED,
			topodevice.ServiceState_UNAVAILABLE)
	}, backoff.NewExponentialBackOff())
}

func updateDevice(deviceStore device.Store, id topodevice.ID, connectivity topodevice.ConnectivityState, channel topodevice.ChannelState, service topodevice.ServiceState) error {
	object, err := deviceStore.Get(topo.ID(id))
	if err != nil {
		return err
	}
	topoDevice := topo.ObjectToDevice(object)
	protocolState, index := containsGnmi(topoDevice.Protocols)
	if protocolState != nil {
		topoDevice.Protocols = remove(topoDevice.Protocols, index)
	} else {
		protocolState = new(topodevice.ProtocolState)
	}
	if protocolState.ConnectivityState == connectivity && protocolState.ChannelState == channel && protocolState.ServiceState == service {
		return nil
	}
	protocolState.Protocol = topodevice.Protocol_GNMI
	protocolState.ConnectivityState = connectivity
	protocolState.ChannelState = channel
	protocolState.ServiceState = service
	topoDevice.Protocols = append(topoDevice.Protocols, protocolState)
	object1 := topo.DeviceToObject(topoDevice)
	_, err = deviceStore.Update(object1)
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
