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

package mastership

import (
	"testing"

	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-lib-go/pkg/atomix"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
)

func TestMastershipElection(t *testing.T) {
	_, address := atomix.StartLocalNode()

	nodeA := cluster.NodeID("a")
	nodeB := cluster.NodeID("b")
	nodeC := cluster.NodeID("c")

	store1, err := newLocalElection(topodevice.ID("test"), "a", address)
	assert.NoError(t, err)

	store2, err := newLocalElection(topodevice.ID("test"), "b", address)
	assert.NoError(t, err)

	store2Ch := make(chan Mastership)
	err = store2.watch(store2Ch)
	assert.NoError(t, err)

	store3, err := newLocalElection(topodevice.ID("test"), "c", address)
	assert.NoError(t, err)

	store3Ch := make(chan Mastership)
	err = store3.watch(store3Ch)
	assert.NoError(t, err)

	master := store1.getMastership()
	assert.NotNil(t, master)

	assert.Equal(t, master.Master, nodeA)

	master = store2.getMastership()
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, nodeB)

	master = store3.getMastership()
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, nodeC)

	err = store1.Close()
	assert.NoError(t, err)

	mastership := <-store2Ch
	assert.Equal(t, cluster.NodeID("b"), mastership.Master)

	master = store2.getMastership()
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, nodeB)

	mastership = <-store3Ch
	assert.Equal(t, cluster.NodeID("b"), mastership.Master)

	master = store3.getMastership()
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, nodeC)

	err = store2.Close()
	assert.NoError(t, err)

	mastership = <-store3Ch
	assert.Equal(t, cluster.NodeID("c"), mastership.Master)

	master = store3.getMastership()
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, nodeC)

	_ = store3.Close()
}
