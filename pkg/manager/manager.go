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
	"github.com/onosproject/onos-config/pkg/listener"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	"log"
	"strings"
)

var mgr Manager

// Manager single point of entry for the config system
type Manager struct {
	ConfigStore    *store.ConfigurationStore
	ChangeStore    *store.ChangeStore
	DeviceStore    *topocache.DeviceStore
	NetworkStore   *store.NetworkStore
	TopoChannel    chan events.TopoEvent
	ChangesChannel chan events.ConfigEvent
}

// NewManager initializes the network config manager subsystem.
func NewManager(configs *store.ConfigurationStore, changes *store.ChangeStore, device *topocache.DeviceStore,
	network *store.NetworkStore, topoCh chan events.TopoEvent) (*Manager, error) {
	log.Println("Creating Manager")
	mgr = Manager{
		ConfigStore:    configs,
		ChangeStore:    changes,
		DeviceStore:    device,
		NetworkStore:   network,
		TopoChannel:    topoCh,
		ChangesChannel: make(chan events.ConfigEvent, 10),
	}

	changeIds := make([]string, 0)
	// Perform a sanity check on the change store
	for changeID, changeObj := range changes.Store {
		err := changeObj.IsValid()
		if err != nil {
			return nil, err
		}
		if changeID != store.B64(changeObj.ID) {
			return nil, fmt.Errorf("Change Id: %s must match %s",
				changeID, store.B64(changeObj.ID))
		}
		changeIds = append(changeIds, changeID)
	}

	changeIdsStr := strings.Join(changeIds, ",")

	for configID, configObj := range configs.Store {
		for _, chID := range configObj.Changes {
			if !strings.Contains(changeIdsStr, store.B64(chID)) {
				return nil, fmt.Errorf(
					"ChangeID %s from Config %s not found in change store",
					store.B64(chID), configID)
			}
		}

	}

	return &mgr, nil

}

// Run starts a synchronizer based on the devices and the northbound services
func (m *Manager) Run() {
	log.Println("Starting Manager")
	// Start the main listener system
	go listener.Listen(m.ChangesChannel)
	go synchronizer.Factory(m.ChangeStore, m.DeviceStore, m.TopoChannel)
}

//Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Println("Closing Manager")
	close(m.ChangesChannel)
}

// GetManager returns the initialized and running instance of manager.
// Should be called only after NewManager and Run are done.
func GetManager() *Manager {
	return &mgr
}
