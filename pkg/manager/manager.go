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
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	"github.com/onosproject/onos-config/pkg/northbound/diags"
	"github.com/onosproject/onos-config/pkg/northbound/gnmi"
	"github.com/onosproject/onos-config/pkg/southbound/synchronizer"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/northbound"

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
}

// Manager single point of entry for the config system.
type Manager struct {
	SouthboundErrorChan     chan events.DeviceResponse
	TopoChannel             chan *topodevice.ListResponse
	OperationalStateChannel chan events.OperationalStateEvent
	Config                  Config
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

// Run starts a synchronizer based on the devices and the northbound services.
func (m *Manager) Run() {
	log.Info("Starting Manager")

	if err := m.Start(); err != nil {
		log.Fatal("Unable to run Manager", err)
	}
}

// Start starts the manager
func (m *Manager) Start() error {
	opts, err := certs.HandleCertPaths(m.Config.CAPath, m.Config.KeyPath, m.Config.CertPath, true)
	if err != nil {
		log.Fatal(err)
	}

	atomixClient := atomix.NewClient(atomix.WithClientID(os.Getenv("POD_NAME")))

	leadershipStore, err := leadership.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load leadership atomix store ", err)
	}

	mastershipStore, err := mastership.NewAtomixStore(atomixClient, os.Getenv("POD_NAME"))
	if err != nil {
		log.Fatal("Cannot load mastership atomix store ", err)
	}

	deviceChangesStore, err := device.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load device atomix store ", err)
	}

	networkChangesStore, err := network.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load network atomix store ", err)
	}

	networkSnapshotStore, err := networksnap.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load network snapshot atomix store ", err)
	}

	deviceSnapshotStore, err := devicesnap.NewAtomixStore(atomixClient)
	if err != nil {
		log.Fatal("Cannot load network atomix store ", err)
	}

	deviceStateStore, err := state.NewStore(networkChangesStore, deviceSnapshotStore)
	if err != nil {
		log.Fatal("Cannot load device store with address %s:", m.Config.TopoAddress, err)
	}
	log.Infof("Topology service connected with endpoint %s", m.Config.TopoAddress)

	deviceCache, err := cache.NewCache(networkChangesStore, deviceSnapshotStore)
	if err != nil {
		log.Fatal("Cannot load device cache", err)
	}

	deviceStore, err := devicestore.NewTopoStore(m.Config.TopoAddress, opts...)
	if err != nil {
		log.Fatal("Cannot load device store with address %s:", m.Config.TopoAddress, err)
	}
	log.Infof("Topology service connected with endpoint %s", m.Config.TopoAddress)

	modelRegistry, err := modelregistry.NewModelRegistry(modelregistry.Config{})
	if err != nil {
		log.Fatal("Failed to load model registry:", err)
	}

	networkChangeController := networkchangectl.NewController(leadershipStore, deviceStore, networkChangesStore, deviceChangesStore)
	deviceChangeController := devicechangectl.NewController(mastershipStore, deviceStore, deviceChangesStore)
	networkSnapshotController := networksnapshotctl.NewController(leadershipStore, networkChangesStore, networkSnapshotStore, deviceSnapshotStore, deviceChangesStore)
	deviceSnapshotController := devicesnapshotctl.NewController(mastershipStore, deviceChangesStore, deviceSnapshotStore)
	m.TopoChannel = make(chan *topodevice.ListResponse, 10)
	m.OperationalStateChannel = make(chan events.OperationalStateEvent)
	m.SouthboundErrorChan = make(chan events.DeviceResponse)
	dsp := dispatcher.NewDispatcher()
	operationalStateCache := make(map[topodevice.ID]devicechange.TypedValueMap)
	operationalStateCacheLock := &sync.RWMutex{}

	// Start the NetworkChange controller
	errNetworkCtrl := networkChangeController.Start()
	if errNetworkCtrl != nil {
		log.Error("Can't start controller ", errNetworkCtrl)
	}
	// Start the DeviceChange controller
	errDeviceChangeCtrl := deviceChangeController.Start()
	if errDeviceChangeCtrl != nil {
		log.Error("Can't start controller ", errDeviceChangeCtrl)
	}
	// Start the NetworkSnapshot controller
	errNetworkSnapshotCtrl := networkSnapshotController.Start()
	if errNetworkSnapshotCtrl != nil {
		log.Error("Can't start controller ", errNetworkSnapshotCtrl)
	}
	// Start the DeviceSnapshot controller
	errDeviceSnapshotCtrl := deviceSnapshotController.Start()
	if errDeviceSnapshotCtrl != nil {
		log.Error("Can't start controller ", errDeviceSnapshotCtrl)
	}

	// Start the main dispatcher system
	go dsp.ListenOperationalState(m.OperationalStateChannel)

	sessionManager, err := synchronizer.NewSessionManager(
		synchronizer.WithTopoChannel(m.TopoChannel),
		synchronizer.WithOpStateChannel(m.OperationalStateChannel),
		synchronizer.WithDispatcher(dsp),
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

	err = sessionManager.Start()
	if err != nil {
		return err
	}

	// Creates gRPC server and registers various services; then serves.
	authorization := false
	if oidcURL := os.Getenv(OIDCServerURL); oidcURL != "" {
		authorization = true
		log.Infof("Authorization enabled. %s=%s", OIDCServerURL, oidcURL)
		// OIDCServerURL is also referenced in jwt.go (from onos-lib-go)
		// and in gNMI Get() where it drives OPA lookup
	} else {
		log.Infof("Authorization not enabled %s", os.Getenv(OIDCServerURL))
	}

	s := northbound.NewServer(northbound.NewServerCfg(mgr.Config.CAPath, mgr.Config.KeyPath, mgr.Config.CertPath,
		int16(mgr.Config.GRPCPort), authorization,
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
		dsp,
		deviceStore,
		&operationalStateCache,
		operationalStateCacheLock)
	s.AddService(diagService)

	gnmiService := gnmi.NewService(modelRegistry,
		deviceChangesStore,
		deviceCache,
		networkChangesStore,
		dsp,
		deviceStore,
		deviceStateStore,
		&operationalStateCache,
		operationalStateCacheLock,
		mgr.Config.AllowUnvalidatedConfig)
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
