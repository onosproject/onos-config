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
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/utils"
	devicetopo "github.com/onosproject/onos-topo/pkg/northbound/device"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMastershipStore(t *testing.T) {
	node, conn := utils.StartLocalNode()

	node1 := cluster.NodeID("node1")
	node2 := cluster.NodeID("node2")
	node3 := cluster.NodeID("node3")

	device1 := devicetopo.ID("device1")
	device2 := devicetopo.ID("device2")

	// Create three stores for three different nodes
	store1, err := newLocalStore(node1, conn)
	assert.NoError(t, err)

	store2, err := newLocalStore(node2, conn)
	assert.NoError(t, err)

	store3, err := newLocalStore(node3, conn)
	assert.NoError(t, err)

	// Verify that the first node that checks mastership for a device wins the election
	// and no other node believes itself to be the master
	master, err := store1.IsMaster(device1)
	assert.NoError(t, err)
	assert.True(t, master)

	master, err = store2.IsMaster(device1)
	assert.NoError(t, err)
	assert.False(t, master)

	master, err = store3.IsMaster(device1)
	assert.NoError(t, err)
	assert.False(t, master)

	// Verify that listening for events for a device enters a node into the device mastership election
	store2Ch2 := make(chan Mastership)
	err = store2.Watch(device2, store2Ch2)
	assert.NoError(t, err)

	// Verify that the watching node is the master
	master, err = store2.IsMaster(device2)
	assert.NoError(t, err)
	assert.True(t, master)

	master, err = store1.IsMaster(device2)
	assert.NoError(t, err)
	assert.False(t, master)

	master, err = store3.IsMaster(device2)
	assert.NoError(t, err)
	assert.False(t, master)

	// Watch device2 mastership on an additional node and verify that it does not cause a mastership change
	store3Ch2 := make(chan Mastership)
	err = store3.Watch(device2, store3Ch2)
	assert.NoError(t, err)

	master, err = store3.IsMaster(device1)
	assert.NoError(t, err)
	assert.False(t, master)

	// Listen for device1 events on remaining nodes
	store2Ch1 := make(chan Mastership)
	err = store2.Watch(device1, store2Ch1)
	assert.NoError(t, err)
	store3Ch1 := make(chan Mastership)
	err = store3.Watch(device1, store3Ch1)
	assert.NoError(t, err)

	// Close node 1 (the master for device 1) and verify a mastership change occurs on nodes 2 and 3
	err = store1.Close()
	assert.NoError(t, err)

	// node2 should now be the master for device1
	mastership := <-store2Ch1
	assert.Equal(t, device1, mastership.Device)
	assert.Equal(t, node2, mastership.Master)

	master, err = store2.IsMaster(device1)
	assert.NoError(t, err)
	assert.True(t, master)

	// node3 should not be the master for device1
	mastership = <-store3Ch1
	assert.Equal(t, device1, mastership.Device)
	assert.Equal(t, node2, mastership.Master)

	master, err = store3.IsMaster(device1)
	assert.NoError(t, err)
	assert.False(t, master)

	// Close node2 and verify mastership for both devices changes
	err = store2.Close()
	assert.NoError(t, err)

	// node3 should now be the master for device1
	mastership = <-store3Ch1
	assert.Equal(t, device1, mastership.Device)
	assert.Equal(t, node3, mastership.Master)

	master, err = store3.IsMaster(device1)
	assert.NoError(t, err)
	assert.True(t, master)

	// node3 should also be the master for device2
	mastership = <-store3Ch2
	assert.Equal(t, device2, mastership.Device)
	assert.Equal(t, node3, mastership.Master)

	master, err = store3.IsMaster(device2)
	assert.NoError(t, err)
	assert.True(t, master)

	_ = store3.Close()
	_ = conn.Close()
	_ = node.Stop()
}
