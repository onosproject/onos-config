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

package network

import (
	"github.com/onosproject/onos-api/go/onos/config/snapshot"
	networksnapshot "github.com/onosproject/onos-api/go/onos/config/snapshot/network"
	networksnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/network"
	"github.com/onosproject/onos-lib-go/pkg/controller"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNetworkSnapshotWatcher(t *testing.T) {
	store, err := networksnapstore.NewLocalStore()
	assert.NoError(t, err)
	defer store.Close()

	watcher := &Watcher{
		Store: store,
	}

	ch := make(chan controller.ID)
	err = watcher.Start(ch)
	assert.NoError(t, err)

	snapshot1 := &networksnapshot.NetworkSnapshot{}

	err = store.Create(snapshot1)
	assert.NoError(t, err)

	select {
	case id := <-ch:
		assert.Equal(t, string(snapshot1.ID), id.String())
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	snapshot2 := &networksnapshot.NetworkSnapshot{}

	err = store.Create(snapshot2)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}

	snapshot1.Status.State = snapshot.State_RUNNING
	err = store.Update(snapshot1)
	assert.NoError(t, err)

	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.FailNow()
	}
}
