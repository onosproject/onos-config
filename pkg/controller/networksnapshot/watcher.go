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

package networksnapshot

import (
	"github.com/onosproject/onos-config/pkg/controller"
	networksnapshotstore "github.com/onosproject/onos-config/pkg/store/networksnapshot"
	"github.com/onosproject/onos-config/pkg/types"
	networksnapshottype "github.com/onosproject/onos-config/pkg/types/networksnapshot"
	"sync"
)

const queueSize = 100

// Watcher is a network snapshot request watcher
type Watcher struct {
	Store networksnapshotstore.Store
	ch    chan *networksnapshottype.NetworkSnapshot
	mu    sync.Mutex
}

// Start starts the network snapshot request watcher
func (w *Watcher) Start(ch chan<- types.ID) error {
	defer close(ch)

	w.mu.Lock()
	if w.ch != nil {
		return nil
	}

	requestCh := make(chan *networksnapshottype.NetworkSnapshot, queueSize)
	w.ch = requestCh
	w.mu.Unlock()

	if err := w.Store.Watch(requestCh); err != nil {
		return err
	}

	for request := range requestCh {
		ch <- types.ID(request.ID)
	}
	return nil
}

// Stop stops the network snapshot request watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	close(w.ch)
	w.mu.Unlock()
}

var _ controller.Watcher = &Watcher{}
