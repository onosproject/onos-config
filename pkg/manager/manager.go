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
	devicechange "github.com/onosproject/onos-config/api/types/change/device"
	devicetype "github.com/onosproject/onos-config/api/types/device"
	"github.com/onosproject/onos-config/pkg/controller"
	devicechangectl "github.com/onosproject/onos-config/pkg/controller/change/device"
	networkchangectl "github.com/onosproject/onos-config/pkg/controller/change/network"
	devicesnapshotctl "github.com/onosproject/onos-config/pkg/controller/snapshot/device"
	networksnapshotctl "github.com/onosproject/onos-config/pkg/controller/snapshot/network"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	devicesnap "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnap "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	log "k8s.io/klog"
	"strings"
)

var mgr Manager

// Manager single point of entry for the config system.
type Manager struct {
	LeadershipStore           leadership.Store
	MastershipStore           mastership.Store
	DeviceChangesStore        device.Store
	DeviceStore               devicestore.Store
	DeviceCache               devicestore.Cache
	NetworkChangesStore       network.Store
	NetworkSnapshotStore      networksnap.Store
	DeviceSnapshotStore       devicesnap.Store
	networkChangeController   *controller.Controller
	deviceChangeController    *controller.Controller
	networkSnapshotController *controller.Controller
	deviceSnapshotController  *controller.Controller
	ModelRegistry             *modelregistry.ModelRegistry
	TopoChannel               chan *devicetopo.ListResponse
	OperationalStateChannel   chan events.OperationalStateEvent
	SouthboundErrorChan       chan events.DeviceResponse
	Dispatcher                *dispatcher.Dispatcher
	OperationalStateCache     map[devicetopo.ID]devicechange.TypedValueMap
}

// NewManager initializes the network config manager subsystem.
func NewManager(leadershipStore leadership.Store, mastershipStore mastership.Store,
	deviceChangesStore device.Store, deviceStore devicestore.Store, deviceCache devicestore.Cache,
	networkChangesStore network.Store, networkSnapshotStore networksnap.Store,
	deviceSnapshotStore devicesnap.Store, topoCh chan *devicetopo.ListResponse) (*Manager, error) {
	log.Info("Creating Manager")
	modelReg := &modelregistry.ModelRegistry{
		ModelPlugins:        make(map[string]modelregistry.ModelPlugin),
		ModelReadOnlyPaths:  make(map[string]modelregistry.ReadOnlyPathMap),
		ModelReadWritePaths: make(map[string]modelregistry.ReadWritePathMap),
		LocationStore:       make(map[string]string),
	}

	mgr = Manager{
		DeviceChangesStore:        deviceChangesStore,
		DeviceStore:               deviceStore,
		DeviceCache:               deviceCache,
		NetworkChangesStore:       networkChangesStore,
		NetworkSnapshotStore:      networkSnapshotStore,
		DeviceSnapshotStore:       deviceSnapshotStore,
		networkChangeController:   networkchangectl.NewController(leadershipStore, deviceStore, networkChangesStore, deviceChangesStore),
		deviceChangeController:    devicechangectl.NewController(mastershipStore, deviceStore, deviceChangesStore),
		networkSnapshotController: networksnapshotctl.NewController(leadershipStore, networkChangesStore, networkSnapshotStore, deviceSnapshotStore),
		deviceSnapshotController:  devicesnapshotctl.NewController(mastershipStore, deviceChangesStore, deviceSnapshotStore),
		TopoChannel:               topoCh,
		ModelRegistry:             modelReg,
		OperationalStateChannel:   make(chan events.OperationalStateEvent, 10),
		SouthboundErrorChan:       make(chan events.DeviceResponse, 10),
		Dispatcher:                dispatcher.NewDispatcher(),
		OperationalStateCache:     make(map[devicetopo.ID]devicechange.TypedValueMap),
	}
	return &mgr, nil
}

