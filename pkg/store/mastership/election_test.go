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
	"context"
	"github.com/atomix/atomix-go-client/pkg/atomix/test"
	"testing"

	"github.com/onosproject/onos-lib-go/pkg/cluster"
	"github.com/stretchr/testify/assert"
)

func TestMastershipElection(t *testing.T) {
	test := test.NewTest(
		test.WithReplicas(1),
		test.WithPartitions(1),
		test.WithDebugLogs())
	assert.NoError(t, test.Start())
	defer test.Stop()

	client1, err := test.NewClient("node-1")
	assert.NoError(t, err)

	election1, err := client1.GetElection(context.TODO(), "masterships")
	assert.NoError(t, err)

	store1, err := newDeviceMastershipElection("test", election1)
	assert.NoError(t, err)

	client2, err := test.NewClient("node-2")
	assert.NoError(t, err)

	election2, err := client2.GetElection(context.TODO(), "masterships")
	assert.NoError(t, err)

	store2, err := newDeviceMastershipElection("test", election2)
	assert.NoError(t, err)

	store2Ch := make(chan Mastership)
	err = store2.watch(store2Ch)
	assert.NoError(t, err)

	client3, err := test.NewClient("node-3")
	assert.NoError(t, err)

	election3, err := client3.GetElection(context.TODO(), "masterships")
	assert.NoError(t, err)

	store3, err := newDeviceMastershipElection("test", election3)
	assert.NoError(t, err)

	store3Ch := make(chan Mastership)
	err = store3.watch(store3Ch)
	assert.NoError(t, err)

	master := store1.getMastership()
	assert.NotNil(t, master)

	assert.Equal(t, master.Master, cluster.NodeID("node-1"))

	master = store2.getMastership()
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, cluster.NodeID("node-2"))

	master = store3.getMastership()
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, cluster.NodeID("node-3"))

	err = store1.Close()
	assert.NoError(t, err)

	mastership := <-store2Ch
	assert.Equal(t, cluster.NodeID("node-2"), mastership.Master)

	master = store2.getMastership()
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, cluster.NodeID("node-2"))

	mastership = <-store3Ch
	assert.Equal(t, cluster.NodeID("node-2"), mastership.Master)

	master = store3.getMastership()
	assert.NotNil(t, master)
	assert.NotEqual(t, master.Master, cluster.NodeID("node-3"))

	err = store2.Close()
	assert.NoError(t, err)

	mastership = <-store3Ch
	assert.Equal(t, cluster.NodeID("node-3"), mastership.Master)

	master = store3.getMastership()
	assert.NoError(t, err)
	assert.NotNil(t, master)
	assert.Equal(t, master.Master, cluster.NodeID("node-3"))

	_ = store3.Close()
}
