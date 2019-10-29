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
	"github.com/onosproject/onos-config/pkg/controller"
	devicechange "github.com/onosproject/onos-config/pkg/controller/change/device"
	networkchange "github.com/onosproject/onos-config/pkg/controller/change/network"
	devicesnapshot "github.com/onosproject/onos-config/pkg/controller/snapshot/device"
	networksnapshot "github.com/onosproject/onos-config/pkg/controller/snapshot/network"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/store"
	"github.com/onosproject/onos-config/pkg/store/change"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	devicesnap "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnap "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	devicechangetypes "github.com/onosproject/onos-config/pkg/types/change/device"
	devicetype "github.com/onosproject/onos-config/pkg/types/device"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"strings"
	"time"
)

var mgr Manager

// Manager single point of entry for the config system.
type Manager struct {
	// deprecated: old store
	ConfigStore *store.ConfigurationStore
	// deprecated: old store
	ChangeStore        *store.ChangeStore
	LeadershipStore    leadership.Store
	MastershipStore    mastership.Store
	DeviceChangesStore device.Store
	DeviceStore        devicestore.Store
	DeviceCache        devicestore.Cache
	// deprecated: old store
	NetworkStore              *store.NetworkStore
	NetworkChangesStore       network.Store
	NetworkSnapshotStore      networksnap.Store
	DeviceSnapshotStore       devicesnap.Store
	networkChangeController   *controller.Controller
	deviceChangeController    *controller.Controller
	networkSnapshotController *controller.Controller
	deviceSnapshotController  *controller.Controller
	ModelRegistry             *modelregistry.ModelRegistry
	TopoChannel               chan *devicetopo.ListResponse
	ChangesChannel            chan events.ConfigEvent
	OperationalStateChannel   chan events.OperationalStateEvent
	SouthboundErrorChan       chan events.DeviceResponse
	Dispatcher                *dispatcher.Dispatcher
	OperationalStateCache     map[devicetopo.ID]devicechangetypes.TypedValueMap
}

