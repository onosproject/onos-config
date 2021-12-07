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

// Package manager is the main coordinator for the ONOS configuration subsystem.
package manager

import (
	"fmt"
	"github.com/atomix/atomix-go-client/pkg/atomix"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-lib-go/pkg/certs"

	"os"
	"sync"

	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
	devicetype "github.com/onosproject/onos-api/go/onos/config/device"
	devicechangectl "github.com/onosproject/onos-config/pkg/controller/change/device"
	networkchangectl "github.com/onosproject/onos-config/pkg/controller/change/network"
	devicesnapshotctl "github.com/onosproject/onos-config/pkg/controller/snapshot/device"
	networksnapshotctl "github.com/onosproject/onos-config/pkg/controller/snapshot/network"
	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/dispatcher"
	"github.com/onosproject/onos-config/pkg/events"
	"github.com/onosproject/onos-config/pkg/modelregistry"
	"github.com/onosproject/onos-config/pkg/southbound"
	"github.com/onosproject/onos-config/pkg/store/change/device"
	"github.com/onosproject/onos-config/pkg/store/change/device/state"
	"github.com/onosproject/onos-config/pkg/store/change/network"
	devicestore "github.com/onosproject/onos-config/pkg/store/device"
	"github.com/onosproject/onos-config/pkg/store/device/cache"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	devicesnap "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	networksnap "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var mgr Manager

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath                 string
	KeyPath                string
	CertPath               string
	GRPCPort               int
	TopoAddress            string
	AllowUnvalidatedConfig bool
	ModelRegistry          *modelregistry.ModelRegistry
}

// Manager single point of entry for the config system.
type Manager struct {
	Config                    Config
	LeadershipStore           leadership.Store
	MastershipStore           mastership.Store
	DeviceChangesStore        device.Store
	DeviceStateStore          state.Store
	DeviceStore               devicestore.Store
	DeviceCache               cache.Cache
	NetworkChangesStore       network.Store
	NetworkSnapshotStore      networksnap.Store
	DeviceSnapshotStore       devicesnap.Store
	networkChangeController   *controller.Controller
	deviceChangeController    *controller.Controller
	networkSnapshotController *controller.Controller
	deviceSnapshotController  *controller.Controller
	ModelRegistry             *modelregistry.ModelRegistry
	TopoChannel               chan *topodevice.ListResponse
	OperationalStateChannel   chan events.OperationalStateEvent
	SouthboundErrorChan       chan events.DeviceResponse
	Dispatcher                *dispatcher.Dispatcher
	OperationalStateCache     map[topodevice.ID]devicechange.TypedValueMap
	OperationalStateCacheLock *sync.RWMutex
}

// NewManager initializes the network config manager subsystem.
func NewManager(cfg Config) *Manager {
	log.Info("Creating Manager")

	mgr = Manager{
		Config: cfg,
	}
	return &mgr
}

// setTargetGenerator is generally only called from test
func (m *Manager) setTargetGenerator(targetGen func() southbound.TargetIf) {
	southbound.TargetGenerator = targetGen
}

