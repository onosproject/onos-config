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
	"github.com/atomix/atomix-go-client/pkg/atomix"
	"github.com/onosproject/onos-config/pkg/controller/connection"
	"github.com/onosproject/onos-config/pkg/controller/controlrelation"
	"github.com/onosproject/onos-config/pkg/controller/node"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	sb "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/onosproject/onos-config/pkg/pluginregistry"

	"os"
	"sync"

	devicechange "github.com/onosproject/onos-api/go/onos/config/change/device"
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
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

// OIDCServerURL - address of an OpenID Connect server
const OIDCServerURL = "OIDC_SERVER_URL"

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath                 string
	KeyPath                string
	CertPath               string
	GRPCPort               int
	TopoAddress            string
	AllowUnvalidatedConfig bool
	PluginPorts			   []uint
}

// Manager single point of entry for the config system.
type Manager struct {
	Config Config
}

// NewManager initializes the network config manager subsystem.
func NewManager(cfg Config) *Manager {
	log.Info("Creating Manager")

	mgr := Manager{
		Config: cfg,
	}
	return &mgr
}

// Run starts a synchronizer based on the devices and the northbound services.
func (m *Manager) Run() {
	log.Info("Starting Manager")

	if err := m.Start(); err != nil {
		log.Fatal("Unable to run Manager", err)
	}
}

