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
	"fmt"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"log"
)

var mgr Manager

// Manager single point of entry for the config system
type Manager struct {
	configStore *store.ConfigurationStore
	changeStore *store.ChangeStore
	deviceStore *topocache.DeviceStore
	networkStore *store.NetworkStore
	topoChannel chan events.Event
}

// NewManager initializes the network config manager subsystem.
func NewManager(configs *store.ConfigurationStore, changes *store.ChangeStore, device *topocache.DeviceStore,
	network *store.NetworkStore, topoCh chan events.Event) *Manager {
	log.Println("Creating Manager")
	mgr = Manager{
		configStore : configs,
		changeStore : changes,
		deviceStore : device,
		networkStore : network,
		topoChannel: topoCh,
	}
	return &mgr

}
// Run starts a synchronizer based on the devices and the northbound services
func (m *Manager) Run() {
	log.Println("Starting Manager")
	go synchronizer.Factory(m.changeStore, m.deviceStore, m.topoChannel)
}

// GetNetworkConfig returns a set of change values given a target, a configuration name, a path and a layer.
// The layer is the numbers of configu changes we want to go back in time for.
func (m *Manager) GetNetworkConfig(target string, configname string, path string, layer int) ([]change.ConfigValue, error){
	if _, ok := m.deviceStore.Store[target]; !ok {
		return nil, fmt.Errorf("Device not present %s", target)
	}
	//TODO the key of the config store shoudl be a tuple of (devicename, configname) use the param
	var config store.Configuration
	for configID, cfg := range m.configStore.Store{
		if cfg.Device == target {
			configname = configID
			config = cfg
			break
		}
	}
	configValues := config.ExtractFullConfig(m.changeStore.Store, layer)
	return configValues, nil
}

// GetManager retuns the initialized and running instance of manager.
// Should be called only after NewManager and Run are done.
func GetManager() *Manager {
	return &mgr
}
