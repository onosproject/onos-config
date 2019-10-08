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
	"github.com/onosproject/onos-config/pkg/store/cluster"
	"github.com/onosproject/onos-config/pkg/store/leadership"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestLeadershipActivator(t *testing.T) {
	node1 := cluster.NodeID("node1")
	node2 := cluster.NodeID("node2")

	store1, err := leadership.NewLocalStore("TestLeadershipActivator", node1)
	assert.NoError(t, err)

	store2, err := leadership.NewLocalStore("TestLeadershipActivator", node2)
	assert.NoError(t, err)

	activator1 := &LeadershipActivator{
		Store: store1,
	}

	activator2 := &LeadershipActivator{
		Store: store2,
	}

	ch1 := make(chan bool)
	err = activator1.Start(ch1)
	assert.NoError(t, err)

	ch2 := make(chan bool)
	err = activator2.Start(ch2)
	assert.NoError(t, err)

	assert.True(t, <-ch1)
	assert.False(t, <-ch2)

	ch := make(chan leadership.Leadership)
	err = store2.Watch(ch)
	assert.NoError(t, err)

	err = store1.Close()
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
	assert.True(t, <-ch2)
}
