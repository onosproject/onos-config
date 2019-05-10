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
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/listener"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"log"
)

var mgr Manager

// Manager single point of entry for the config system
type Manager struct {
	ConfigStore    *store.ConfigurationStore
	ChangeStore    *store.ChangeStore
	deviceStore    *topocache.DeviceStore
	NetworkStore   *store.NetworkStore
	topoChannel    chan events.Event
	changesChannel chan events.Event
}

// NewManager initializes the network config manager subsystem.
func NewManager(configs *store.ConfigurationStore, changes *store.ChangeStore, device *topocache.DeviceStore,
	network *store.NetworkStore, topoCh chan events.Event) *Manager {
	log.Println("Creating Manager")
	mgr = Manager{
		ConfigStore:    configs,
		ChangeStore:    changes,
		deviceStore:    device,
		NetworkStore:   network,
		topoChannel:    topoCh,
		changesChannel: make(chan events.Event, 10),
	}
	return &mgr

}

// Run starts a synchronizer based on the devices and the northbound services
func (m *Manager) Run() {
	log.Println("Starting Manager")
	// Start the main listener system
	go listener.Listen(m.changesChannel)
	go synchronizer.Factory(m.ChangeStore, m.deviceStore, m.topoChannel)
}

//Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Println("Closing Manager")
	close(m.changesChannel)
}

// GetManager retuns the initialized and running instance of manager.
// Should be called only after NewManager and Run are done.
func GetManager() *Manager {
	return &mgr
}
