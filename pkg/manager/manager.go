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
	"github.com/onosproject/onos-config/pkg/store/change"
	log "k8s.io/klog"
	"strings"
	"time"
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
	SouthboundErrorChan     chan events.DeviceResponse
	Dispatcher              dispatcher.Dispatcher
	ModelRegistry           map[string]ModelPlugin
	ModelReadOnlyPaths      map[string][]string
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
		SouthboundErrorChan:     make(chan events.DeviceResponse, 10),
		Dispatcher:              dispatcher.NewDispatcher(),
		ModelRegistry:           make(map[string]ModelPlugin),
		ModelReadOnlyPaths:      make(map[string][]string),
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
	log.Info("Configuration store loaded from ", configStoreFile)

	changeStore, err := store.LoadChangeStore(changeStoreFile)
	if err != nil {
		log.Error("Cannot load change store ", err)
		return nil, err
	}
	log.Info("Change store loaded from ", changeStoreFile)

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

// ValidateStores validate configurations against their ModelPlugins at startup
func (m *Manager) ValidateStores() error {
	validationErrors := make(chan error)
	cfgCount := len(m.ConfigStore.Store)
	if cfgCount > 0 {
		for _, configObj := range m.ConfigStore.Store {
			go validateConfiguration(configObj, validationErrors)
		}
		for valErr := range validationErrors {
			if valErr != nil {
				return valErr
			}
			cfgCount--
			if cfgCount == 0 {
				return nil
			}
		}
	}
	return nil
}

// validateConfiguration is a go routine for validating a Configuration against it ModelPlugin
func validateConfiguration(configObj store.Configuration, errChan chan error) {
	modelPluginName := configObj.Type + "-" + configObj.Version
	cfgModelPlugin, pluginExists := mgr.ModelRegistry[modelPluginName]
	if pluginExists {
		log.Info("Validating config ", configObj.Name, " with Model Plugin ", modelPluginName)
		fullconfig := configObj.ExtractFullConfig(mgr.ChangeStore.Store, 0)
		configJSON, err := store.BuildTree(fullconfig, true)
		if err != nil {
			errChan <- err
			return
		}
		ygotModelConfig, err := cfgModelPlugin.UnmarshalConfigValues(configJSON)
		if err != nil {
			log.Error(string(configJSON))
			errChan <- err
			return
		}
		err = cfgModelPlugin.Validate(ygotModelConfig)
		if err != nil {
			log.Error(string(configJSON))
			errChan <- err
			return
		}
	} else {
		log.Warning("No Model Plugin available for ", modelPluginName)
	}
	errChan <- nil
}

// Run starts a synchronizer based on the devices and the northbound services.
func (m *Manager) Run() {
	log.Info("Starting Manager")
	// Start the main dispatcher system
	go m.Dispatcher.Listen(m.ChangesChannel)
	go m.Dispatcher.ListenOperationalState(m.OperationalStateChannel)
	// Listening for errors in the Southbound
	go listenOnResponseChannel(m.SouthboundErrorChan)
	go synchronizer.Factory(m.ChangeStore, m.ConfigStore, m.DeviceStore, m.TopoChannel,
		m.OperationalStateChannel, m.SouthboundErrorChan, &m.Dispatcher)
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

func listenOnResponseChannel(respChan chan events.DeviceResponse) {
	log.Info("Listening for Errors in Manager")
	for err := range respChan {
		if strings.Contains(err.Error().Error(), "desc =") {
			log.Errorf("Error reported to channel %s",
				strings.Split(err.Error().Error(), " desc = ")[1])
		} else {
			log.Error("Response reported to channel ", err.Error().Error())
		}
		//TODO handle device connection errors accordingly
	}
}

func (m *Manager) computeAndStoreChange(updates map[string]*change.TypedValue,
	deletes []string) (change.ID, error) {
	var newChanges = make([]*change.Value, 0)
	//updates
	for path, value := range updates {
		changeValue, _ := change.CreateChangeValue(path, value, false)
		newChanges = append(newChanges, changeValue)
	}
	//deletes
	for _, path := range deletes {
		changeValue, _ := change.CreateChangeValue(path, change.CreateTypedValueEmpty(), true)
		newChanges = append(newChanges, changeValue)
	}
	configChange, err := change.CreateChange(newChanges,
		fmt.Sprintf("Created at %s", time.Now().Format(time.RFC3339)))
	if err != nil {
		return nil, err
	}
	if m.ChangeStore.Store[store.B64(configChange.ID)] != nil {
		log.Info("Change ID = ", store.B64(configChange.ID), " already exists - not overwriting")
	} else {
		m.ChangeStore.Store[store.B64(configChange.ID)] = configChange
		log.Info("Added change ", store.B64(configChange.ID), " to ChangeStore (in memory)")
	}
	return configChange.ID, nil
}
