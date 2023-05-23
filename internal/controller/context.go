package controller

import (
	"github.com/golang/mock/gomock"
	configapi "github.com/onosproject/onos-api/go/onos/config/v2"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	gnmiinternal "github.com/onosproject/onos-config/internal/southbound/gnmi"
	configurationstoreinternal "github.com/onosproject/onos-config/internal/store/configuration"
	proposalstoreinternal "github.com/onosproject/onos-config/internal/store/proposal"
	topostoreinternal "github.com/onosproject/onos-config/internal/store/topo"
	transactionstoreinternal "github.com/onosproject/onos-config/internal/store/transaction"
	"github.com/onosproject/onos-config/pkg/pluginregistry"
	"github.com/onosproject/onos-config/pkg/southbound/gnmi"
	configurationstore "github.com/onosproject/onos-config/pkg/store/configuration"
	proposalstore "github.com/onosproject/onos-config/pkg/store/proposal"
	topostore "github.com/onosproject/onos-config/pkg/store/topo"
	transactionstore "github.com/onosproject/onos-config/pkg/store/transaction"
	"testing"
)

func newTestContext(t *testing.T, test test.TestCase) *TestContext {
	ctrl := gomock.NewController(t)
	context := &TestContext{
		ctrl:           ctrl,
		Conns:          gnmiinternal.NewMockConnManager(ctrl),
		Topo:           topostoreinternal.NewMockStore(ctrl),
		Plugins:        pluginregistry.NewMockPluginRegistry(ctrl),
		Configurations: configurationstoreinternal.NewMockStore(ctrl),
		Proposals:      proposalstoreinternal.NewMockStore(ctrl),
		Transactions:   transactionstoreinternal.NewMockStore(ctrl),
	}
	context.init(test)
	return context
}

type TestContext struct {
	ctrl           *gomock.Controller
	NodeID         topoapi.ID
	Index          configapi.Index
	Conns          gnmi.ConnManager
	Topo           topostore.Store
	Plugins        pluginregistry.PluginRegistry
	Configurations configurationstore.Store
	Proposals      proposalstore.Store
	Transactions   transactionstore.Store
}

func (c *TestContext) Finish() {
	c.ctrl.Finish()
}

func (c *TestContext) init(test test.TestCase) {
	c.NodeID = topoapi.ID(test.Context.Node)
	c.Index = configapi.Index(test.Context.Index)
	c.initConns(test)
	c.initTopo(test)
	c.initConfigurations(test)
	c.initProposals(test)
	c.initTransactions(test)
}

func (c *TestContext) initConns(test test.TestCase) {

}

func (c *TestContext) initTopo(test test.TestCase) {

}

func (c *TestContext) initConfigurations(test test.TestCase) {

}

func (c *TestContext) initProposals(test test.TestCase) {

}

func (c *TestContext) initTransactions(test test.TestCase) {

}
