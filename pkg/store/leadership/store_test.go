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

package leadership

import (
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/utils"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLeadershipStore(t *testing.T) {
	node, address := utils.StartLocalNode()

	store1, err := newLocalStore("a", address)
	assert.NoError(t, err)

	store2, err := newLocalStore("b", address)
	assert.NoError(t, err)

	store2Ch := make(chan Leadership)
	err = store2.Watch(store2Ch)
	assert.NoError(t, err)

	store3, err := newLocalStore("c", address)
	assert.NoError(t, err)

	store3Ch := make(chan Leadership)
	err = store3.Watch(store3Ch)
	assert.NoError(t, err)

	leader, err := store1.IsLeader()
	assert.NoError(t, err)
	assert.True(t, leader)

	leader, err = store2.IsLeader()
	assert.NoError(t, err)
	assert.False(t, leader)

	leader, err = store3.IsLeader()
	assert.NoError(t, err)
	assert.False(t, leader)

	err = store1.Close()
	assert.NoError(t, err)

	leadership := <-store2Ch
	assert.Equal(t, cluster.NodeID("b"), leadership.Leader)

	leader, err = store2.IsLeader()
	assert.NoError(t, err)
	assert.True(t, leader)

	leadership = <-store3Ch
	assert.Equal(t, cluster.NodeID("b"), leadership.Leader)

	leader, err = store3.IsLeader()
	assert.NoError(t, err)
	assert.False(t, leader)

	err = store2.Close()
	assert.NoError(t, err)

	leadership = <-store3Ch
	assert.Equal(t, cluster.NodeID("c"), leadership.Leader)

	leader, err = store3.IsLeader()
	assert.NoError(t, err)
	assert.True(t, leader)

	_ = store3.Close()
	_ = node.Stop()
}
