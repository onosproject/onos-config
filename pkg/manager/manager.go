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

// Package manager is is the main coordinator for the ONOS configuration subsystem.
package manager

import (
	"fmt"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/southbound/topocache"
	"github.com/onosproject/onos-config/pkg/store"
	log "k8s.io/klog"
	"strings"
)

var mgr Manager

// Manager single point of entry for the config system.
type Manager struct {
	ConfigStore             *store.ConfigurationStore
	ChangeStore             *store.ChangeStore
	DeviceStore             *topocache.DeviceStore
	NetworkStore            *store.NetworkStore
	TopoChannel             chan events.TopoEvent
	ChangesChannel          chan events.ConfigEvent
	OperationalStateChannel chan events.OperationalStateEvent
	Dispatcher              dispatcher.Dispatcher
	ModelRegistry           map[string]ModelPlugin
}

// NewManager initializes the network config manager subsystem.
func NewManager(configs *store.ConfigurationStore, changes *store.ChangeStore, device *topocache.DeviceStore,
	network *store.NetworkStore, topoCh chan events.TopoEvent) (*Manager, error) {
	log.Info("Creating Manager")
	mgr = Manager{
		ConfigStore:             configs,
		ChangeStore:             changes,
		DeviceStore:             device,
		NetworkStore:            network,
		TopoChannel:             topoCh,
		ChangesChannel:          make(chan events.ConfigEvent, 10),
		OperationalStateChannel: make(chan events.OperationalStateEvent, 10),
		Dispatcher:              dispatcher.NewDispatcher(),
		ModelRegistry:           make(map[string]ModelPlugin),
	}

	changeIds := make([]string, 0)
	// Perform a sanity check on the change store
	for changeID, changeObj := range changes.Store {
		err := changeObj.IsValid()
		if err != nil {
			return nil, err
		}
		if changeID != store.B64(changeObj.ID) {
			return nil, fmt.Errorf("ChangeID: %s must match %s",
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

// LoadManager creates a configuration subsystem manager primed with stores loaded from the specified files.
func LoadManager(configStoreFile string, changeStoreFile string, deviceStoreFile string, networkStoreFile string) (*Manager, error) {
	topoChannel := make(chan events.TopoEvent, 10)

	configStore, err := store.LoadConfigStore(configStoreFile)
	if err != nil {
		log.Error("Cannot load config store ", err)
		return nil, err
	}
	log.Info("Configuration store loaded from", configStoreFile)

	changeStore, err := store.LoadChangeStore(changeStoreFile)
	if err != nil {
		log.Error("Cannot load change store ", err)
		return nil, err
	}
	log.Info("Change store loaded from", changeStoreFile)

	deviceStore, err := topocache.LoadDeviceStore(deviceStoreFile, topoChannel)
	if err != nil {
		log.Error("Cannot load device store ", err)
		return nil, err
	}
	log.Info("Device store loaded from ", deviceStoreFile)

	networkStore, err := store.LoadNetworkStore(networkStoreFile)
	if err != nil {
		log.Error("Cannot load network store ", err)
		return nil, err
	}
	log.Info("Network store loaded from ", networkStoreFile)

	return NewManager(&configStore, &changeStore, deviceStore, networkStore, topoChannel)
}

// Run starts a synchronizer based on the devices and the northbound services.
func (m *Manager) Run() {
	log.Info("Starting Manager")
	// Start the main dispatcher system
	go m.Dispatcher.Listen(m.ChangesChannel)
	go m.Dispatcher.ListenOperationalState(m.OperationalStateChannel)
	go synchronizer.Factory(m.ChangeStore, m.DeviceStore, m.TopoChannel, m.OperationalStateChannel, &m.Dispatcher)
}

//Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Info("Closing Manager")
	close(m.ChangesChannel)
	close(m.OperationalStateChannel)
}

// GetManager returns the initialized and running instance of manager.
// Should be called only after NewManager and Run are done.
func GetManager() *Manager {
	return &mgr
}
