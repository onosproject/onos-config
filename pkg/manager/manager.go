// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package manager is the main coordinator for the ONOS configuration subsystem.
package manager

import (
	"github.com/atomix/go-sdk/pkg/client"
	"github.com/onosproject/onos-config/pkg/controller/connection"
	"github.com/onosproject/onos-config/pkg/controller/target"
	configurationcontroller "github.com/onosproject/onos-config/pkg/controller/v2/configuration"
	mastershipcontroller "github.com/onosproject/onos-config/pkg/controller/v2/mastership"
	proposalcontroller "github.com/onosproject/onos-config/pkg/controller/v2/proposal"
	"github.com/onosproject/onos-config/pkg/store/v2/proposal"

	"github.com/onosproject/onos-config/pkg/controller/node"
	"github.com/onosproject/onos-config/pkg/northbound/admin"
	gnminb "github.com/onosproject/onos-config/pkg/northbound/gnmi/v2"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	sb "github.com/onosproject/onos-config/pkg/southbound/gnmi"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-config/pkg/store/v2/configuration"
	"github.com/onosproject/onos-config/pkg/store/v2/transaction"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/northbound"

	"os"

	transactioncontroller "github.com/onosproject/onos-config/pkg/controller/v2/transaction"
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
	pluginRegistry pluginregistry.PluginRegistry
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
	topo topo.Store,
	transactionsStore transaction.Store,
	proposalsStore proposal.Store,
	configurationsStore configuration.Store,
	pluginRegistry pluginregistry.PluginRegistry, conns sb.ConnManager) error {
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
	gnmi := gnminb.NewService(topo, transactionsStore, proposalsStore, configurationsStore, pluginRegistry, conns)
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

// startTargetController starts target controller
func (m *Manager) startTargetController(topo topo.Store, conns sb.ConnManager) error {
	targetController := target.NewController(topo, conns)
	return targetController.Start()
}

// startMastershipController starts mastership controller
func (m *Manager) startMastershipController(topo topo.Store, configurations configuration.Store) error {
	mastershipController := mastershipcontroller.NewController(topo, configurations)
	return mastershipController.Start()
}

func (m *Manager) startConfigurationController(topo topo.Store, conns sb.ConnManager, configurations configuration.Store) error {
	configurationController := configurationcontroller.NewController(topo, conns, configurations)
	return configurationController.Start()
}

func (m *Manager) startProposalController(topo topo.Store, conns sb.ConnManager, proposals proposal.Store, configurations configuration.Store, pluginRegistry pluginregistry.PluginRegistry) error {
	proposalController := proposalcontroller.NewController(topo, conns, proposals, configurations, pluginRegistry)
	return proposalController.Start()
}

func (m *Manager) startTransactionController(transactions transaction.Store, proposals proposal.Store) error {
	transactionController := transactioncontroller.NewController(transactions, proposals)
	return transactionController.Start()
}

// Start starts the manager
func (m *Manager) Start() error {
	opts, err := certs.HandleCertPaths(m.Config.CAPath, m.Config.KeyPath, m.Config.CertPath, true)
	if err != nil {
		return err
	}

	atomixClient := client.NewClient()

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

	// Create the proposals store
	proposals, err := proposal.NewAtomixStore(atomixClient)
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

	err = m.startTargetController(topoStore, conns)
	if err != nil {
		return err
	}

	err = m.startConfigurationController(topoStore, conns, configurations)
	if err != nil {
		return err
	}

	err = m.startProposalController(topoStore, conns, proposals, configurations, m.pluginRegistry)
	if err != nil {
		return err
	}

	err = m.startTransactionController(transactions, proposals)
	if err != nil {
		return err
	}

	err = m.startMastershipController(topoStore, configurations)
	if err != nil {
		return err
	}

	err = m.startNorthboundServer(topoStore, transactions, proposals, configurations, m.pluginRegistry, conns)
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