// Creates gRPC server and registers various services; then serves.
func (m *Manager) startNorthboundServer(
	deviceChangesStore device.Store,
	modelRegistry *modelregistry.ModelRegistry,
	deviceCache cache.Cache,
	networkChangesStore network.Store,
	deviceStore devicestore.Store,
	dispatcherInstance *dispatcher.Dispatcher,
	deviceStateStore state.Store,
	operationalStateCache *map[topodevice.ID]devicechange.TypedValueMap,
	operationalStateCacheLock *sync.RWMutex,
	networkSnapshotStore networksnap.Store, deviceSnapshotStore devicesnap.Store) error {
	authorization := false
	if oidcURL := os.Getenv(OIDCServerURL); oidcURL != "" {
		authorization = true
		log.Infof("Authorization enabled. %s=%s", OIDCServerURL, oidcURL)
		// OIDCServerURL is also referenced in jwt.go (from onos-lib-go)
		// and in gNMI Get() where it drives OPA lookup
	} else {
		log.Infof("Authorization not enabled %s", os.Getenv(OIDCServerURL))
	}

	s := northbound.NewServer(northbound.NewServerCfg(m.Config.CAPath, m.Config.KeyPath, m.Config.CertPath,
		int16(m.Config.GRPCPort), authorization,
		northbound.SecurityConfig{
			AuthenticationEnabled: authorization,
			AuthorizationEnabled:  authorization,
		}))

	s.AddService(logging.Service{})

	adminService := admin.NewService(networkChangesStore, networkSnapshotStore, deviceSnapshotStore)
	s.AddService(adminService)

	diagService := diags.NewService(deviceChangesStore,
		deviceCache,
		networkChangesStore,
		dispatcherInstance,
		deviceStore,
		operationalStateCache,
		operationalStateCacheLock)
	s.AddService(diagService)

	gnmiService := gnmi.NewService(
		modelRegistry,
		deviceChangesStore,
		deviceCache,
		networkChangesStore,
		dispatcherInstance,
		deviceStore,
		deviceStateStore,
		operationalStateCache,
		operationalStateCacheLock,
		m.Config.AllowUnvalidatedConfig,
	)
	s.AddService(gnmiService)

	doneCh := make(chan error)
	go func() {
		err := s.Serve(func(started string) {
			log.Info("Started NBI on ", started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

// Start the NetworkChange controller
func (m *Manager) startNetworkChangeController(leadershipStore leadership.Store, deviceStore devicestore.Store, networkChangesStore network.Store, deviceChangesStore device.Store) error {
	networkChangeController := networkchangectl.NewController(leadershipStore, deviceStore, networkChangesStore, deviceChangesStore)
	return networkChangeController.Start()
}

// Start the DeviceChange controller
func (m *Manager) startDeviceChangeController(mastershipStore mastership.Store, deviceStore devicestore.Store, deviceChangesStore device.Store) error {
	deviceChangeController := devicechangectl.NewController(mastershipStore, deviceStore, deviceChangesStore)
	return deviceChangeController.Start()
}

// Start the NetworkSnapshot controller
func (m *Manager) startNetworkSnapshotController(leadershipStore leadership.Store, networkChangesStore network.Store, networkSnapshotStore networksnap.Store,
	deviceSnapshotStore devicesnap.Store, deviceChangesStore device.Store) error {
	networkSnapshotController := networksnapshotctl.NewController(leadershipStore, networkChangesStore, networkSnapshotStore, deviceSnapshotStore, deviceChangesStore)
	return networkSnapshotController.Start()
}

// Start the DeviceSnapshot controller
func (m *Manager) startDeviceSnapshotController(mastershipStore mastership.Store, deviceChangesStore device.Store, deviceSnapshotStore devicesnap.Store) error {
	deviceSnapshotController := devicesnapshotctl.NewController(mastershipStore, deviceChangesStore, deviceSnapshotStore)
	return deviceSnapshotController.Start()
}

// startNodeController starts node controller
func (m *Manager) startNodeController(topo topo.Store) error {
	nodeController := node.NewController(topo)
	return nodeController.Start()
}

// startConnController starts connection controller
func (m *Manager) startConnController(topo topo.Store, conns sb.ConnManager) error {
	connController := connection.NewController(topo, conns)
	return connController.Start()
}

// startControlRelationController starts control relation controller
func (m *Manager) startControlRelationController(topo topo.Store, conns sb.ConnManager) error {
	controlRelationController := controlrelation.NewController(topo, conns)
	return controlRelationController.Start()
}

// Start the main dispatcher system
func (m *Manager) startDispatcherSystem(
	dispatcherInstance *dispatcher.Dispatcher,
	deviceChangesStore device.Store,
	modelRegistry *modelregistry.ModelRegistry,
	deviceStore devicestore.Store,
	operationalStateCache map[topodevice.ID]devicechange.TypedValueMap,
	operationalStateCacheLock *sync.RWMutex,
	mastershipStore mastership.Store) error {

	topoChannel := make(chan *topodevice.ListResponse, 10)
	operationalStateChannel := make(chan events.OperationalStateEvent)

	go dispatcherInstance.ListenOperationalState(operationalStateChannel)

	sessionManager, err := synchronizer.NewSessionManager(
		synchronizer.WithTopoChannel(topoChannel),
		synchronizer.WithOpStateChannel(operationalStateChannel),
		synchronizer.WithDispatcher(dispatcherInstance),
		synchronizer.WithModelRegistry(modelRegistry),
		synchronizer.WithOperationalStateCache(operationalStateCache),
		synchronizer.WithNewTargetFn(southbound.TargetGenerator),
		synchronizer.WithOperationalStateCacheLock(operationalStateCacheLock),
		synchronizer.WithDeviceChangeStore(deviceChangesStore),
		synchronizer.WithMastershipStore(mastershipStore),
		synchronizer.WithDeviceStore(deviceStore),
		synchronizer.WithSessions(make(map[topodevice.ID]*synchronizer.Session)),
	)
	if err != nil {
		return err
	}

	return sessionManager.Start()
}

// Start starts the manager
func (m *Manager) Start() error {
	opts, err := certs.HandleCertPaths(m.Config.CAPath, m.Config.KeyPath, m.Config.CertPath, true)
	if err != nil {
		return err
	}

	atomixClient := atomix.NewClient(atomix.WithClientID(os.Getenv("POD_NAME")))

	topoStore, err := topo.NewStore(m.Config.TopoAddress, opts...)
	if err != nil {
		return err
	}

	conns := sb.NewConnManager()
	err = m.startNodeController(topoStore)
	if err != nil {
		return err
	}

	err = m.startConnController(topoStore, conns)
	if err != nil {
		return err
	}

	err = m.startControlRelationController(topoStore, conns)
	if err != nil {
		return err
	}

	leadershipStore, err := leadership.NewAtomixStore(atomixClient)
	if err != nil {
		return err
	}

	mastershipStore, err := mastership.NewAtomixStore(atomixClient, os.Getenv("POD_NAME"))
	if err != nil {
		return err
	}

	deviceChangesStore, err := device.NewAtomixStore(atomixClient)
	if err != nil {
		return err
	}

	networkChangesStore, err := network.NewAtomixStore(atomixClient)
	if err != nil {
		return err
	}

	networkSnapshotStore, err := networksnap.NewAtomixStore(atomixClient)
	if err != nil {
		return err
	}

	deviceSnapshotStore, err := devicesnap.NewAtomixStore(atomixClient)
	if err != nil {
		return err
	}

	deviceStateStore, err := state.NewStore(networkChangesStore, deviceSnapshotStore)
	if err != nil {
		return err
	}

	deviceCache, err := cache.NewCache(networkChangesStore, deviceSnapshotStore)
	if err != nil {
		return err
	}

	deviceStore, err := devicestore.NewTopoStore(m.Config.TopoAddress, opts...)
	if err != nil {
		return err
	}
	log.Infof("Topology service connected with endpoint %s", m.Config.TopoAddress)

	modelRegistry, err := modelregistry.NewModelRegistry(modelregistry.Config{})
	if err != nil {
		return err
	}

	_ = pluginregistry.NewPluginRegistry(m.Config.PluginPorts...)

	dispatcherInstance := dispatcher.NewDispatcher()
	operationalStateCache := make(map[topodevice.ID]devicechange.TypedValueMap)
	operationalStateCacheLock := &sync.RWMutex{}

	// Start the NetworkChange controller
	err = m.startNetworkChangeController(leadershipStore, deviceStore, networkChangesStore, deviceChangesStore)
	if err != nil {
		return err
	}

	// Start the DeviceChange controller
	err = m.startDeviceChangeController(mastershipStore, deviceStore, deviceChangesStore)
	if err != nil {
		return err
	}

	// Start the NetworkSnapshot controller
	err = m.startNetworkSnapshotController(leadershipStore, networkChangesStore, networkSnapshotStore, deviceSnapshotStore, deviceChangesStore)
	if err != nil {
		return err
	}

	// Start the DeviceSnapshot controller
	err = m.startDeviceSnapshotController(mastershipStore, deviceChangesStore, deviceSnapshotStore)
	if err != nil {
		return err
	}

	// Start the main dispatcher system
	err = m.startDispatcherSystem(
		dispatcherInstance,
		deviceChangesStore,
		modelRegistry,
		deviceStore,
		operationalStateCache,
		operationalStateCacheLock,
		mastershipStore)
	if err != nil {
		return err
	}

	// Start the northbound server
	err = m.startNorthboundServer(deviceChangesStore, modelRegistry, deviceCache, networkChangesStore, deviceStore, dispatcherInstance,
		deviceStateStore, &operationalStateCache, operationalStateCacheLock, networkSnapshotStore, deviceSnapshotStore)
	if err != nil {
		return err
	}

	return nil
}

// Close kills the manager
func (m *Manager) Close() {
	log.Info("Closing Manager")
}