// NewManager initializes the network config manager subsystem.
func NewManager(configStore *store.ConfigurationStore, leadershipStore leadership.Store, mastershipStore mastership.Store,
	deviceChangesStore device.Store, changeStore *store.ChangeStore, deviceStore devicestore.Store, deviceCache devicestore.Cache,
	networkStore *store.NetworkStore, networkChangesStore network.Store, networkSnapshotStore networksnap.Store,
	deviceSnapshotStore devicesnap.Store, topoCh chan *devicetopo.ListResponse) (*Manager, error) {
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
		DeviceCache: deviceCache,
		//TODO remove deprecated NetworkStore
		NetworkStore:              networkStore,
		NetworkChangesStore:       networkChangesStore,
		NetworkSnapshotStore:      networkSnapshotStore,
		DeviceSnapshotStore:       deviceSnapshotStore,
		networkChangeController:   networkchange.NewController(leadershipStore, deviceStore, networkChangesStore, deviceChangesStore),
		deviceChangeController:    devicechange.NewController(mastershipStore, deviceStore, deviceChangesStore),
		networkSnapshotController: networksnapshot.NewController(leadershipStore, networkChangesStore, networkSnapshotStore, deviceSnapshotStore),
		deviceSnapshotController:  devicesnapshot.NewController(mastershipStore, deviceChangesStore, deviceSnapshotStore),
		TopoChannel:               topoCh,
		ModelRegistry:             modelReg,
		ChangesChannel:            make(chan events.ConfigEvent, 10),
		OperationalStateChannel:   make(chan events.OperationalStateEvent, 10),
		SouthboundErrorChan:       make(chan events.DeviceResponse, 10),
		Dispatcher:                dispatcher.NewDispatcher(),
		OperationalStateCache:     make(map[devicetopo.ID]devicechangetypes.TypedValueMap),
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
func LoadManager(configStoreFile string, changeStoreFile string, networkStoreFile string, leadershipStore leadership.Store,
	mastershipStore mastership.Store, deviceChangesStore device.Store,
	deviceCache devicestore.Cache, networkChangesStore network.Store,
	networkSnapshotStore networksnap.Store, deviceSnapshotStore devicesnap.Store, opts ...grpc.DialOption) (*Manager, error) {
	topoChannel := make(chan *devicetopo.ListResponse, 10)

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

	deviceStore, err := devicestore.NewTopoStore(opts...)
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

	return NewManager(&configStore, leadershipStore, mastershipStore, deviceChangesStore, &changeStore, deviceStore, deviceCache, networkStore, networkChangesStore, networkSnapshotStore, deviceSnapshotStore, topoChannel)
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

	// Start the NetworkChange controller
	errNetworkCtrl := m.networkChangeController.Start()
	if errNetworkCtrl != nil {
		log.Error("Can't start controller ", errNetworkCtrl)
	}
	// Start the DeviceChange controller
	errDeviceChangeCtrl := m.deviceChangeController.Start()
	if errDeviceChangeCtrl != nil {
		log.Error("Can't start controller ", errDeviceChangeCtrl)
	}
	// Start the NetworkSnapshot controller
	errNetworkSnapshotCtrl := m.networkSnapshotController.Start()
	if errNetworkSnapshotCtrl != nil {
		log.Error("Can't start controller ", errNetworkSnapshotCtrl)
	}
	// Start the DeviceSnapshot controller
	errDeviceSnapshotCtrl := m.deviceSnapshotController.Start()
	if errDeviceSnapshotCtrl != nil {
		log.Error("Can't start controller ", errDeviceSnapshotCtrl)
	}

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
		subject := devicetopo.ID(event.Subject())
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

// Deprecated: computeChange works on legacy, non-atomix stores
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

// ComputeNewDeviceChange computes a given device change the given updates and deletes, according to the path
// on the configuration for the specified target
func (m *Manager) ComputeNewDeviceChange(deviceName devicetype.ID, version devicetype.Version,
	deviceType devicetype.Type, updates devicechangetypes.TypedValueMap,
	deletes []string, description string) (*devicechangetypes.Change, error) {

	var newChanges = make([]*devicechangetypes.ChangeValue, 0)
	//updates
	for path, value := range updates {
		updateValue, err := devicechangetypes.NewChangeValue(path, value, false)
		if err != nil {
			log.Warningf("Error creating value for %s %v", path, err)
			continue
		}
		newChanges = append(newChanges, updateValue)
	}
	//deletes
	for _, path := range deletes {
		deleteValue, _ := devicechangetypes.NewChangeValue(path, devicechangetypes.NewTypedValueEmpty(), true)
		newChanges = append(newChanges, deleteValue)
	}
	//description := fmt.Sprintf("Originally created as part of %s", description)
	//if description == "" {
	//	description = fmt.Sprintf("Created at %s", time.Now().Format(time.RFC3339))
	//}
	//TODO lost description of Change
	changeElement := &devicechangetypes.Change{
		DeviceID:      deviceName,
		DeviceVersion: version,
		DeviceType:    deviceType,
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

// CheckCacheForDevice checks against the device cache (of the device change store
// to see if a device of that name is already present)
func (m *Manager) CheckCacheForDevice(target devicetype.ID, deviceType devicetype.Type,
	version devicetype.Version) (devicetype.Type, devicetype.Version, error) {

	deviceInfos := mgr.DeviceCache.GetDevicesByID(target)
	if len(deviceInfos) == 0 {
		// New device - need type and version
		if deviceType == "" || version == "" {
			return "", "", status.Error(codes.Internal,
				fmt.Sprintf("target %s is not known. Need to supply a type and version through Extensions 101 and 102", target))
		}
		return deviceType, version, nil
	} else if len(deviceInfos) == 1 {
		log.Infof("Handling target %s as %s:%s", target, deviceType, version)
		if deviceInfos[0].Version != version {
			log.Infof("Ignoring device type %s and version %s from extension for %s. Using %s and %s",
				deviceType, version, target, deviceInfos[0].Type, deviceInfos[0].Version)
		}
		return deviceInfos[0].Type, deviceInfos[0].Version, nil
	} else {
		// n devices of that name already exist - have to choose 1 or exit
		for _, di := range deviceInfos {
			if di.Version == version {
				log.Infof("Handling target %s as %s:%s", target, deviceType, version)
				return di.Type, di.Version, nil
			}
		}
		// Else allow it as a new version
		if deviceType == deviceInfos[0].Type && version != "" {
			log.Infof("Handling target %s as %s:%s", target, deviceType, version)
			return deviceType, version, nil
		} else if deviceType != "" && deviceType != deviceInfos[0].Type {
			return "", "", status.Error(codes.Internal,
				fmt.Sprintf("target %s type given %s does not match expected %s",
					target, deviceType, deviceInfos[0].Type))
		}

		return "", "", status.Error(codes.Internal,
			fmt.Sprintf("target %s has %d versions. Specify 1 version with extension 102",
				target, len(deviceInfos)))
	}
}

// ExtractTypeAndVersion gets a deviceType and a Version based on the available store, topo and extension info
// deprecated: now that we use device cache directly
func (m *Manager) ExtractTypeAndVersion(target devicetopo.ID, storedDevice *devicetopo.Device,
	versionExt string, deviceTypeExt string) (devicestore.Info, error) {

	var (
		deviceType string
		version    string
	)

	if storedDevice == nil && (deviceTypeExt == "" || versionExt == "") {
		devices := m.DeviceCache.GetDevicesByID(devicetype.ID(target))
		if len(devices) == 0 {
			return devicestore.Info{}, fmt.Errorf("device %s is not connected and Extensions were not "+
				"complete given %s, %s", target, versionExt, deviceTypeExt)
		} else if len(devices) > 1 {
			return devicestore.Info{}, fmt.Errorf("device %s is ambiguous and Extensions were not "+
				"complete given %s, %s", target, versionExt, deviceTypeExt)
		}
		return devicestore.Info{
			Type:    devices[0].Type,
			Version: devices[0].Version,
		}, nil
	}

	if storedDevice == nil && deviceTypeExt != "" && versionExt != "" {
		return devicestore.Info{Version: devicetype.Version(versionExt), Type: devicetype.Type(deviceTypeExt)}, nil
	}

	if storedDevice != nil && deviceTypeExt == "" {
		deviceType = string(storedDevice.Type)
	}
	if storedDevice != nil && versionExt == "" {
		version = storedDevice.Version
	}

	return devicestore.Info{Version: devicetype.Version(version), Type: devicetype.Type(deviceType)}, nil
}
