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
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"testing"
	"time"

	topodevice "github.com/onosproject/onos-config/pkg/device"
	"github.com/onosproject/onos-config/pkg/store/mastership"
	"github.com/onosproject/onos-lib-go/pkg/cluster"
	"github.com/stretchr/testify/assert"
)

type testDeviceResolver struct {
}

func (r testDeviceResolver) Resolve(id controller.ID) (topodevice.ID, error) {
	return topodevice.ID(id.String()), nil
}

func TestMastershipFilter(t *testing.T) {
	test := test.NewTest(
		test.WithReplicas(1),
		test.WithPartitions(1))
	assert.NoError(t, test.Start())
	defer test.Stop()

	node1 := cluster.NodeID("node1")
	node2 := cluster.NodeID("node2")

	atomixClient1, err := test.NewClient(string(node1))
	assert.NoError(t, err)

	atomixClient2, err := test.NewClient(string(node2))
	assert.NoError(t, err)

	store1, err := mastership.NewAtomixStore(atomixClient1, node1)
	assert.NoError(t, err)

	store2, err := mastership.NewAtomixStore(atomixClient2, node2)
	assert.NoError(t, err)

	filter1 := MastershipFilter{store1, testDeviceResolver{}, node1}
	filter2 := MastershipFilter{store2, testDeviceResolver{}, node2}

	device1 := topodevice.ID("device1")
	device2 := topodevice.ID("device2")

	master, err := store1.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, node1)

	master, err = store2.GetMastership(device1)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, node2)

	master, err = store2.GetMastership(device2)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, node2)

	master, err = store1.GetMastership(device2)
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, node1)

	assert.True(t, filter1.Accept(controller.NewID(string(device1))))
	assert.False(t, filter2.Accept(controller.NewID(string(device1))))
	assert.True(t, filter2.Accept(controller.NewID(string(device2))))
	assert.False(t, filter1.Accept(controller.NewID(string(device2))))

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

	assert.True(t, filter2.Accept(controller.NewID(string(device1))))
	assert.True(t, filter2.Accept(controller.NewID(string(device2))))
}
