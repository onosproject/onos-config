// Copyright 2021-present Open Networking Foundation.
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

package node

import (
	"context"
	"sync"

	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/controller/utils"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/controller"
)

const queueSize = 100

// Watcher is a topology watcher
type Watcher struct {
	topo   topo.Store
	cancel context.CancelFunc
	mu     sync.Mutex
}

// Start starts the topo store watcher
func (w *Watcher) Start(ch chan<- controller.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}

	eventCh := make(chan topoapi.Event, queueSize)
	ctx, cancel := context.WithCancel(context.Background())

	err := w.topo.Watch(ctx, eventCh, nil)
	if err != nil {
		cancel()
		return err
	}
	w.cancel = cancel

	go func() {
		ch <- controller.NewID(utils.GetOnosConfigID())
		for event := range eventCh {
			log.Debugf("Received topo event '%s'", event.Object.ID)
			if entity, ok := event.Object.Obj.(*topoapi.Object_Entity); ok &&
				entity.Entity.KindID == topoapi.ONOS_CONFIG {
				ch <- controller.NewID(event.Object.ID)
			}
		}
		close(ch)
	}()
	return nil
}

// Stop stops the topology watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}
