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

package device

import (
	"github.com/onosproject/onos-config/api/types"
	devicesnapshot "github.com/onosproject/onos-config/api/types/snapshot/device"
	"github.com/onosproject/onos-config/pkg/controller"
	devicesnapstore "github.com/onosproject/onos-config/pkg/store/snapshot/device"
	"github.com/onosproject/onos-config/pkg/store/stream"
	"sync"
)

const queueSize = 100

// Watcher is a device snapshot watcher
type Watcher struct {
	Store devicesnapstore.Store
	ctx   stream.Context
	mu    sync.Mutex
}

// Start starts the device snapshot watcher
func (w *Watcher) Start(ch chan<- types.ID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.ctx != nil {
		return nil
	}

	snapshotCh := make(chan stream.Event, queueSize)

	ctx, err := w.Store.Watch(snapshotCh)
	if err != nil {
		return err
	}
	w.ctx = ctx

	go func() {
		for request := range snapshotCh {
			ch <- types.ID(request.Object.(*devicesnapshot.DeviceSnapshot).ID)
		}
		close(ch)
	}()
	return nil
}

// Stop stops the device snapshot watcher
func (w *Watcher) Stop() {
	w.mu.Lock()
	if w.ctx != nil {
		w.ctx.Close()
	}
	w.mu.Unlock()
}

var _ controller.Watcher = &Watcher{}
