// Copyright 2022-present Open Networking Foundation.
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

package gnmi

import (
	"context"
	atomixtest "github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/atomix/atomix-go-client/pkg/atomix/test/rsm"
	"github.com/golang/mock/gomock"
	gnmitest "github.com/onosproject/onos-config/pkg/northbound/gnmi/test"
	"github.com/onosproject/onos-config/pkg/store/configuration"
	"github.com/onosproject/onos-config/pkg/store/transaction"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

func Test_GetNoTarget(t *testing.T) {
	server, mctl := createServer(t)
	defer mctl.Finish()

	noTargetPath1 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}
	noTargetPath2 := gnmi.Path{Elem: make([]*gnmi.PathElem, 0)}

	request := gnmi.GetRequest{
		Path: []*gnmi.Path{&noTargetPath1, &noTargetPath2},
	}

	_, err := server.Get(context.TODO(), &request)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "has no target")
}

func createServer(t *testing.T) (*Server, *gomock.Controller) {
	mctl := gomock.NewController(t)
	cfgStore, txStore := testStores(t)
	return &Server{
		mu:             sync.RWMutex{},
		pluginRegistry: gnmitest.NewMockPluginRegistry(mctl),
		topo:           gnmitest.NewMockStore(mctl),
		transactions:   txStore,
		configurations: cfgStore,
	}, mctl
}

func testStores(t *testing.T) (configuration.Store, transaction.Store) {
	test := atomixtest.NewTest(rsm.NewProtocol(), atomixtest.WithReplicas(1), atomixtest.WithPartitions(1))
	assert.NoError(t, test.Start())
	defer test.Stop()

	client1, err := test.NewClient("node-1")
	assert.NoError(t, err)

	cfgStore, err := configuration.NewAtomixStore(client1)
	assert.NoError(t, err)

	txStore, err := transaction.NewAtomixStore(client1)
	assert.NoError(t, err)

	return cfgStore, txStore
}