func (m *Manager) initializeStores() {
	opts, err := certs.HandleCertPaths(m.Config.CAPath, m.Config.KeyPath, m.Config.CertPath, true)
	if err != nil {
		log.Fatal(err)
	}

	atomixClient := atomix.NewClient(atomix.WithClientID(os.Getenv("POD_NAME")))

	m.LeadershipStore, err = leadership.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load leadership atomix store ", err)
	}

	m.MastershipStore, err = mastership.NewAtomixStore(atomixClient, os.Getenv("POD_NAME"))
	if err != nil {
		log.Fatal("Cannot load mastership atomix store ", err)
	}

	m.DeviceChangesStore, err = device.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load device atomix store ", err)
	}

	m.NetworkChangesStore, err = network.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load network atomix store ", err)
	}

	m.NetworkSnapshotStore, err = networksnap.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load network snapshot atomix store ", err)
	}

	m.DeviceSnapshotStore, err = devicesnap.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load network atomix store ", err)
	}

	m.DeviceStateStore, err = state.NewStore(m.NetworkChangesStore, m.DeviceSnapshotStore)
	if err != nil {
		log.Fatal("Cannot load device store with address %s:", m.Config.TopoAddress, err)
	}
	log.Infof("Topology service connected with endpoint %s", m.Config.TopoAddress)

	m.DeviceCache, err = cache.NewCache(m.NetworkChangesStore, m.DeviceSnapshotStore)
	if err != nil {
		log.Fatal("Cannot load device cache", err)
	}

	m.DeviceStore, err = devicestore.NewTopoStore(m.Config.TopoAddress, opts...)
	if err != nil {
		log.Fatal("Cannot load device store with address %s:", m.Config.TopoAddress, err)
	}
	log.Infof("Topology service connected with endpoint %s", m.Config.TopoAddress)

}

// Run starts a synchronizer based on the devices and the northbound services.
func (m *Manager) Run() {
	log.Info("Starting Manager")

	m.initializeStores()
	log.Info("Stores initialized")
	if err := m.Start(); err != nil {
		log.Fatal("Unable to run Manager", err)
	}

	log.Info("Manager Started")
}

// Start starts the manager
func (m *Manager) Start() error {
	m.networkChangeController = networkchangectl.NewController(m.LeadershipStore, m.DeviceStore, m.NetworkChangesStore, m.DeviceChangesStore)
	m.deviceChangeController = devicechangectl.NewController(m.MastershipStore, m.DeviceStore, m.DeviceChangesStore)
	m.networkSnapshotController = networksnapshotctl.NewController(m.LeadershipStore, m.NetworkChangesStore, m.NetworkSnapshotStore, m.DeviceSnapshotStore, m.DeviceChangesStore)
	m.deviceSnapshotController = devicesnapshotctl.NewController(m.MastershipStore, m.DeviceChangesStore, m.DeviceSnapshotStore)
	m.TopoChannel = make(chan *topodevice.ListResponse, 10)
	m.OperationalStateChannel = make(chan events.OperationalStateEvent)
	m.SouthboundErrorChan = make(chan events.DeviceResponse)
	m.Dispatcher = dispatcher.NewDispatcher()
	m.OperationalStateCache = make(map[topodevice.ID]devicechange.TypedValueMap)
	m.OperationalStateCacheLock = &sync.RWMutex{}
	m.ModelRegistry = m.Config.ModelRegistry

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

	sessionManager, err := synchronizer.NewSessionManager(
		synchronizer.WithTopoChannel(m.TopoChannel),
		synchronizer.WithOpStateChannel(m.OperationalStateChannel),
		synchronizer.WithDispatcher(m.Dispatcher),
		synchronizer.WithModelRegistry(m.ModelRegistry),
		synchronizer.WithOperationalStateCache(m.OperationalStateCache),
		synchronizer.WithNewTargetFn(southbound.TargetGenerator),
		synchronizer.WithOperationalStateCacheLock(m.OperationalStateCacheLock),
		synchronizer.WithDeviceChangeStore(m.DeviceChangesStore),
		synchronizer.WithMastershipStore(m.MastershipStore),
		synchronizer.WithDeviceStore(m.DeviceStore),
		synchronizer.WithSessions(make(map[topodevice.ID]*synchronizer.Session)),
	)

	if err != nil {
		return err
	}

	err = sessionManager.Start()
	if err != nil {
		return err
	}

	return nil
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

// CheckCacheForDevice checks against the device cache (of the device change store
// to see if a device of that name is already present)
func (m *Manager) CheckCacheForDevice(target devicetype.ID, deviceType devicetype.Type,
	version devicetype.Version) (devicetype.Type, devicetype.Version, error) {

	deviceInfos := mgr.DeviceCache.GetDevicesByID(target)
	topoDevice, errTopoDevice := mgr.DeviceStore.Get(topodevice.ID(target))
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
