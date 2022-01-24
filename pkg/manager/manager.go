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
	configurationcontroller "github.com/onosproject/onos-config/pkg/controller/configuration"
	"github.com/onosproject/onos-config/pkg/controller/connection"
	mastershipcontroller "github.com/onosproject/onos-config/pkg/controller/mastership"

	"github.com/onosproject/onos-config/pkg/controller/controlrelation"
	"github.com/onosproject/onos-config/pkg/controller/node"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	gnminb "github.com/onosproject/onos-config/pkg/northbound/gnmi/v2"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	sb "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-config/pkg/store/transaction"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/northbound"

	"os"

	transactioncontroller "github.com/onosproject/onos-config/pkg/controller/transaction"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

// OIDCServerURL - address of an OpenID Connect server
const OIDCServerURL = "OIDC_SERVER_URL"

var log = logging.GetLogger("manager")

// Config is a manager configuration
type Config struct {
	CAPath      string
	KeyPath     string
	CertPath    string
	GRPCPort    int
	TopoAddress string
	Plugins     []string
}

// Manager single point of entry for the config system.
type Manager struct {
	Config         Config
	pluginRegistry *pluginregistry.PluginRegistry
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
func (m *Manager) startNorthboundServer(topo topo.Store, transactionsStore transaction.Store,
	configurationsStore configuration.Store,
	pluginRegistry *pluginregistry.PluginRegistry) error {
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

	adminService := admin.NewService(transactionsStore, configurationsStore, pluginRegistry)
	gnmi := gnminb.NewService(topo, transactionsStore, configurationsStore, pluginRegistry)
	s.AddService(adminService)
	s.AddService(gnmi)

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

// startMastershipController starts mastership controller
func (m *Manager) startMastershipController(topo topo.Store) error {
	mastershipController := mastershipcontroller.NewController(topo)
	return mastershipController.Start()
}
func (m *Manager) startConfigurationController(topo topo.Store, conns sb.ConnManager, configurations configuration.Store, pluginRegistry *pluginregistry.PluginRegistry) error {
	configurationController := configurationcontroller.NewController(topo, conns, configurations, pluginRegistry)
	return configurationController.Start()
}

func (m *Manager) startTransactionController(topo topo.Store, configurations configuration.Store, transactions transaction.Store, pluginRegistry *pluginregistry.PluginRegistry) error {
	transactionController := transactioncontroller.NewController(topo, transactions, configurations, pluginRegistry)
	return transactionController.Start()

}

// Start starts the manager
func (m *Manager) Start() error {
	opts, err := certs.HandleCertPaths(m.Config.CAPath, m.Config.KeyPath, m.Config.CertPath, true)
	if err != nil {
		return err
	}

	atomixClient := atomix.NewClient(atomix.WithClientID(os.Getenv("POD_NAME")))

	// Create new topo store
	topoStore, err := topo.NewStore(m.Config.TopoAddress, opts...)
	if err != nil {
		return err
	}

	// Create the transactions store
	transactions, err := transaction.NewAtomixStore(atomixClient)
	if err != nil {
		return err
	}

	// Create the configurations store
	configurations, err := configuration.NewAtomixStore(atomixClient)
	if err != nil {
		return err
	}

	// Create new plugin registry
	m.pluginRegistry = pluginregistry.NewPluginRegistry(m.Config.Plugins...)
	m.pluginRegistry.Start()

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
	err = m.startConfigurationController(topoStore, conns, configurations, m.pluginRegistry)
	if err != nil {
		return err
	}

	err = m.startTransactionController(topoStore, configurations, transactions, m.pluginRegistry)
	if err != nil {
		return err
	}

	err = m.startMastershipController(topoStore)
	if err != nil {
		return err
	}

	err = m.startNorthboundServer(topoStore, transactions, configurations, m.pluginRegistry)
	if err != nil {
		return err
	}

	log.Info("Manager started")
	return nil
}

// Close kills the manager
func (m *Manager) Close() {
	log.Info("Closing Manager")
	m.pluginRegistry.Stop()
}
