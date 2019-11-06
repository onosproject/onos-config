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
	"github.com/onosproject/onos-config/pkg/store/device"
	topodevice "github.com/onosproject/onos-topo/api/device"
	log "k8s.io/klog"
)

// DeviceConnected signals the corresponding topology service that the device connected.
func (m *Manager) DeviceConnected(id topodevice.ID) (*topodevice.Device, error) {
	log.Infof("Device %s connected", id)
	return updateDevice(m.DeviceStore, id, topodevice.ConnectivityState_REACHABLE, topodevice.ChannelState_CONNECTED,
		topodevice.ServiceState_AVAILABLE)
}

// DeviceDisconnected signal the corresponding topology service that the device disconnected.
func (m *Manager) DeviceDisconnected(id topodevice.ID, err error) (*topodevice.Device, error) {
	log.Infof("Device %s disconnected or had error in connection %s", id, err)
	//TODO check different possible availabilities based on error
	return updateDevice(m.DeviceStore, id, topodevice.ConnectivityState_UNREACHABLE, topodevice.ChannelState_DISCONNECTED,
		topodevice.ServiceState_UNAVAILABLE)
}

func updateDevice(deviceStore device.Store, id topodevice.ID, connectivity topodevice.ConnectivityState, channel topodevice.ChannelState,
	service topodevice.ServiceState) (*topodevice.Device, error) {
	topoDevice, err := deviceStore.Get(id)
	if err != nil {
		return nil, err
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
	updatedDevice, err := deviceStore.Update(topoDevice)
	if err != nil {
		log.Errorf("Device %s is not updated %s", id, err.Error())
		return nil, err
	}
	log.Infof("Device %s is updated with states %s, %s, %s", id, connectivity, channel, service)
	return updatedDevice, nil
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
