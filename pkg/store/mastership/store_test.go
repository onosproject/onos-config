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

func TestMastershipStore(t *testing.T) {
	_, address := atomix.StartLocalNode()

	node1 := cluster.NodeID("node1")
	node2 := cluster.NodeID("node2")
	node3 := cluster.NodeID("node3")

	device1 := topodevice.ID("device1")
	device2 := topodevice.ID("device2")

	// Create three stores for three different nodes
	store1, err := newLocalStore(node1, address)
	assert.NoError(t, err)

	store2, err := newLocalStore(node2, address)
	assert.NoError(t, err)

	store3, err := newLocalStore(node3, address)
	assert.NoError(t, err)

	// Verify that the first node that checks mastership for a device wins the election
	// and no other node believes itself to be the master
	master, err := store1.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, node1)
	// Verify the master term for the given device is correct
	// Since the election is performed once for device1 then the term will be 1
	assert.Equal(t, master.Term, Term(1))

	master, err = store2.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, node2)

	master, err = store3.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, node3)

	// Verify that listening for events for a device enters a node into the device mastership election
	store2Ch2 := make(chan Mastership)
	err = store2.Watch(device2, store2Ch2)
	assert.NoError(t, err)

	// Verify that the watching node is the master
	master, err = store2.GetMastership(device2)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, node2)

	// Verify the master term for the given device is correct
	// Since the election is performed once for device2 then the term will be 1
	assert.Equal(t, master.Term, Term(1))

	master, err = store1.GetMastership(device2)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, node1)

	master, err = store3.GetMastership(device2)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, node3)

	// Watch device2 mastership on an additional node and verify that it does not cause a mastership change
	store3Ch2 := make(chan Mastership)
	err = store3.Watch(device2, store3Ch2)
	assert.NoError(t, err)

	master, err = store3.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, node3)

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

	master, err = store2.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, node2)

	// Verify the master term for the given device is correct
	// Since the master of device1 is changed then its term has been increased by 1
	assert.Equal(t, master.Term, Term(2))

	// node3 should not be the master for device1
	mastership = <-store3Ch1
	assert.Equal(t, device1, mastership.Device)
	assert.Equal(t, node2, mastership.Master)

	master, err = store3.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, node3)

	// Close node2 and verify mastership for both devices changes
	err = store2.Close()
	assert.NoError(t, err)

	// node3 should now be the master for device1
	mastership = <-store3Ch1
	assert.Equal(t, device1, mastership.Device)
	assert.Equal(t, node3, mastership.Master)

	master, err = store3.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, node3)

	// Verify the master term for the given device is correct
	// Since master of device1 has been changed its term has been increased by 1
	assert.Equal(t, master.Term, Term(3))

	// node3 should also be the master for device2
	mastership = <-store3Ch2
	assert.Equal(t, device2, mastership.Device)
	assert.Equal(t, node3, mastership.Master)

	master, err = store3.GetMastership(device2)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, node3)
	// Since master of device2 has been changed its term has been increased by 1
	assert.Equal(t, master.Term, Term(2))

	_ = store3.Close()
}
