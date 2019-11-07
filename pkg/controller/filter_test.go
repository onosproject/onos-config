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

package controller

import (
	"github.com/onosproject/onos-config/api/types"
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	topodevice "github.com/onosproject/onos-topo/api/device"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type testDeviceResolver struct {
}

func (r testDeviceResolver) Resolve(id types.ID) (topodevice.ID, error) {
	return topodevice.ID(id), nil
}

func TestMastershipFilter(t *testing.T) {
	node1 := cluster.NodeID("node1")
	node2 := cluster.NodeID("node2")

	store1, err := mastership.NewLocalStore("TestMastershipFilter", node1)
	assert.NoError(t, err)

	store2, err := mastership.NewLocalStore("TestMastershipFilter", node2)
	assert.NoError(t, err)

	filter1 := &MastershipFilter{
		Store:    store1,
		Resolver: testDeviceResolver{},
	}

	filter2 := &MastershipFilter{
		Store:    store2,
		Resolver: testDeviceResolver{},
	}

	device1 := topodevice.ID("device1")
	device2 := topodevice.ID("device2")

	master, err := store1.IsMaster(device1)
	assert.NoError(t, err)
	assert.True(t, master)

	master, err = store2.IsMaster(device1)
	assert.NoError(t, err)
	assert.False(t, master)

	master, err = store2.IsMaster(device2)
	assert.NoError(t, err)
	assert.True(t, master)

	master, err = store1.IsMaster(device2)
	assert.NoError(t, err)
	assert.False(t, master)

	assert.True(t, filter1.Accept(types.ID(device1)))
	assert.False(t, filter2.Accept(types.ID(device1)))
	assert.True(t, filter2.Accept(types.ID(device2)))
	assert.False(t, filter1.Accept(types.ID(device2)))

	ch := make(chan mastership.Mastership)
	err = store2.Watch(device1, ch)
	assert.NoError(t, err)

	err = store1.Close()
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	assert.True(t, filter2.Accept(types.ID(device1)))
	assert.True(t, filter2.Accept(types.ID(device2)))
}
