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
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	devicetopo "github.com/onosproject/onos-config/pkg/store/device"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	devicepb "github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	log "k8s.io/klog"
	"strings"
	"time"
)

var mgr Manager

// Manager single point of entry for the config system.
type Manager struct {
	ConfigStore             *store.ConfigurationStore
	ChangeStore             *store.ChangeStore
	DeviceChangesStore      device.Store
	DeviceStore             devicetopo.Store
	NetworkStore            *store.NetworkStore
	NetworkChangesStore     network.Store
	ModelRegistry           *modelregistry.ModelRegistry
	TopoChannel             chan *devicepb.ListResponse
	ChangesChannel          chan events.ConfigEvent
	OperationalStateChannel chan events.OperationalStateEvent
	SouthboundErrorChan     chan events.DeviceResponse
	Dispatcher              *dispatcher.Dispatcher
	OperationalStateCache   map[devicepb.ID]devicechangetypes.TypedValueMap
}

// NewManager initializes the network config manager subsystem.
func NewManager(configStore *store.ConfigurationStore, deviceChangesStore device.Store, changeStore *store.ChangeStore, deviceStore devicetopo.Store,
	networkStore *store.NetworkStore, networkChangesStore network.Store, topoCh chan *devicepb.ListResponse) (*Manager, error) {
	log.Info("Creating Manager")
	modelReg := &modelregistry.ModelRegistry{
		ModelPlugins:        make(map[string]modelregistry.ModelPlugin),
		ModelReadOnlyPaths:  make(map[string]modelregistry.ReadOnlyPathMap),
		ModelReadWritePaths: make(map[string]modelregistry.ReadWritePathMap),
		LocationStore:       make(map[string]string),
	}

	mgr = Manager{
		//TODO remove deprecated ConfigStore
		ConfigStore:        configStore,
		DeviceChangesStore: deviceChangesStore,
		//TODO remove deprecated ChangeStore
		ChangeStore: changeStore,
		DeviceStore: deviceStore,
		//TODO remove deprecated NetworkStore
		NetworkStore:            networkStore,
		NetworkChangesStore:     networkChangesStore,
		TopoChannel:             topoCh,
		ModelRegistry:           modelReg,
		ChangesChannel:          make(chan events.ConfigEvent, 10),
		OperationalStateChannel: make(chan events.OperationalStateEvent, 10),
		SouthboundErrorChan:     make(chan events.DeviceResponse, 10),
		Dispatcher:              dispatcher.NewDispatcher(),
		OperationalStateCache:   make(map[devicepb.ID]devicechangetypes.TypedValueMap),
	}

	changeIds := make([]string, 0)
	// Perform a sanity check on the change store
	for changeID, changeObj := range changeStore.Store {
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

	for configID, configObj := range configStore.Store {
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
func LoadManager(configStoreFile string, changeStoreFile string, networkStoreFile string,
	deviceChangesStore device.Store, networkChangesStore network.Store, opts ...grpc.DialOption) (*Manager, error) {
	topoChannel := make(chan *devicepb.ListResponse, 10)

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

	deviceStore, err := devicetopo.NewTopoStore(opts...)
	if err != nil {
		log.Error("Cannot load device store ", err)
		return nil, err
	}
	if deviceStore != nil {
		var err error
		go func() {
			err = deviceStore.Watch(topoChannel)
			if err != nil {
				log.Error("Cannot Watch devices", err)
			}
		}()
		log.Info("Device store loaded")
	}
	networkStore, err := store.LoadNetworkStore(networkStoreFile)
	if err != nil {
		log.Error("Cannot load network store ", err)
		return nil, err
	}
	log.Info("Network store loaded from ", networkStoreFile)

	return NewManager(&configStore, deviceChangesStore, &changeStore, deviceStore, networkStore, networkChangesStore, topoChannel)
}

// ValidateStores validate configurations against their ModelPlugins at startup
func (m *Manager) ValidateStores() error {
	validationErrors := make(chan error)
	cfgCount := len(m.ConfigStore.Store)
	if cfgCount > 0 {
		for _, configObj := range m.ConfigStore.Store {
			go validateConfiguration(configObj, m.ChangeStore.Store, validationErrors)
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
func validateConfiguration(configObj store.Configuration, changeStore map[string]*change.Change, errChan chan error) {
	modelPluginName := configObj.Type + "-" + configObj.Version
	cfgModelPlugin, pluginExists := mgr.ModelRegistry.ModelPlugins[modelPluginName]
	if pluginExists {
		log.Info("Validating config ", configObj.Name, " with Model Plugin ", modelPluginName)
		fullconfig := configObj.ExtractFullConfig(nil, mgr.ChangeStore.Store, 0)
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

		// Also validate that configs do not contain any read only paths
		for _, changeID := range configObj.Changes {
			change, ok := changeStore[store.B64(changeID)]
			if !ok {
				err = fmt.Errorf("error retrieving paths for %s %s", store.B64(changeID), modelPluginName)
				errChan <- err
				return
			}
			for _, path := range change.Config {
				changePath := modelregistry.RemovePathIndices(path.Path)
				roPathsAndValues, ok := mgr.ModelRegistry.ModelReadOnlyPaths[modelPluginName]
				if !ok {
					log.Warningf("Can't find read only paths for %s", modelPluginName)
					continue
				}
				roPaths := modelregistry.Paths(roPathsAndValues)
				for _, ropath := range roPaths {
					if strings.HasPrefix(changePath, ropath) {
						err = fmt.Errorf("error read only path in configuration %s matches %s for %s",
							changePath, ropath, modelPluginName)
						errChan <- err
						return
					}
				}
			}
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
	go listenOnResponseChannel(m.SouthboundErrorChan, m)
	//TODO we need to find a way to avoid passing down parameter but at the same time not hve circular dependecy sb-mgr
	go synchronizer.Factory(m.ChangeStore, m.ConfigStore, m.TopoChannel,
		m.OperationalStateChannel, m.SouthboundErrorChan, m.Dispatcher, m.ModelRegistry, m.OperationalStateCache)
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

func listenOnResponseChannel(respChan chan events.DeviceResponse, m *Manager) {
	log.Info("Listening for Errors in Manager")
	for event := range respChan {
		subject := devicepb.ID(event.Subject())
		switch event.EventType() {
		case events.EventTypeDeviceConnected:
			_, err := m.DeviceConnected(subject)
			if err != nil {
				log.Error("Can't notify connection", err)
			}
			//TODO unblock config
		case events.EventTypeErrorDeviceConnect:
			_, err := m.DeviceDisconnected(subject, event.Error())
			if err != nil {
				log.Error("Can't notify disconnection", err)
			}
			//TODO unblock config
		default:
			if strings.Contains(event.Error().Error(), "desc =") {
				log.Errorf("Error reported to channel %s",
					strings.Split(event.Error().Error(), " desc = ")[1])
			} else {
				log.Error("Response reported to channel ", event.Error().Error())
			}
		}
	}
}

func (m *Manager) computeChange(updates devicechangetypes.TypedValueMap,
	deletes []string, description string) (*change.Change, error) {
	var newChanges = make([]*devicechangetypes.ChangeValue, 0)
	//updates
	for path, value := range updates {
		changeValue, err := devicechangetypes.NewChangeValue(path, value, false)
		if err != nil {
			log.Warningf("Error creating value for %s %v", path, err)
			continue
		}
		newChanges = append(newChanges, changeValue)
	}
	//deletes
	for _, path := range deletes {
		changeValue, _ := devicechangetypes.NewChangeValue(path, devicechangetypes.NewTypedValueEmpty(), true)
		newChanges = append(newChanges, changeValue)
	}
	if description == "" {
		description = fmt.Sprintf("Created at %s", time.Now().Format(time.RFC3339))
	}
	return change.NewChange(newChanges, description)
}

func (m *Manager) computeDeviceChange(ID devicepb.ID, deviceVersion string, updates devicechangetypes.TypedValueMap,
	deletes []string, description string) (*devicechangetypes.Change, error) {
	var newChanges = make([]*devicechangetypes.ChangeValue, 0)
	//updates
	for path, value := range updates {
		changeValue, err := devicechangetypes.NewChangeValue(path, value, false)
		if err != nil {
			log.Warningf("Error creating value for %s %v", path, err)
			continue
		}
		newChanges = append(newChanges, changeValue)
	}
	//deletes
	for _, path := range deletes {
		changeValue, _ := devicechangetypes.NewChangeValue(path, devicechangetypes.NewTypedValueEmpty(), true)
		newChanges = append(newChanges, changeValue)
	}
	//if description == "" {
	//	description = fmt.Sprintf("Created at %s", time.Now().Format(time.RFC3339))
	//}
	//TODO lost description of Change
	changeElement := &devicechangetypes.Change{
		DeviceID:      ID,
		DeviceVersion: deviceVersion,
		Values:        newChanges,
	}

	return changeElement, nil
}

func (m *Manager) storeChange(configChange *change.Change) (change.ID, error) {
	if m.ChangeStore.Store[store.B64(configChange.ID)] != nil {
		log.Info("Change ID = ", store.B64(configChange.ID), " already exists - not overwriting")
	} else {
		m.ChangeStore.Store[store.B64(configChange.ID)] = configChange
		log.Info("Added change ", store.B64(configChange.ID), " to ChangeStore (in memory)")
	}
	return configChange.ID, nil
}
