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
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	log "k8s.io/klog"
)

// DeviceConnected signals the corresponding topology service that the device connected.
func (m *Manager) DeviceConnected(id devicetopo.ID) (*devicetopo.Device, error) {
	log.Infof("Device %s connected", id)
	return updateDevice(m.DeviceStore, id, devicetopo.ConnectivityState_REACHABLE, devicetopo.ChannelState_CONNECTED,
		devicetopo.ServiceState_AVAILABLE)
}

// DeviceDisconnected signal the corresponding topology service that the device disconnected.
func (m *Manager) DeviceDisconnected(id devicetopo.ID, err error) (*devicetopo.Device, error) {
	log.Infof("Device %s disconnected or had error in connection %s", id, err)
	//TODO check different possible availabilities based on error
	return updateDevice(m.DeviceStore, id, devicetopo.ConnectivityState_UNREACHABLE, devicetopo.ChannelState_DISCONNECTED,
		devicetopo.ServiceState_UNAVAILABLE)
}

func updateDevice(deviceStore device.Store, id devicetopo.ID, connectivity devicetopo.ConnectivityState, channel devicetopo.ChannelState,
	service devicetopo.ServiceState) (*devicetopo.Device, error) {
	topoDevice, err := deviceStore.Get(id)
	if err != nil {
		return nil, err
	}
	protocolState, index := containsGnmi(topoDevice.Protocols)
	if protocolState != nil {
		topoDevice.Protocols = remove(topoDevice.Protocols, index)
	} else {
		protocolState = new(devicetopo.ProtocolState)
	}
	protocolState.Protocol = devicetopo.Protocol_GNMI
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

func containsGnmi(protocols []*devicetopo.ProtocolState) (*devicetopo.ProtocolState, int) {
	for i, p := range protocols {
		if p.Protocol == devicetopo.Protocol_GNMI {
			return p, i
		}
	}
	return nil, -1
}

func remove(s []*devicetopo.ProtocolState, i int) []*devicetopo.ProtocolState {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