// LoadManager creates a configuration subsystem manager primed with stores loaded from the specified files.
func LoadManager(leadershipStore leadership.Store, mastershipStore mastership.Store, deviceChangesStore device.Store,
	deviceCache devicestore.Cache, networkChangesStore network.Store,
	networkSnapshotStore networksnap.Store, deviceSnapshotStore devicesnap.Store, opts ...grpc.DialOption) (*Manager, error) {
	topoChannel := make(chan *devicetopo.ListResponse, 10)

	deviceStore, err := devicestore.NewTopoStore(opts...)
	if err != nil {
		log.Error("Cannot load device store ", err)
		return nil, err
	}

	return NewManager(leadershipStore, mastershipStore, deviceChangesStore, deviceStore, deviceCache,
		networkChangesStore, networkSnapshotStore, deviceSnapshotStore, topoChannel)
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
	go m.Dispatcher.ListenOperationalState(m.OperationalStateChannel)
	// Listening for errors in the Southbound
	go listenOnResponseChannel(m.SouthboundErrorChan, m)
	//TODO we need to find a way to avoid passing down parameter but at the same time not hve circular dependecy sb-mgr
	go synchronizer.Factory(m.TopoChannel, m.OperationalStateChannel, m.SouthboundErrorChan,
		m.Dispatcher, m.ModelRegistry, m.OperationalStateCache)

	err := m.DeviceStore.Watch(m.TopoChannel)
	if err != nil {
		log.Error("Error Watching devices", err)
	}
	log.Info("Device store watch started")
}

//Close kills the channels and manager related objects
func (m *Manager) Close() {
	log.Info("Closing Manager")
	close(m.TopoChannel)
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
		case events.EventTypeErrorDeviceConnect:
			_, err := m.DeviceDisconnected(subject, event.Error())
			if err != nil {
				log.Error("Can't notify disconnection", err)
			}
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

// ComputeDeviceChange computes a given device change the given updates and deletes, according to the path
// on the configuration for the specified target
func (m *Manager) ComputeDeviceChange(deviceName devicetype.ID, version devicetype.Version,
	deviceType devicetype.Type, updates devicechange.TypedValueMap,
	deletes []string, description string) (*devicechange.Change, error) {

	var newChanges = make([]*devicechange.ChangeValue, 0)
	//updates
	for path, value := range updates {
		updateValue, err := devicechange.NewChangeValue(path, value, false)
		if err != nil {
			log.Warningf("Error creating value for %s %v", path, err)
			continue
		}
		newChanges = append(newChanges, updateValue)
	}
	//deletes
	for _, path := range deletes {
		deleteValue, _ := devicechange.NewChangeValue(path, devicechange.NewTypedValueEmpty(), true)
		newChanges = append(newChanges, deleteValue)
	}
	//description := fmt.Sprintf("Originally created as part of %s", description)
	//if description == "" {
	//	description = fmt.Sprintf("Created at %s", time.Now().Format(time.RFC3339))
	//}
	//TODO lost description of Change
	changeElement := &devicechange.Change{
		DeviceID:      deviceName,
		DeviceVersion: version,
		DeviceType:    deviceType,
		Values:        newChanges,
	}

	return changeElement, nil
}

// CheckCacheForDevice checks against the device cache (of the device change store
// to see if a device of that name is already present)
func (m *Manager) CheckCacheForDevice(target devicetype.ID, deviceType devicetype.Type,
	version devicetype.Version) (devicetype.Type, devicetype.Version, error) {

	deviceInfos := mgr.DeviceCache.GetDevicesByID(target)
	topoDevice, errTopoDevice := mgr.DeviceStore.Get(devicetopo.ID(target))
	if errTopoDevice != nil {
		log.Infof("Device %s not found in topo store", target)
	}

	if len(deviceInfos) == 0 {
		// New device - need type and version
		if deviceType == "" || version == "" {
			if errTopoDevice == nil && topoDevice != nil {
				return devicetype.Type(topoDevice.Type), devicetype.Version(topoDevice.Version), nil
			}
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
